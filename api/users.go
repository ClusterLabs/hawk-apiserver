package api

import "net/http"

// The overview and creation form are native; Edit retains the Ruby form.
func UsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/cib/live/users/new" || r.URL.Path == "/cib/live/users/new/" {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		renderTemplate(w, "user_new", map[string]any{"Title": "Create ACL Target"})
		return
	}
	if r.URL.Path != "/cib/live/users" && r.URL.Path != "/cib/live/users/" || r.Method == http.MethodPost {
		if Routehandler != nil {
			Routehandler.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	renderTemplate(w, "users", map[string]any{"Title": "ACL Targets"})
}
func UsersListHandler(w http.ResponseWriter, r *http.Request) {
	// Roles and targets share the same CIB snapshot and feature checks.
	serveRolesList(w, r, runClusterConfigCommand)
}
func UserDeleteHandler(w http.ResponseWriter, r *http.Request) {
	deleteACLRecord(w, r, runClusterConfigCommand, true)
}
