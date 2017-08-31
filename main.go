package main

import (
	"github.com/krig/go-pacemaker"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
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
	xmldoc string
	version *pacemaker.CibVersion
	lock sync.Mutex
	notifier chan chan string
}

func (acib* AsyncCib) Start() {
	if acib.notifier == nil {
		acib.notifier = make(chan chan string)
	}
	cibFetcher := func () {
		for {
			cib, err := pacemaker.OpenCib()
			if err != nil {
				log.Printf("Failed to connect to Pacemaker: %s", err)
				time.Sleep(5 * time.Second)
			}
			for cib != nil {
				func() {
					cibxml, err := cib.Query()
					if err != nil {
						log.Printf("Failed to query CIB: %s", err)
					} else {
						acib.notifyNewCib(cibxml)
					}
				}()

				waiter := make(chan int)
				_, err = cib.Subscribe(func(event pacemaker.CibEvent, doc *pacemaker.CibDocument) {
					if event == pacemaker.UpdateEvent {
						acib.notifyNewCib(doc)
					} else {
						log.Printf("lost connection: %s\n", event)
						waiter <- 1
					}
				})
				if err != nil {
					log.Printf("Failed to subscribe, rechecking every 5 seconds")
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

func (acib* AsyncCib) notifyNewCib(cibxml *pacemaker.CibDocument) {
	text := cibxml.ToString()
	version := cibxml.Version()
	log.Printf("[CIB]: %v", version)
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
    Port int    `json:"port"`
    Key string `json:"key"`
    Cert string `json:"cert"`
	Route []ConfigRoute `json:"route"`
}

type ConfigRoute struct {
	Handler string `json:"handler"`
	Path string `json:"path"`
	Target *string `json:"target"`
}

type routeHandler struct {
	cib AsyncCib
	config *Config
	proxies map[*ConfigRoute]*httputil.ReverseProxy
}

func NewRouteHandler(config *Config) *routeHandler {
	return &routeHandler{
		config: config,
		proxies: make(map[*ConfigRoute]*httputil.ReverseProxy),
	}
}

func (handler *routeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range handler.config.Route {
		if !strings.HasPrefix(r.URL.Path, route.Path) {
			continue
		}
		if route.Handler == "api/v1" {
			log.Printf("[api/v1] %v", r.URL.Path)
			handler.serveAPI(w, r, &route)
			return
		} else if route.Handler == "monitor" &&
			(r.URL.Path == route.Path || r.URL.Path == fmt.Sprintf("%s.json", route.Path)) {
			log.Printf("[monitor] %v", r.URL.Path)
			handler.serveMonitor(w, r, &route)
			return
		} else if route.Handler == "file" && route.Target != nil {
			filename := path.Clean(fmt.Sprintf("%v%v", *route.Target, r.URL.Path))
			info, err := os.Stat(filename)
			if !os.IsNotExist(err) && !info.IsDir() {
				log.Printf("[file] %s", filename)
				http.ServeFile(w, r, filename)
				return
			}
		} else if route.Handler == "proxy" && route.Target != nil {
			log.Printf("[proxy] %s -> %s", r.URL.Path, *route.Target)
			rproxy := handler.proxyForRoute(&route)
			if rproxy == nil {
				http.Error(w, "Bad web server configuration.", 500)
			}
			rproxy.ServeHTTP(w, r)
			return
		}
	}
	http.Error(w, fmt.Sprintf("Unmatched request: %v.", r.URL.Path), 500)
	return
}

func (handler *routeHandler) proxyForRoute(route *ConfigRoute) *httputil.ReverseProxy {
	proxy, ok := handler.proxies[route]
	if ok {
		return proxy
	}

	url, err := url.Parse(*route.Target)
	if err != nil {
		log.Print(err)
		return nil
	}
	proxy = httputil.NewSingleHostReverseProxy(url)
	handler.proxies[route] = proxy
	return proxy
}

func (handler *routeHandler) serveAPI(w http.ResponseWriter, r *http.Request, route *ConfigRoute) {
	if !checkHawkAuthMethods(r) {
		http.Error(w, "Unauthorized request.", 401)
		return
	}
	if r.Method == "GET" {
		if strings.HasPrefix(r.URL.Path, fmt.Sprintf("%s/cib", route.Path)) {
			xmldoc := handler.cib.Get()
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, xmldoc)
			return
		}
	}
	http.Error(w, fmt.Sprintf("[api/v1]: No route for %v.", r.URL.Path), 500)
}

func (handler *routeHandler) serveMonitor(w http.ResponseWriter, r *http.Request, route *ConfigRoute) {
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
}

func main() {
	config := Config{
		Port: 17630,
		Key: "/etc/hawk/hawk.key",
		Cert: "/etc/hawk/hawk.pem",
		Route: []ConfigRoute {
			{
				Handler: "api/v1",
				Path: "/api/v1",
				Target: nil,
			},
		},
	}

	port := flag.Int("port", config.Port, "Port to listen to")
	key := flag.String("key", config.Key, "TLS key file")
	cert := flag.String("cert", config.Cert, "TLS cert file")
	cfgfile := flag.String("config", "", "Configuration file")

	flag.Parse()

	if *cfgfile != "" {
		parseConfigFile(*cfgfile, &config)
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

	routehandler := NewRouteHandler(&config)
	routehandler.cib.Start()
	gziphandler := NewGzipHandler(routehandler)
	fmt.Printf("Listening to https://0.0.0.0:%d\n", *port)
	ListenAndServeWithRedirect(fmt.Sprintf(":%d", *port), gziphandler, *cert, *key)
}
