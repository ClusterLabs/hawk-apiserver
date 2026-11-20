package api

import "net/http"

// resourceFallback marks an editor route that has not been ported. It never
// invokes another application or attempts to patch a Rails response.
func resourceFallback(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
