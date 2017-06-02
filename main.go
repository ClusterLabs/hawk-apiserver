package main

import (
	"fmt"
	"net/http"
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"net/url"
	"flag"
	"time"
	"sync"
	"io"
	"github.com/krig/go-pacemaker"
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

func ListenAndServeWithRedirect(addr string, mux *http.ServeMux, cert string, key string) {
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
			handler: mux,
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
					acib.xmldoc = cibxml
					acib.lock.Unlock()
				}()

				waiter := make(chan int)
				_, err = cib.Subscribe(func(event pacemaker.CibEvent, cib string) {
					if event == pacemaker.UpdateEvent {
						log.Print("Got new CIB UpdateEvent, writing to xmldoc...")
						acib.lock.Lock()
						acib.xmldoc = cib
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

func main() {
	// verbose := pflag.BoolP("verbose", "v", false, "Show verbose debug information")
	port := flag.Int("port", 17630, "Port to listen to")
	key := flag.String("key", "harmonies.key", "TLS key file")
	cert := flag.String("cert", "harmonies.pem", "TLS cert file")

	flag.Parse()
	
	mux := http.NewServeMux()

	asyncCib := AsyncCib{}
	asyncCib.Start()

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "img/favicon.ico")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "html/index.html")
	})

	mux.HandleFunc("/api/cib", func(w http.ResponseWriter, r *http.Request) {
		xmldoc := asyncCib.Get()
		w.Header().Set("Content-Type", "application/xml")
		io.WriteString(w, xmldoc)
	})

	fmt.Printf("Listening to https://0.0.0.0:%d\n", *port)
	ListenAndServeWithRedirect(fmt.Sprintf(":%d", *port), mux, *cert, *key)
}

