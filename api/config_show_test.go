package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func configViewTestDirectory(t *testing.T) {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
}
func TestConfigShow(t *testing.T) {
	configViewTestDirectory(t)
	for _, tt := range []struct {
		name, url, accept, input, contentType, language string
		raw                                             bool
	}{
		{name: "show", url: "/cib/live/config", input: "show\n", contentType: "text/html", language: "crmsh"},
		{name: "XML page", url: "/cib/live/config?xml=true", input: "show xml\n", contentType: "text/html", language: "xml"},
		{name: "raw XML", url: "/cib/live/config.xml", input: "show xml\n", contentType: "application/xml", raw: true},
		{name: "XML accept", url: "/cib/live/config", accept: "application/xml", input: "show xml\n", contentType: "application/xml", raw: true},
		{name: "browser accept", url: "/cib/live/config", accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", input: "show\n", contentType: "text/html", language: "crmsh"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output := `<cib><configuration description="&lt;script&gt;"/></cib>`
			run := func(_ context.Context, name string, args []string, input string) ([]byte, error) {
				require.Equal(t, "crm", name)
				require.Equal(t, []string{"configure"}, args)
				require.Equal(t, tt.input, input)
				return []byte(output), nil
			}
			r := httptest.NewRequest("GET", tt.url, nil)
			r.Header.Set("Accept", tt.accept)
			w := httptest.NewRecorder()
			serveConfigShow(w, r, run)
			require.Equal(t, 200, w.Code)
			require.Contains(t, w.Header().Get("Content-Type"), tt.contentType)
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			if tt.raw {
				require.Equal(t, output, w.Body.String())
			} else {
				require.Contains(t, w.Body.String(), `class="`+tt.language+` hljs"`)
				require.Contains(t, w.Body.String(), "&lt;cib&gt;")
				require.NotContains(t, w.Body.String(), output)
				for _, link := range []string{"/cib/live/config/edit", "/cib/live/config?xml=true", "/cib/live/graph"} {
					require.Contains(t, w.Body.String(), link)
				}
			}
		})
	}
	w := httptest.NewRecorder()
	serveConfigShow(w, httptest.NewRequest("POST", "/cib/live/config", nil), nil)
	require.Equal(t, 405, w.Code)
	w = httptest.NewRecorder()
	serveConfigShow(w, httptest.NewRequest("GET", "/cib/live/config", nil), func(context.Context, string, []string, string) ([]byte, error) { return nil, fmt.Errorf("read failed") })
	require.Equal(t, 500, w.Code)
}

func TestGraph(t *testing.T) {
	configViewTestDirectory(t)
	w := httptest.NewRecorder()
	serveGraph(w, httptest.NewRequest("GET", "/cib/live/graph", nil), nil)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "graph-placeholder")
	require.Contains(t, w.Body.String(), "/cib/live/graph.svg?t=")
	require.Contains(t, w.Body.String(), `target="_blank"`)
	for _, mode := range []string{"success", "command error", "missing file", "invalid SVG"} {
		t.Run(mode, func(t *testing.T) {
			file := ""
			run := func(_ context.Context, name string, args []string, input string) ([]byte, error) {
				require.Equal(t, "crm", name)
				require.Len(t, args, 5)
				require.Equal(t, []string{"configure", "graph", "dot"}, args[:3])
				require.Equal(t, "svg", args[4])
				require.Empty(t, input)
				file = args[3]
				if mode == "command error" {
					return nil, fmt.Errorf("graph <failed> & stopped\nnext line")
				}
				if mode == "missing file" {
					return nil, nil
				}
				data := `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="20"><text>cluster</text></svg>`
				if mode == "invalid SVG" {
					data = "not XML"
				}
				require.NoError(t, os.WriteFile(file, []byte(data), 0600))
				return nil, nil
			}
			w := httptest.NewRecorder()
			serveGraph(w, httptest.NewRequest("GET", "/cib/live/graph.svg", nil), run)
			require.Equal(t, 200, w.Code)
			require.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			var root struct{ XMLName xml.Name }
			require.NoError(t, xml.Unmarshal(w.Body.Bytes(), &root))
			require.Equal(t, "svg", root.XMLName.Local)
			if mode == "success" {
				require.Contains(t, w.Body.String(), ">cluster</text>")
			} else {
				require.Contains(t, w.Body.String(), "#ed1c24")
			}
			if mode == "command error" {
				require.Contains(t, w.Body.String(), "graph &lt;failed&gt; &amp; stopped")
			}
			_, err := os.Stat(filepath.Dir(file))
			require.True(t, os.IsNotExist(err), "temporary graph directory must be removed")
		})
	}
	w = httptest.NewRecorder()
	serveGraph(w, httptest.NewRequest("POST", "/cib/live/graph.svg", nil), nil)
	require.Equal(t, 405, w.Code)
}

func TestConfigStatusCompatibility(t *testing.T) {
	old := Routehandler
	t.Cleanup(func() { Routehandler = old })
	Routehandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"existing"}`))
	})
	r := httptest.NewRequest("GET", "/cib/live/config", nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	serveConfigShow(w, r, nil)
	require.True(t, strings.Contains(w.Body.String(), `"existing"`))
}
