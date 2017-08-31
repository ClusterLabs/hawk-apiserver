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


type AsyncCib struct {
	xmldoc string
	version *pacemaker.CibVersion
	lock sync.Mutex
}

func (acib* AsyncCib) Start() {
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
					}
					log.Print("Got new CIB, writing to xmldoc...")
					acib.lock.Lock()
					acib.xmldoc = cibxml.ToString()
					acib.version = cibxml.Version()
					acib.lock.Unlock()
				}()

				waiter := make(chan int)
				_, err = cib.Subscribe(func(event pacemaker.CibEvent, doc *pacemaker.CibDocument) {
					if event == pacemaker.UpdateEvent {
						log.Print("Got new CIB UpdateEvent, writing to xmldoc...")
						acib.lock.Lock()
						acib.xmldoc = doc.ToString()
						acib.version = doc.Version()
						acib.lock.Unlock()
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

func (handler *routeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range handler.config.Route {
		if !strings.HasPrefix(r.URL.Path, route.Path) {
			continue
		}
		if route.Handler == "api/v1" {
			log.Printf("Serving API for %v", route.Path)
			handler.serveAPI(w, r, &route)
			return
		} else if route.Handler == "monitor" {
			log.Printf("Monitor handler for %v", route.Path)
			handler.serveMonitor(w, r, &route)
			return
		} else if route.Handler == "file" && route.Target != nil {
			filename := path.Clean(fmt.Sprintf("%v%v", *route.Target, r.URL.Path))
			info, err := os.Stat(filename)
			if !os.IsNotExist(err) && !info.IsDir() {
				log.Printf("Serving file %s", filename)
				http.ServeFile(w, r, filename)
				return
			}
		} else if route.Handler == "proxy" && route.Target != nil {
			log.Printf("Proxying %s to %s", r.URL.Path, *route.Target)
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
	http.Error(w, "Unmatched request.", 500)
}

func (handler *routeHandler) serveMonitor(w http.ResponseWriter, r *http.Request, route *ConfigRoute) {
	epoch := ""
	args := strings.Split(r.URL.RawQuery, "&")
	if len(args) >= 1 {
		epoch = args[0]
	}

	w.Header().Set("Content-Type", "text/event-stream")
	if r.Header.Get("Origin") != "" {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
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

	/*
	connect_timeout := 60e9
	timeout := make (chan bool)
 
	go func () {
		time.Sleep(connect_timeout)
		timeout <- true
	}()
 	select {
	case msg := <-messages:
		io.WriteString(w, msg)
	case stop := <-timeout:
		return
	}
*/

	new_epoch := ""
	ver := handler.cib.Version()
	if ver != nil {
		new_epoch = ver.String()
		if new_epoch == epoch {
			log.Printf("Current version == queried version, wait for new cib for up to 60s...")
		}
	} else {
		log.Print("No cib connection, wait for new cib for up to 60s...")
	}
	io.WriteString(w, fmt.Sprintf("{\"epoch\":\"%s\"}\n", new_epoch))
	
	http.Error(w, "monitor: No Cib connection", 500)
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
	
	routehandler := &routeHandler{
		config: &config,
		proxies: make(map[*ConfigRoute]*httputil.ReverseProxy),
	}
	routehandler.cib.Start()
	zipper := NewGzipHandler(routehandler)
	fmt.Printf("Listening to https://0.0.0.0:%d\n", *port)
	ListenAndServeWithRedirect(fmt.Sprintf(":%d", *port), zipper, *cert, *key)
}
