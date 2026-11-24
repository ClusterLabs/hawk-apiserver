package api

import "net/http"

// CommandsHandler serves the shared live and batch command log UI.
func CommandsHandler(w http.ResponseWriter, r *http.Request) {
	base := "/cib/live"
	switch r.URL.Path {
	case "/cib/live/commands", "/cib/live/commands/":
	case batchBase + "/commands", batchBase + "/commands/":
		base = batchBase
	default:
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data := map[string]any{"Title": "Command Log", "CIBBase": base, "CommandsURL": base + "/commands.json"}
	if base == batchBase {
		data["BatchMode"] = true
		data["BatchConfig"] = map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}
	}
	renderTemplate(w, "commands", data)
}
