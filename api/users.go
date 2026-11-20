package api

import (
	"net/http"
	"strings"
)

// Serve the ACL target overview and shared Create/Edit form.
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
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 5 && parts[0] == "cib" && (parts[1] == "live" || parts[1] == batchID) && parts[2] == "users" && parts[4] == "edit" {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", 405)
			return
		}
		data := map[string]any{"Title": "Edit ACL Target", "Editing": true, "TargetID": parts[3], "CIBBase": "/cib/" + parts[1]}
		if parts[1] == batchID {
			data["BatchMode"] = true
			data["BatchConfig"] = map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}
		}
		renderTemplate(w, "user_new", data)
		return
	}
	base := "/cib/live"
	if strings.HasPrefix(r.URL.Path, batchBase+"/") {
		base = batchBase
	}
	if r.URL.Path != base+"/users" && r.URL.Path != base+"/users/" || r.Method == http.MethodPost {
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
	data := map[string]any{"Title": "ACL Targets", "CIBBase": base}
	if base == batchBase {
		data["BatchMode"] = true
		data["BatchConfig"] = map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}
	}
	renderTemplate(w, "users", data)
}
func UsersListHandler(w http.ResponseWriter, r *http.Request) {
	// Roles and targets share the same CIB snapshot and feature checks.
	serveRolesList(w, r, runClusterConfigCommand)
}
func UserDeleteHandler(w http.ResponseWriter, r *http.Request) {
	deleteACLRecord(w, r, runClusterConfigCommand, true)
}
