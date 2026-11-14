package api

import (
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Keep the existing status JSON endpoint available to Ruby-backed consumers.
func configurationFormat(r *http.Request) string {
	if strings.HasSuffix(r.URL.Path, ".xml") || r.URL.Query().Get("format") == "xml" {
		return "xml"
	}
	if strings.HasSuffix(r.URL.Path, ".svg") || r.URL.Query().Get("format") == "svg" {
		return "svg"
	}
	accept := r.Header.Get("Accept")
	if !strings.Contains(accept, "text/html") {
		if strings.Contains(accept, "application/json") {
			return "json"
		}
		if strings.Contains(accept, "image/svg+xml") {
			return "svg"
		}
		if strings.Contains(accept, "application/xml") || strings.Contains(accept, "text/xml") {
			return "xml"
		}
	}
	return "html"
}

func ConfigShowHandler(w http.ResponseWriter, r *http.Request) {
	serveConfigShow(w, r, runClusterConfigCommand)
}
func serveConfigShow(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.URL.Path != "/cib/live/config" && r.URL.Path != "/cib/live/config/" && r.URL.Path != "/cib/live/config.xml" || configurationFormat(r) == "json" {
		if Routehandler != nil {
			Routehandler.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	rawXML := configurationFormat(r) == "xml"
	showXML := rawXML || r.URL.Query().Get("xml") == "true"
	command := "show\n"
	if showXML {
		command = "show xml\n"
	}
	output, err := run(r.Context(), "crm", []string{"configure"}, command)
	w.Header().Set("Cache-Control", "no-store")
	if err != nil {
		http.Error(w, "Unable to read configuration: "+err.Error(), 500)
		return
	}
	if rawXML {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write(output)
		return
	}
	mode, language := "show", "crmsh"
	if showXML {
		mode, language = "xml", "xml"
	}
	renderTemplate(w, "config_show", map[string]any{"Title": "Show Configuration", "Heading": "Show Configuration", "Icon": "list", "Mode": mode, "Language": language, "Configuration": string(output)})
}

func GraphHandler(w http.ResponseWriter, r *http.Request) { serveGraph(w, r, runClusterConfigCommand) }
func serveGraph(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.URL.Path != "/cib/live/graph" && r.URL.Path != "/cib/live/graph/" && r.URL.Path != "/cib/live/graph.svg" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if configurationFormat(r) != "svg" {
		renderTemplate(w, "config_show", map[string]any{"Title": "Cluster Graph", "Heading": "Graph", "Icon": "image", "Mode": "graph", "Graph": true, "GraphURL": fmt.Sprintf("/cib/live/graph.svg?t=%d", time.Now().UnixNano())})
		return
	}
	directory, err := os.MkdirTemp("", "hawk-graph-")
	if err != nil {
		http.Error(w, "Unable to prepare graph: "+err.Error(), 500)
		return
	}
	defer os.RemoveAll(directory)
	filename := filepath.Join(directory, "graph.svg")
	_, err = run(r.Context(), "crm", []string{"configure", "graph", "dot", filename, "svg"}, "")
	var data []byte
	if err == nil {
		data, err = os.ReadFile(filename)
	}
	if err != nil {
		data = graphErrorSVG(err.Error())
	} else {
		var root struct{ XMLName xml.Name }
		if decodeErr := xml.Unmarshal(data, &root); decodeErr != nil || root.XMLName.Local != "svg" {
			data = graphErrorSVG("Unable to render cluster graph.")
		}
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Content-Disposition", `inline; filename="graph.svg"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}

// Ruby embeds graph failures as red text in an SVG. Escape every line so
// command diagnostics remain valid image content even when they contain XML.
func graphErrorSVG(message string) []byte {
	lines := strings.Split(strings.TrimSpace(stripANSI(message)), "\n")
	var svg strings.Builder
	fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" width="640" height="%d">`, 16*(len(lines)+1))
	for i, line := range lines {
		fmt.Fprintf(&svg, `<text style="font-family:arial,sans-serif;font-size:12px;fill:#ed1c24" x="10" y="%d">%s</text>`, 16*(i+1), html.EscapeString(line))
	}
	svg.WriteString("</svg>")
	return []byte(svg.String())
}
