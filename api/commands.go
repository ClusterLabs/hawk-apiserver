package api

import "net/http"

// CommandsHandler serves the command log UI. Its data is supplied by the
// existing Rails /cib/live/commands.json endpoint through the Ruby fallback.
func CommandsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/commands" && r.URL.Path != "/cib/live/commands/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	renderTemplate(w, "commands", map[string]any{"Title": "Command Log"})
}
