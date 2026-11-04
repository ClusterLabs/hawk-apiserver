package api

import (
	"bytes"
	"net/http"
	"strings"
)

type groupPageResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *groupPageResponse) Header() http.Header { return r.header }
func (r *groupPageResponse) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}
func (r *groupPageResponse) Write(data []byte) (int, error) {
	r.WriteHeader(http.StatusOK)
	return r.body.Write(data)
}

// Editing remains in Ruby. Bridge the Go creation flash into its existing flash
// container without changing the installed Rails application.
func GroupRubyHandler(w http.ResponseWriter, r *http.Request) {
	resourceRubyHandler(w, r, "Group")
}

func TagRubyHandler(w http.ResponseWriter, r *http.Request) {
	resourceRubyHandler(w, r, "Tag")
}

func resourceRubyHandler(w http.ResponseWriter, r *http.Request, resource string) {
	if Routehandler == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/edit") || r.URL.Query().Get("flash") != "created" {
		Routehandler.ServeHTTP(w, r)
		return
	}
	request := r.Clone(r.Context())
	request.Header = r.Header.Clone()
	request.Header.Set("Accept-Encoding", "identity")
	response := &groupPageResponse{header: make(http.Header)}
	Routehandler.ServeHTTP(response, request)
	body := response.body.Bytes()
	if response.status == http.StatusOK && strings.Contains(response.header.Get("Content-Type"), "text/html") && response.header.Get("Content-Encoding") == "" {
		const container = `<div class="row" id="flashes">`
		flash := `<div class="alert-success alert alert-dismissible" role="alert">` +
			`<button class="close" data-dismiss="alert" type="button"><span aria-hidden="true">&times;</span>` +
			`<span class="sr-only">Close</span></button><p>` + resource + ` created successfully</p></div>` +
			`<script>const groupFlashURL = new URL(window.location.href); groupFlashURL.searchParams.delete("flash"); history.replaceState({}, "", groupFlashURL);</script>`
		body = bytes.Replace(body, []byte(container), []byte(container+flash), 1)
		response.header.Del("Content-Length")
		response.header.Del("ETag")
		response.header.Set("Cache-Control", "no-store")
	}
	for key, values := range response.header {
		w.Header()[key] = values
	}
	if response.status == 0 {
		response.status = http.StatusOK
	}
	w.WriteHeader(response.status)
	_, _ = w.Write(body)
}
