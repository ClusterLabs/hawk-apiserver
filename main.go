package main

import (
	"flag"
	"fmt"
	"github.com/ClusterLabs/hawk-apiserver/api"
	"github.com/ClusterLabs/hawk-apiserver/cib"
	"github.com/ClusterLabs/hawk-apiserver/metrics"
	"github.com/ClusterLabs/hawk-apiserver/server"
	"github.com/ClusterLabs/hawk-apiserver/util"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
)

type routeHandler struct {
	cib      cib.AsyncCib
	config   *util.Config
	proxies  map[*util.ConfigRoute]*server.ReverseProxy
	proxymux sync.Mutex
}

// newRoutehandler creates a routeHandler object from a configuration
func newRouteHandler(config *util.Config) *routeHandler {
	return &routeHandler{
		config:  config,
		proxies: make(map[*util.ConfigRoute]*server.ReverseProxy),
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

func (handler *routeHandler) proxyForRoute(route *util.ConfigRoute) *server.ReverseProxy {
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
	proxy = server.NewSingleHostReverseProxy(url, "", http.DefaultMaxIdleConnsPerHost)
	handler.proxymux.Lock()
	handler.proxies[route] = proxy
	handler.proxymux.Unlock()
	return proxy
}

const allConfigTypes = "(cluster_property|rsc_defaults|op_defaults|" +
	"nodes|resources|primitives|groups|masters|clones|bundles|" +
	"constraints|locations|colocations|orders|alerts|tags|acls|fencing)"

const allStatusTypes = "(nodes|resources|summary|failures)"

func (handler *routeHandler) serveAPI(w http.ResponseWriter, r *http.Request, route *util.ConfigRoute) bool {
	log.Debugf("[api/v1] %v", r.URL.Path)
	if !util.CheckHawkAuthMethods(r) {
		http.Error(w, "Unauthorized request.", 401)
		return true
	}
	if r.Method == "GET" {
		prefix := route.Path + "/configuration/"

		// all types below cib/configuration
		allTypes := allConfigTypes
		match, _ := regexp.MatchString(prefix+allTypes+"(/?|/.+/?)$", r.URL.Path)
		if match {
			return api.HandleConfiguration(w, r, handler.cib.Get())
		}

		prefix = route.Path + "/status/"
		allTypes = allStatusTypes
		match, _ = regexp.MatchString(prefix+allTypes+"(/?|/.+/?)$", r.URL.Path)
		if match {
			return api.HandleStatus(w, r, util.GetStdout("crm_mon", "-X"))
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

func (handler *routeHandler) serveMonitor(w http.ResponseWriter, r *http.Request, route *util.ConfigRoute) bool {
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

func (handler *routeHandler) serveFile(w http.ResponseWriter, r *http.Request, route *util.ConfigRoute) bool {
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

func (handler *routeHandler) serveProxy(w http.ResponseWriter, r *http.Request, route *util.ConfigRoute) bool {
	log.Debugf("[proxy] %s -> %s", r.URL.Path, *route.Target)
	rproxy := handler.proxyForRoute(route)
	if rproxy == nil {
		http.Error(w, "Bad web server configuration.", 500)
		return true
	}
	rproxy.ServeHTTP(w, r, nil)
	return true
}

func (handler *routeHandler) serveMetrics(w http.ResponseWriter, r *http.Request, route *util.ConfigRoute) bool {
	log.Debugf("[metrics] %s", r.URL.Path)
	return metrics.HandleMetrics(w)
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
		DisableSorting:   true,
	})

	config := util.Config{
		Listen:   "0.0.0.0",
		Port:     17630,
		Key:      "/etc/hawk/hawk.key",
		Cert:     "/etc/hawk/hawk.pem",
		LogLevel: "info",
		Route: []util.ConfigRoute{
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
		util.ParseConfigFile(*cfgfile, &config)
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
	gziphandler := server.NewGzipHandler(routehandler)
	fmt.Printf("Listening to https://%s:%d\n", config.Listen, config.Port)
	server.ListenAndServeWithRedirect(fmt.Sprintf("%s:%d", config.Listen, config.Port), gziphandler, config.Cert, config.Key)
}
