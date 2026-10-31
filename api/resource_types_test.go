package api

import (
	"bytes"
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResourceTypesWithoutCrm(t *testing.T) {
	types := resourceTypes(context.Background(), filepath.Join(t.TempDir(), "missing-crm"))
	require.Len(t, types, 5)
	for _, item := range types {
		require.NotEqual(t, "templates", item.Path)
		require.NotEqual(t, "masters", item.Path)
	}
	require.Equal(t, "alerts", types[len(types)-1].Path)
}

func TestResourceTypesWithTemplates(t *testing.T) {
	crm := filepath.Join(t.TempDir(), "crm")
	require.NoError(t, os.WriteFile(crm, []byte("#!/bin/sh\n[ \"$1\" = configure ] && [ \"$2\" = help ] && [ \"$3\" = rsc_template ]\n"), 0755))
	types := resourceTypes(context.Background(), crm)
	require.Len(t, types, 6)
	tmpl, err := template.ParseFiles("../templates/layout.html", "../templates/resource_types.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "layout.html", map[string]any{"Title": "Add Resource", "ResourceTypes": types}))
	for _, item := range types {
		require.NotEqual(t, "masters", item.Path)
		require.Contains(t, body.String(), `href="/cib/live/`+item.Path+`/new"`)
		require.Contains(t, body.String(), `id="help-`+item.Path+`"`)
	}
	require.Contains(t, body.String(), `href="/cib/live/resources"`)
}

func TestResourceTypesHandlerRejectsOtherRequests(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodPost, "/cib/live/resources/types", http.StatusMethodNotAllowed},
		{http.MethodGet, "/cib/live/resources/types/unknown", http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		ResourceTypesHandler(response, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, response.Code)
	}
}
