package api

import (
	"bytes"
	"html/template"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigEditTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/config_edit.html")
	require.NoError(t, err)
	var body bytes.Buffer
	tabs := []string{"resources", "constraints", "nodes", "tags", "alerts", "fencing"}
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Tabs": tabs}))
	for _, tab := range tabs {
		require.Contains(t, body.String(), `href="#`+tab+`"`)
		require.Contains(t, body.String(), `data-config-table="`+tab+`"`)
	}
	for _, link := range []string{"/cib/live/config", "/cib/live/config?xml=true", "/cib/live/graph", "/cib/live/fencing/edit"} {
		require.Contains(t, body.String(), `href="`+link+`"`)
	}
	require.Equal(t, 6, strings.Count(body.String(), `data-config-table=`))
}
func TestConfigEditHandler(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, tt := range []struct {
		method, path string
		status       int
	}{{"GET", "/cib/live/config/edit", 200}, {"HEAD", "/cib/live/config/edit/", 200}, {"POST", "/cib/live/config/edit", 405}, {"DELETE", "/cib/live/config/edit", 405}, {"GET", "/cib/live/config/edit/unknown", 404}} {
		response := httptest.NewRecorder()
		ConfigEditHandler(response, httptest.NewRequest(tt.method, tt.path, nil))
		require.Equal(t, tt.status, response.Code)
		if tt.status == 405 {
			require.Equal(t, "GET, HEAD", response.Header().Get("Allow"))
		}
		if tt.status == 200 {
			require.Contains(t, response.Body.String(), `id="config-editor"`)
		}
	}
}
