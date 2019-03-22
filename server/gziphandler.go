package server

// Original code under Apache 2.0 license.

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// GzipHandler
//
// A http.Handler function which provides gzip compression to request
// responses. Simply create the handler using NewGzipHandler and
// chain it like any other http.Handler function.
//
// This is a stripped down version of github.com/NYTimes/gziphandler.

const (
	// Only enable gzip compression if we have at least
	// minSize bytes of data to compress
	minSize = 512
)

// GzipResponseWriter is a http.ResponseWriter
// which compresses its input.
type GzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
	code   int
	buf    []byte
}

type codings map[string]float64

func (w *GzipResponseWriter) Write(b []byte) (int, error) {
	// set content type
	if _, ok := w.Header()["Content-Type"]; !ok {
		w.Header().Set("Content-Type", http.DetectContentType(b))
	}

	if w.writer != nil {
		n, err := w.writer.Write(b)
		return n, err
	}

	// save the data to be written later
	w.buf = append(w.buf, b...)

	// only enable compression if write is >= minSize
	// and compression isn't already enabled
	if w.Header().Get("Content-Encoding") == "" && len(w.buf) >= minSize {
		err := w.startGzip()
		if err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (w *GzipResponseWriter) startGzip() error {
	w.Header().Set("Content-Encoding", "gzip")
	// if the Content-Length is already set, then calls to Write on gzip
	// will fail to set the Content-Length header since its already set
	// See: https://github.com/golang/go/issues/14975.
	w.Header().Del("Content-Length")
	if w.code != 0 {
		w.ResponseWriter.WriteHeader(w.code)
	}

	// Bytes written during ServeHTTP are redirected to this gzip writer
	// before being written to the underlying response.
	if w.writer == nil {
		w.writer = gzip.NewWriter(nil)
	}
	w.writer.Reset(w.ResponseWriter)

	// Flush the buffer into the gzip response.
	n, err := w.writer.Write(w.buf)
	// This should never happen (per io.Writer docs), but if the write didn't
	// accept the entire buffer but returned no specific error, we have no clue
	// what's going on, so abort just to be safe.
	if err == nil && n < len(w.buf) {
		return io.ErrShortWrite
	}

	w.buf = nil
	return err
}

// WriteHeader just saves the response code until close / actual write.
func (w *GzipResponseWriter) WriteHeader(code int) {
	w.code = code
}

// Close the writer but keep it around for reuse.
func (w *GzipResponseWriter) Close() error {
	if w.writer == nil {
		// Gzip not trigged yet, write out regular response.
		if w.code != 0 {
			w.ResponseWriter.WriteHeader(w.code)
		}
		if w.buf != nil {
			_, writeErr := w.ResponseWriter.Write(w.buf)
			// Returns the error if any at write.
			if writeErr != nil {
				return fmt.Errorf("gziphandler: write to regular responseWriter at close gets error: %q", writeErr.Error())
			}
		}
		return nil
	}

	return w.writer.Close()
}

// Flush flushes the underlying *gzip.Writer and then the underlying
// http.ResponseWriter if it is an http.Flusher. This makes GzipResponseWriter
// an http.Flusher.
func (w *GzipResponseWriter) Flush() {
	if w.writer != nil {
		w.writer.Flush()
	}
	if fw, ok := w.ResponseWriter.(http.Flusher); ok {
		fw.Flush()
	}
}

// Hijack implements http.Hijacker. If the underlying ResponseWriter is a
// Hijacker, its Hijack method is returned. Otherwise an error is returned.
func (w *GzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("http.Hijacker interface is not supported")
}

// verify Hijacker interface implementation
var _ http.Hijacker = &GzipResponseWriter{}

// NewGzipHandler returns a http.Handler with gzip compression.
func NewGzipHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding")

		if acceptsGzip(r) {
			gw := &GzipResponseWriter{
				ResponseWriter: w,
			}
			defer gw.Close()

			h.ServeHTTP(gw, r)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

// acceptsGzip returns true if the given HTTP request indicates that it will
// accept a gzipped response.
func acceptsGzip(r *http.Request) bool {
	acceptedEncodings, _ := parseEncodings(r.Header.Get("Accept-Encoding"))
	return acceptedEncodings["gzip"] > 0.0
}

// parseEncodings attempts to parse a list of codings, per RFC 2616, as might
// appear in an Accept-Encoding header. It returns a map of content-codings to
// quality values, and an error containing the errors encountered. It's probably
// safe to ignore those, because silently ignoring errors is how the internet
// works.
//
// See: http://tools.ietf.org/html/rfc2616#section-14.3.
func parseEncodings(s string) (codings, error) {
	c := make(codings)
	var e []string

	for _, ss := range strings.Split(s, ",") {
		coding, qvalue, err := parseCoding(ss)

		if err != nil {
			e = append(e, err.Error())
		} else {
			c[coding] = qvalue
		}
	}

	// TODO (adammck): Use a proper multi-error struct, so the individual errors
	//                 can be extracted if anyone cares.
	if len(e) > 0 {
		return c, fmt.Errorf("errors while parsing encodings: %s", strings.Join(e, ", "))
	}

	return c, nil
}

// parseCoding parses a single conding (content-coding with an optional qvalue),
// as might appear in an Accept-Encoding header. It attempts to forgive minor
// formatting errors.
func parseCoding(s string) (coding string, qvalue float64, err error) {
	for n, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		qvalue = 1.0

		if n == 0 {
			coding = strings.ToLower(part)
		} else if strings.HasPrefix(part, "q=") {
			qvalue, err = strconv.ParseFloat(strings.TrimPrefix(part, "q="), 64)

			if qvalue < 0.0 {
				qvalue = 0.0
			} else if qvalue > 1.0 {
				qvalue = 1.0
			}
		}
	}

	if coding == "" {
		err = fmt.Errorf("empty content-coding")
	}

	return
}
