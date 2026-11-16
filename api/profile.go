package api

import "net/http"

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	const editPath = "/cib/live/profile/edit"
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost && r.Method != http.MethodPut {
		w.Header().Set("Allow", "GET, HEAD, POST, PUT")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	warning := true
	if cookie, err := r.Cookie("stonithwarning"); err == nil {
		warning = cookie.Value == "true"
	}
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid preferences", 400)
			return
		}
		if r.PostForm.Has("revert") {
			http.Redirect(w, r, editPath, http.StatusSeeOther)
			return
		}
		value := r.PostForm.Get("profile[stonithwarning]")
		if value != "true" && value != "false" {
			http.Error(w, "Select whether to display the fencing warning", 422)
			return
		}
		// Match Ruby: disabling sets a session cookie; enabling removes the override.
		// Language selection is unavailable, so leave any existing locale cookie alone.
		cookie := &http.Cookie{Name: "stonithwarning", Value: "false", Path: "/"}
		if value == "true" {
			cookie.Value = ""
			cookie.MaxAge = -1
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, editPath+"?updated=true", http.StatusSeeOther)
		return
	}
	message := ""
	if r.URL.Query().Get("updated") == "true" {
		message = "Preferences updated successfully"
	}
	renderTemplate(w, "profile", map[string]any{"Title": "Preferences", "Warning": warning, "Message": message})
}
