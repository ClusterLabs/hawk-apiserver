package api

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConstraintTypesTemplate(t *testing.T) {
	types := constraintTypes()
	require.Equal(t, []constraintType{
		{Name: "Location", Path: "locations", Icon: "globe", HelpFilter: "location"},
		{Name: "Colocation", Path: "colocations", Icon: "tasks", HelpFilter: "colocation"},
		{Name: "Order", Path: "orders", Icon: "sort-amount-down", HelpFilter: "order"},
	}, types)
	tmpl, err := template.ParseFiles("../templates/layout.html", "../templates/constraint_types.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "layout.html", map[string]any{"Title": "Add Constraint", "ConstraintTypes": types}))
	for _, item := range types {
		require.Contains(t, body.String(), `href="/cib/live/`+item.Path+`/new"`)
		require.Contains(t, body.String(), `data-help-filter=".`+item.HelpFilter+`"`)
		require.Contains(t, body.String(), `id="help-`+item.Path+`"`)
	}
	require.Contains(t, body.String(), `href="/cib/live/constraints"`)
}

func TestConstraintTypesHandlerRejectsOtherRequests(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodPost, "/cib/live/constraints/types", http.StatusMethodNotAllowed},
		{http.MethodGet, "/cib/live/constraints/types/unknown", http.StatusNotFound},
		{http.MethodPost, "/cib/hacluster/constraints/types", http.StatusMethodNotAllowed},
		{http.MethodGet, "/cib/hacluster/constraints/types/unknown", http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		ConstraintTypesHandler(response, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, response.Code)
	}
}

func TestConstraintTypesHandlerCIBContext(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, base := range []string{"/cib/live", batchBase} {
		for _, suffix := range []string{"", "/"} {
			response := httptest.NewRecorder()
			ConstraintTypesHandler(response, httptest.NewRequest("GET", base+"/constraints/types"+suffix, nil))
			require.Equal(t, 200, response.Code)
			html := response.Body.String()
			for _, item := range constraintTypes() {
				require.Contains(t, html, `href="`+base+`/`+item.Path+`/new"`)
				require.Contains(t, html, `id="help-`+item.Path+`"`)
			}
			require.Contains(t, html, `href="`+base+`/constraints"`)
			require.Equal(t, base == batchBase, strings.Contains(html, `id="btn-show-batch"`))
			require.Contains(t, html, `/static/js/resource_types.js`)
			if base == batchBase {
				require.Contains(t, html, `/static/js/batch.js`)
				require.NotContains(t, html, `href="/cib/live/`)
			}
		}
	}
}
