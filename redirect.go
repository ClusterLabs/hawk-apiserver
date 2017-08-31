package main

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/url"
)

// Provides ListenAndServeWithRedirect(),
// a function which enables seamless
// HTTP -> HTTPS redirect on the same port.
// This is useful for Hawk so that if someone
// accesses the :7630 port over HTTP, it'll
// automagically redirect to HTTPS.

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
