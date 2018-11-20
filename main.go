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

// AsyncCib wraps the CIB retrieval from go-pacemaker in an
// asynchronous interface, so that other parts of the server have a
// single copy of the CIB available at any time.
//
// Also provides a subscription interface for the long polling request
// end point, via Wait().
type AsyncCib struct {
	xmldoc   string
	version  *pacemaker.CibVersion
	lock     sync.Mutex
	notifier chan chan string
}

// LogRecord records the last warning and error messages, to avoid
// spamming the log with duplicate messages.
type LogRecord struct {
	warning string
	error   string
}

// Start launches two goroutines, one which runs the go-pacemaker
// mainloop and one which listens for CIB events (the CIB fetcher
// goroutine).
func (acib *AsyncCib) Start() {
	if acib.notifier == nil {
		acib.notifier = make(chan chan string)
	}

	msg := ""
	lastLog := LogRecord{warning: "", error: ""}

	cibFile := os.Getenv("CIB_file")

	cibFetcher := func() {
		for {
			var cib *pacemaker.Cib = nil
			var err error = nil
			if cibFile != "" {
				cib, err = pacemaker.OpenCib(pacemaker.FromFile(cibFile))
			} else {
				cib, err = pacemaker.OpenCib()
			}
			if err != nil {
				msg = fmt.Sprintf("Failed to connect to Pacemaker: %v", err)
				if msg != lastLog.warning {
					log.Warnf(msg)
					lastLog.warning = msg
				}
				time.Sleep(5 * time.Second)
			}
			for cib != nil {
				func() {
					cibxml, err := cib.Query()
					if err != nil {
						msg = fmt.Sprintf("Failed to query CIB: %v", err)
						if msg != lastLog.error {
							log.Errorf(msg)
							lastLog.error = msg
						}
					} else {
						acib.notifyNewCib(cibxml)
					}
				}()

				waiter := make(chan int)
				_, err = cib.Subscribe(func(event pacemaker.CibEvent, doc *pacemaker.CibDocument) {
					if event == pacemaker.UpdateEvent {
						acib.notifyNewCib(doc)
					} else {
						msg = fmt.Sprintf("lost connection: %v", event)
						if msg != lastLog.warning {
							log.Warnf(msg)
							lastLog.warning = msg
						}
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

// Wait blocks for up to `timeout` seconds for a CIB change event.
func (acib *AsyncCib) Wait(timeout int, defval string) string {
	requestChan := make(chan string)
	select {
	case acib.notifier <- requestChan:
	case <-time.After(time.Duration(timeout) * time.Second):
		return defval
	}
	return <-requestChan
}

// Get returns the current CIB XML document (or nil).
func (acib *AsyncCib) Get() string {
	acib.lock.Lock()
	defer acib.lock.Unlock()
	return acib.xmldoc
}

// Version returns the current CIB version (or nil).
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

// Config is the internal representation of the configuration file.
type Config struct {
	Listen   string        `json:"listen"`
	Port     int           `json:"port"`
	Key      string        `json:"key"`
	Cert     string        `json:"cert"`
	LogLevel string        `json:"loglevel"`
	Route    []ConfigRoute `json:"route"`
}

// ConfigRoute is used in the configuration to map routes to handlers.
//
// Possible handlers (this list may be outdated)a:
//
//   * `api/v1` - Exposes a CIB API endpoint.
//   * `metrics` - Prometheus metrics, typically mapped to `/metrics`.
//   * `monitor` - Typically mapped to `/monitor` to handle
//     long-polling for CIB updates.
//   * `file` - A static file serving route mapped to a directory.
//   * `proxy` - Proxies requests to another server.
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

// newRoutehandler creates a routeHandler object from a configuration
func newRouteHandler(config *Config) *routeHandler {
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
		} else if route.Handler == "metrics" {
			if handler.serveMetrics(w, r, &route) {
				return
			}
		} else if route.Handler == "file" && route.Target != nil {
			// TODO(krig): Verify configuration file (ensure Target != nil) in config parser
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
	if route.Handler != "proxy" {
		return nil
	}

	handler.proxymux.Lock()
	proxy, ok := handler.proxies[route]
	handler.proxymux.Unlock()
	if ok {
		return proxy
	}

	// TODO(krig): Parse and verify URL in config parser?
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
		match, _ := regexp.MatchString(prefix+"nodes(/?|/[a-zA-Z0-9]+/?)$", r.URL.Path)
		if match {
			return handleAPINodes(w, r, handler.cib.Get())
		}
		match, _ = regexp.MatchString(prefix+"resources(/?|/[a-zA-Z0-9]+/?)$", r.URL.Path)
		if match {
			return handleAPIResources(w, r, handler.cib.Get())
		}
		match, _ = regexp.MatchString(prefix+"cluster/?$", r.URL.Path)
		if match {
			return handleAPICluster(w, r, handler.cib.Get())
		}
		match, _ = regexp.MatchString(prefix+"constraints(/?|/[a-zA-Z0-9]+/?)$", r.URL.Path)
		if match {
			return handleAPIConstraints(w, r, handler.cib.Get())
		}
		match, _ = regexp.MatchString(prefix + "rsc_defaults/?$", r.URL.Path)
		if match {
			return handleApiRscDefaults(w, r, handler.cib.Get())
		}
		match, _ = regexp.MatchString(prefix + "op_defaults/?$", r.URL.Path)
		if match {
			return handleApiOpDefaults(w, r, handler.cib.Get())
		}
		match, _ = regexp.MatchString(prefix + "alerts(/?|/[a-zA-Z0-9]+/?)$", r.URL.Path)
		if match {
			return handleApiAlerts(w, r, handler.cib.Get())
		}
		match, _ = regexp.MatchString(prefix + "tags(/?|/[a-zA-Z0-9]+/?)$", r.URL.Path)
		if match {
			return handleApiTags(w, r, handler.cib.Get())
		}
		match, _ = regexp.MatchString(prefix + "acls(/?|/[a-zA-Z0-9]+/?)$", r.URL.Path)
		if match {
			return handleApiAcls(w, r, handler.cib.Get())
		}
		match, _ = regexp.MatchString(prefix + "fencing(/?|/[a-zA-Z0-9]+/?)$", r.URL.Path)
		if match {
			return handleApiFencing(w, r, handler.cib.Get())
		}
		if strings.HasPrefix(r.URL.Path, prefix+"cib.xml") {
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

	newEpoch := ""
	ver := handler.cib.Version()
	if ver != nil {
		newEpoch = ver.String()
	}
	if newEpoch == "" || newEpoch == epoch {
		// either we haven't managed to connect
		// to the CIB yet, or there hasn't been
		// any change since we asked last.
		// Wait with a timeout for something to
		// appear, and return whatever we had
		// if we time out
		newEpoch = handler.cib.Wait(60, newEpoch)
	}
	io.WriteString(w, fmt.Sprintf("{\"epoch\":\"%s\"}\n", newEpoch))
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

func (handler *routeHandler) serveMetrics(w http.ResponseWriter, r *http.Request, route *ConfigRoute) bool {
	log.Debugf("[metrics] %s", r.URL.Path)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	return handleMetrics(w)
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

	routehandler := newRouteHandler(&config)
	routehandler.cib.Start()
	gziphandler := NewGzipHandler(routehandler)
	fmt.Printf("Listening to https://%s:%d\n", config.Listen, config.Port)
	ListenAndServeWithRedirect(fmt.Sprintf("%s:%d", config.Listen, config.Port), gziphandler, config.Cert, config.Key)
}
