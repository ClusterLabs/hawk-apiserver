package main

import (
	"flag"
	"fmt"
	"github.com/krig/go-pacemaker"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

// AsyncCib
//
// Wraps the CIB retrieval from go-pacemaker
// in an asynchronous interface, so that
// other parts of the server have a single
// copy of the CIB available at any time.
// Also provides a subscription interface
// for the long polling request end point,
// via Wait().

type AsyncCib struct {
	xmldoc   string
	version  *pacemaker.CibVersion
	lock     sync.Mutex
	notifier chan chan string
}

func (acib *AsyncCib) Start() {
	if acib.notifier == nil {
		acib.notifier = make(chan chan string)
	}
	cibFetcher := func() {
		for {
			cib, err := pacemaker.OpenCib()
			if err != nil {
				log.Warnf("Failed to connect to Pacemaker: %s", err)
				time.Sleep(5 * time.Second)
			}
			for cib != nil {
				func() {
					cibxml, err := cib.Query()
					if err != nil {
						log.Errorf("Failed to query CIB: %s", err)
					} else {
						acib.notifyNewCib(cibxml)
					}
				}()

				waiter := make(chan int)
				_, err = cib.Subscribe(func(event pacemaker.CibEvent, doc *pacemaker.CibDocument) {
					if event == pacemaker.UpdateEvent {
						acib.notifyNewCib(doc)
					} else {
						log.Warnf("lost connection: %s\n", event)
						waiter <- 1
					}
				})
				if err != nil {
					log.Infof("Failed to subscribe, rechecking every 5 seconds")
					time.Sleep(5 * time.Second)
				} else {
					<-waiter
				}
			}
		}
	}

	go cibFetcher()
	go pacemaker.Mainloop()
}

func (acib *AsyncCib) Wait(timeout int, defval string) string {
	requestChan := make(chan string)
	select {
	case acib.notifier <- requestChan:
	case <-time.After(time.Duration(timeout) * time.Second):
		return defval
	}
	return <-requestChan
}

func (acib *AsyncCib) Get() string {
	acib.lock.Lock()
	defer acib.lock.Unlock()
	return acib.xmldoc
}

func (acib *AsyncCib) Version() *pacemaker.CibVersion {
	acib.lock.Lock()
	defer acib.lock.Unlock()
	return acib.version
}

func (acib *AsyncCib) notifyNewCib(cibxml *pacemaker.CibDocument) {
	text := cibxml.ToString()
	version := cibxml.Version()
	log.Infof("[CIB]: %v", version)
	acib.lock.Lock()
	acib.xmldoc = text
	acib.version = version
	acib.lock.Unlock()
	// Notify anyone waiting
Loop:
	for {
		select {
		case clientchan := <-acib.notifier:
			clientchan <- version.String()
		default:
			break Loop
		}
	}
}

type Config struct {
	Listen   string        `json:"listen"`
	Port     int           `json:"port"`
	Key      string        `json:"key"`
	Cert     string        `json:"cert"`
	LogLevel string        `json:"loglevel"`
	Route    []ConfigRoute `json:"route"`
}

type ConfigRoute struct {
	Handler string  `json:"handler"`
	Path    string  `json:"path"`
	Target  *string `json:"target"`
}

type routeHandler struct {
	cib      AsyncCib
	config   *Config
	proxies  map[*ConfigRoute]*ReverseProxy
	proxymux sync.Mutex
}

func NewRouteHandler(config *Config) *routeHandler {
	return &routeHandler{
		config:  config,
		proxies: make(map[*ConfigRoute]*ReverseProxy),
	}
}

func (handler *routeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range handler.config.Route {
		if !strings.HasPrefix(r.URL.Path, route.Path) {
			continue
		}
		if route.Handler == "api/v1" {
			if handler.serveAPI(w, r, &route) {
				return
			}
		} else if route.Handler == "monitor" {
			if handler.serveMonitor(w, r, &route) {
				return
			}
		} else if route.Handler == "file" && route.Target != nil {
			if handler.serveFile(w, r, &route) {
				return
			}
		} else if route.Handler == "proxy" && route.Target != nil {
			if handler.serveProxy(w, r, &route) {
				return
			}
		}
	}
	http.Error(w, fmt.Sprintf("Unmatched request: %v.", r.URL.Path), 500)
	return
}

func (handler *routeHandler) proxyForRoute(route *ConfigRoute) *ReverseProxy {
	handler.proxymux.Lock()
	proxy, ok := handler.proxies[route]
	handler.proxymux.Unlock()
	if ok {
		return proxy
	}

	url, err := url.Parse(*route.Target)
	if err != nil {
		log.Error(err)
		return nil
	}
	proxy = NewSingleHostReverseProxy(url, "", http.DefaultMaxIdleConnsPerHost)
	handler.proxymux.Lock()
	handler.proxies[route] = proxy
	handler.proxymux.Unlock()
	return proxy
}

func (handler *routeHandler) serveAPI(w http.ResponseWriter, r *http.Request, route *ConfigRoute) bool {
	log.Debugf("[api/v1] %v", r.URL.Path)
	if !checkHawkAuthMethods(r) {
		http.Error(w, "Unauthorized request.", 401)
		return true
	}
	if r.Method == "GET" {
		prefix := route.Path + "/configuration/"
		// all types below cib/configuration
		all_types := "(nodes|resources|cluster|constraints|rsc_defaults|op_defaults|alerts|tags|acls|fencing)"
		match, _ := regexp.MatchString(prefix + all_types + "(/?|/.+/?)$", r.URL.Path)
		if match {
			return handleConfigApi(w, r, handler.cib.Get())
		}
		if strings.HasPrefix(r.URL.Path, prefix + "cib.xml") {
			xmldoc := handler.cib.Get()
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, xmldoc)
			return true
		}
	}
	http.Error(w, fmt.Sprintf("[api/v1]: No route for %v.", r.URL.Path), 500)
	return true
}

func (handler *routeHandler) serveMonitor(w http.ResponseWriter, r *http.Request, route *ConfigRoute) bool {
	if r.URL.Path != route.Path && r.URL.Path != fmt.Sprintf("%s.json", route.Path) {
		return false
	}
	log.Debugf("[monitor] %v", r.URL.Path)

	epoch := ""
	args := strings.Split(r.URL.RawQuery, "&")
	if len(args) >= 1 {
		epoch = args[0]
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if r.Header.Get("Origin") != "" {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-CSRF-Token, Token")
		w.Header().Set("Access-Control-Max-Age", "1728000")
	}
	// Flush headers if possible
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	new_epoch := ""
	ver := handler.cib.Version()
	if ver != nil {
		new_epoch = ver.String()
	}
	if new_epoch == "" || new_epoch == epoch {
		// either we haven't managed to connect
		// to the CIB yet, or there hasn't been
		// any change since we asked last.
		// Wait with a timeout for something to
		// appear, and return whatever we had
		// if we time out
		new_epoch = handler.cib.Wait(60, new_epoch)
	}
	io.WriteString(w, fmt.Sprintf("{\"epoch\":\"%s\"}\n", new_epoch))
	return true
}

func (handler *routeHandler) serveFile(w http.ResponseWriter, r *http.Request, route *ConfigRoute) bool {
	filename := path.Clean(fmt.Sprintf("%v%v", *route.Target, r.URL.Path))
	info, err := os.Stat(filename)
	if !os.IsNotExist(err) && !info.IsDir() {
		log.Debugf("[file] %s", filename)
		e := fmt.Sprintf(`W/"%x-%x"`, info.ModTime().Unix(), info.Size())
		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, e) {
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
		w.Header().Set("Cache-Control", "public, max-age=2592000")
		w.Header().Set("ETag", e)
		http.ServeFile(w, r, filename)
		return true
	}
	return false
}

func (handler *routeHandler) serveProxy(w http.ResponseWriter, r *http.Request, route *ConfigRoute) bool {
	log.Debugf("[proxy] %s -> %s", r.URL.Path, *route.Target)
	rproxy := handler.proxyForRoute(route)
	if rproxy == nil {
		http.Error(w, "Bad web server configuration.", 500)
		return true
	}
	rproxy.ServeHTTP(w, r, nil)
	return true
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
		DisableSorting:   true,
	})

	config := Config{
		Listen:   "0.0.0.0",
		Port:     17630,
		Key:      "/etc/hawk/hawk.key",
		Cert:     "/etc/hawk/hawk.pem",
		LogLevel: "info",
		Route: []ConfigRoute{
			{
				Handler: "api/v1",
				Path:    "/api/v1",
				Target:  nil,
			},
		},
	}

	listen := flag.String("listen", config.Listen, "Address to listen to")
	port := flag.Int("port", config.Port, "Port to listen to")
	key := flag.String("key", config.Key, "TLS key file")
	cert := flag.String("cert", config.Cert, "TLS cert file")
	loglevel := flag.String("loglevel", config.LogLevel, "Log level (debug|info|warning|error|fatal|panic)")
	cfgfile := flag.String("config", "", "Configuration file")

	flag.Parse()

	if *cfgfile != "" {
		parseConfigFile(*cfgfile, &config)
	}

	if *listen != "0.0.0.0" {
		config.Listen = *listen
	}
	if *port != 17630 {
		config.Port = *port
	}
	if *key != "/etc/hawk/hawk.key" {
		config.Key = *key
	}
	if *cert != "/etc/hawk/hawk.pem" {
		config.Cert = *cert
	}
	if *loglevel != "info" {
		config.LogLevel = *loglevel
	}

	lvl, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.Errorf("Failed to parse loglevel \"%v\" (must be debug|info|warning|error|fatal|panic)", config.LogLevel)
		lvl = log.InfoLevel
	}
	log.SetLevel(lvl)

	routehandler := NewRouteHandler(&config)
	routehandler.cib.Start()
	gziphandler := NewGzipHandler(routehandler)
	fmt.Printf("Listening to https://%s:%d\n", config.Listen, config.Port)
	ListenAndServeWithRedirect(fmt.Sprintf("%s:%d", config.Listen, config.Port), gziphandler, config.Cert, config.Key)
}
