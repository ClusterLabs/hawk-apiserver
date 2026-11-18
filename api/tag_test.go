package api

import (
	"context"
	"errors"
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTagObjects(t *testing.T) {
	objects, err := tagObjectsFromXML([]byte(`<cib><configuration><resources><primitive id="r10"><meta_attributes id="excluded"/></primitive><group id="g1"><primitive id="r2"/></group><clone id="c1"><primitive id="r3"/></clone><master id="m1"><group id="g2"><primitive id="r4"/></group></master><template id="excluded-template"/><tag id="excluded-tag"/></resources><constraints><rsc_location id="loc1"><rule id="excluded-rule"/></rsc_location><rsc_ticket id="ticket1"/></constraints></configuration></cib>`))
	require.NoError(t, err)
	require.Equal(t, []string{"c1", "g1", "g2", "loc1", "m1", "r2", "r3", "r4", "r10", "ticket1"}, objects)
	objects, err = tagObjectsFromXML([]byte(`<cib><configuration/></cib>`))
	require.NoError(t, err)
	require.Empty(t, objects)
	require.NotNil(t, objects)
}

func TestTagCreate(t *testing.T) {
	read := func() (CIB, int, error) { return CIB{TagObjects: []string{"r1", "loc1"}}, 0, nil }
	calls := 0
	run := func(_ context.Context, args ...string) error {
		calls++
		require.Equal(t, []string{"configure", "tag", "database", "r1", "loc1"}, args)
		return nil
	}
	response := httptest.NewRecorder()
	createTag(response, httptest.NewRequest("POST", "/", strings.NewReader(`{"id":"database","refs":["r1","loc1"]}`)), read, run)
	require.Equal(t, 200, response.Code)
	require.Equal(t, 1, calls)
	require.Contains(t, response.Body.String(), "/cib/live/tags/database/edit?flash=created")
	for _, payload := range []string{`{"id":"","refs":["r1"]}`, `{"id":"bad id","refs":["r1"]}`, `{"id":"tag1","refs":[]}`, `{"id":"tag1","refs":["invalid.ref"]}`, `{"id":"tag1","refs":["missing"]}`} {
		response = httptest.NewRecorder()
		createTag(response, httptest.NewRequest("POST", "/", strings.NewReader(payload)), read, run)
		require.GreaterOrEqual(t, response.Code, 400)
	}
	require.Equal(t, 1, calls)
	response = httptest.NewRecorder()
	createTag(response, httptest.NewRequest("POST", "/", strings.NewReader(`{"id":"database","refs":["r1","loc1"]}`)), read, func(context.Context, ...string) error { return errors.New("crm failed") })
	require.Equal(t, 500, response.Code)
	require.Contains(t, response.Body.String(), "crm failed")
}

func TestTagTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/tag_new.html")
	require.NoError(t, err)
	var body strings.Builder
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", nil))
	require.Contains(t, body.String(), `name="tag[id]"`)
	require.Contains(t, body.String(), `class="list-group tag-control"`)
	require.Contains(t, body.String(), `href="/cib/live/resources/types"`)
	response := httptest.NewRecorder()
	TagNewHandler(response, httptest.NewRequest("POST", "/cib/live/tags/new", nil))
	require.Equal(t, 405, response.Code)
}

func TestTagCreateError(t *testing.T) {
	require.Equal(t, `The ID "tag123" is already in use`, tagCreateError("tag123", errors.New("\x1b[31mERROR\x1b[0m: Cannot create tag with ID 'tag123': Found existing tag with same ID: exit status 1")))
	require.Equal(t, "Cannot create tag", tagCreateError("tag123", errors.New("\x1b[31mERROR\x1b[0m: Cannot create tag: exit status 1")))
}

func TestBatchTagForm(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/layout.html", "../templates/tag_new.html")
	require.NoError(t, err)
	var body strings.Builder
	require.NoError(t, tmpl.ExecuteTemplate(&body, "layout.html", map[string]any{
		"Title": "Create Tag", "CIBBase": batchBase, "BatchMode": true, "TagAPI": batchAPI + "/tag",
		"BatchConfig": map[string]string{"base": batchBase, "api": batchAPI, "id": batchID},
	}))
	html := body.String()
	require.Contains(t, html, `data-api-base="/api/batch/hacluster/tag"`)
	require.Contains(t, html, `data-objects-api="/api/batch/hacluster/tag/objects"`)
	require.Contains(t, html, `href="/cib/hacluster/resources/types"`)
	require.Contains(t, html, `id="btn-show-batch"`)
	require.Contains(t, html, `/static/js/batch.js`)
	require.NotContains(t, html, `href="/cib/live/`)
}

func TestBatchTagObjectsAndCreationStayScoped(t *testing.T) {
	backend, calls := batchTestBackend(t)
	original := backend.run
	fixture := `<cib><configuration><resources><clone id="cl"><group id="g"><primitive id="nested"/></group></clone><primitive id="r1"/></resources><constraints><rsc_location id="loc1"/></constraints></configuration></cib>`
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		out, err := original(ctx, live, name, args, input)
		if name == "cibadmin" {
			return []byte(fixture), nil
		}
		return out, err
	}
	response := batchTestRequest(backend, "POST", "tag/objects", `{}`)
	require.Equal(t, 200, response.Code)
	require.JSONEq(t, `["cl","g","loc1","nested","r1"]`, response.Body.String())
	require.Equal(t, []batchTestCall{{false, "cibadmin", []string{"-Ql"}, ""}}, *calls)
	*calls = nil
	response = batchTestRequest(backend, "POST", "tag/create", `{"id":"batch-tag","refs":["nested","loc1"]}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "/cib/hacluster/tags/batch-tag/edit?flash=created")
	require.Len(t, *calls, 2)
	require.Equal(t, []string{"configure", "tag", "batch-tag", "nested", "loc1"}, (*calls)[1].args)
	for _, call := range *calls {
		require.False(t, call.live)
	}
	*calls = nil
	response = batchTestRequest(backend, "POST", "tag/create", `{"id":"batch-tag","refs":["missing"]}`)
	require.Equal(t, 409, response.Code)
	require.Len(t, *calls, 1)
	require.Equal(t, "cibadmin", (*calls)[0].name)
	*calls = nil
	response = batchTestRequest(backend, "POST", "tag/create", `{"id":"batch-tag","refs":[]}`)
	require.Equal(t, 400, response.Code)
	require.Empty(t, *calls)
}
