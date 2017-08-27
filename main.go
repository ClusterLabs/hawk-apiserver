package main

import (
	"bufio"
	"crypto/tls"
	"github.com/krig/go-pacemaker"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
)


type Adapter func(http.Handler) http.Handler

// Adapt function to enable middlewares on the standard library
func Adapt(h http.Handler, adapters ...Adapter) http.Handler {
    for _, adapter := range adapters {
        h = adapter(h)
    }
    return h
}

type SplitListener struct {
	net.Listener
	config *tls.Config
}

func (l *SplitListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	bconn := &Conn{
		Conn: c,
		buf: bufio.NewReader(c),
	}

	// inspect the first bytes to see if it is HTTPS
	hdr, err := bconn.buf.Peek(6)
	if err != nil {
		log.Printf("Short %s\n", c.RemoteAddr().String())
		bconn.Close()
		return nil, err
	}

	// SSL 3.0 or TLS 1.0, 1.1 and 1.2
	if hdr[0] == 0x16 && hdr[1] == 0x3 && hdr[5] == 0x1 {
		return tls.Server(bconn, l.config), nil
	// SSL 2
	} else if hdr[0] == 0x80 {
		return tls.Server(bconn, l.config), nil
	}
	return bconn, nil
}

type Conn struct {
	net.Conn
	buf *bufio.Reader
}

func (c *Conn) Read(b []byte) (int, error) {
	return c.buf.Read(b)
}

type HTTPRedirectHandler struct {
	handler http.Handler
}

func (handler *HTTPRedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil {
		u := url.URL{
			Scheme: "https",
			Opaque: r.URL.Opaque,
			User: r.URL.User,
			Host: r.Host,
			Path: r.URL.Path,
			RawQuery: r.URL.RawQuery,
			Fragment: r.URL.Fragment,
		}
		log.Printf("http -> %s\n", u.String())
		http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
		return
	}
	handler.handler.ServeHTTP(w, r)
}

func ListenAndServeWithRedirect(addr string, handler http.Handler, cert string, key string) {
	config := &tls.Config{}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http1/1"}
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(cert, key)
	if err != nil {
		log.Fatal(err)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	listener := &SplitListener{
		Listener: ln,
		config: config,
	}


	srv := &http.Server{
		Addr: addr,
		Handler: &HTTPRedirectHandler{
			handler: handler,
		},
	}
	srv.SetKeepAlivesEnabled(true)
	srv.Serve(listener)
}

type AsyncCib struct {
	xmldoc string
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
					acib.lock.Unlock()
				}()

				waiter := make(chan int)
				_, err = cib.Subscribe(func(event pacemaker.CibEvent, doc *pacemaker.CibDocument) {
					if event == pacemaker.UpdateEvent {
						log.Print("Got new CIB UpdateEvent, writing to xmldoc...")
						acib.lock.Lock()
						acib.xmldoc = doc.ToString()
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
			handler.serveAPI(w, r, route.Path)
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

func (handler *routeHandler) serveAPI(w http.ResponseWriter, r *http.Request, apiroot string) {
	if !checkAuth(r) {
		http.Error(w, "Unauthorized request.", 401)
		return
	}
	if r.Method == "GET" {
		if strings.HasPrefix(r.URL.Path, fmt.Sprintf("%s/cib", apiroot)) {
			xmldoc := handler.cib.Get()
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, xmldoc)
			return
		}
	}
	http.Error(w, "Unmatched request.", 500)
}

type offsetContext struct {
	start int
	end int
	line int
	pos int
}

func contextAtOffset(str string, offset int64) offsetContext {
	start, end := strings.LastIndex(str[:offset], "\n")+1, len(str)
	if idx := strings.Index(str[start:], "\n"); idx >= 0 {
		end = start + idx
	}
	line, pos := strings.Count(str[:start], "\n"), int(offset) - start - 1
	return offsetContext{
		start: start,
		end: end,
		line: line,
		pos: pos,
	}
}

func fatalSyntaxError(js string, err error) {
	syntax, ok := err.(*json.SyntaxError)
	if !ok {
		log.Fatal(err)
		return
	}
	ctx := contextAtOffset(js, syntax.Offset)
	log.Printf("Error in line %d: %s", ctx.line, err)
	log.Printf("%s", js[ctx.start:ctx.end])
	log.Fatalf("%s^", strings.Repeat(" ", ctx.pos))
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
		log.Printf("Reading %v...", *cfgfile)
		raw, err := ioutil.ReadFile(*cfgfile)
		if err != nil {
			log.Fatal(err)
		}

		err = json.Unmarshal(raw, &config)
		if err != nil {
			fatalSyntaxError(string(raw), err)
		}
		config.Port = *port
		config.Key = *key
		config.Cert = *cert

		tb, err := json.Marshal(&config)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s", tb)
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


func checkAuth(r *http.Request) bool {
	// Try hawk attrd cookie
	var user string
	var session string
	for _, c := range r.Cookies() {
		if c.Name == "hawk_remember_me_id" {
			user = c.Value
		}
		if c.Name == "hawk_remember_me_key" {
			session = c.Value
		}
	}
	if user != "" && session != "" {
		cmd := exec.Command("/usr/sbin/attrd_updater", "-R", "-Q", "-A", "-n", fmt.Sprintf("hawk_session_%v", user))
		if cmd != nil {
			out, _ := cmd.StdoutPipe()
			cmd.Start()
			// for each line, look for value="..."
			// if ... == sessioncookie, then OK
			scanner := bufio.NewScanner(out)
			tomatch := fmt.Sprintf("value=\"%v\"", session)
			for scanner.Scan() {
				l := scanner.Text()
				if strings.Contains(l, tomatch) {
					log.Printf("Valid session cookie for %v", user)
					return true
				}
			}
			cmd.Wait()
		}
	}
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	if !checkBasicAuth(user, pass) {
		return false
	}
	return true
}

func checkBasicAuth(user, pass string) bool {
	// /usr/sbin/hawk_chkpwd passwd <user>
	// write password
	// close
	cmd := exec.Command("/usr/sbin/hawk_chkpwd", "passwd", user)
	if cmd == nil {
		log.Print("Authorization failed: /usr/sbin/hawk_chkpwd not found")
		return false
	}
	cmd.Stdin = strings.NewReader(pass)
	err := cmd.Run()
	if err != nil {
		log.Printf("Authorization failed: %v", err)
		return false
	}
	return true
}
