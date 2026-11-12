package api

import "net/http"

// ConfigEditHandler renders the configuration overview. Existing Ruby JSON and
// operation endpoints remain the source of truth for each configuration object.
func ConfigEditHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/config/edit" && r.URL.Path != "/cib/live/config/edit/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	renderTemplate(w, "config_edit", map[string]any{"Title": "Edit Configuration", "Tabs": []string{"resources", "constraints", "nodes", "tags", "alerts", "fencing"}})
}
