package api

import "net/http"

// ConfigEditHandler shares the overview and object endpoints across CIBs.
func ConfigEditHandler(w http.ResponseWriter, r *http.Request) {
	base := "/cib/live"
	switch r.URL.Path {
	case "/cib/live/config/edit", "/cib/live/config/edit/":
	case batchBase + "/config/edit", batchBase + "/config/edit/":
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
	data := map[string]any{"Title": "Edit Configuration", "CIBBase": base, "Tabs": []string{"resources", "constraints", "nodes", "tags", "alerts", "fencing"}}
	if base == batchBase {
		data["BatchMode"] = true
		data["BatchConfig"] = map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}
	}
	renderTemplate(w, "config_edit", data)
}
