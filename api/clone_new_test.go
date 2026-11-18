package api

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloneChildrenMatchRuby(t *testing.T) {
	var cib CIB
	require.NoError(t, xml.Unmarshal([]byte(groupEditFixture), &cib))
	require.Equal(t, []string{"free2", "free10", "g", "other"}, cloneChildren(cib))
}

func TestBatchCloneNewForm(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/layout.html", "../templates/clone_new.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "layout.html", map[string]any{
		"Title": "Create Clone", "CIBBase": batchBase, "BatchMode": true,
		"CreateAPI": batchAPI + "/clone/create", "ChildrenAPI": batchAPI + "/clone/children",
		"BatchConfig": map[string]string{"base": batchBase, "api": batchAPI, "id": batchID},
	}))
	html := body.String()
	require.Contains(t, html, `options-api="/api/batch/hacluster/clone/children"`)
	require.Contains(t, strings.ReplaceAll(html, `\/`, `/`), batchAPI+"/clone/create")
	require.Contains(t, html, `href="/cib/hacluster/config/edit"`)
	require.Contains(t, html, `id="btn-show-batch"`)
	require.Contains(t, html, `{"target-role":"Stopped"}`)
	require.NotContains(t, html, `href="/cib/live/`)
	require.NotContains(t, html, `changeClass()`)
}

func TestBatchCloneCreationStaysScoped(t *testing.T) {
	backend, calls := batchTestBackend(t)
	original := backend.run
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		out, err := original(ctx, live, name, args, input)
		if name == "cibadmin" {
			return []byte(groupEditFixture), nil
		}
		return out, err
	}
	response := batchTestRequest(backend, "POST", "clone/children", `{}`)
	require.Equal(t, 200, response.Code)
	var content SelectContent
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &content))
	require.Len(t, content.Options, 4)
	require.Contains(t, response.Body.String(), `"Name":"g"`)
	require.NotContains(t, response.Body.String(), `"Name":"busy"`)
	*calls = nil
	response = batchTestRequest(backend, "POST", "clone/create", `{"id":"batch-clone","childResource":"g","primitive":[{"id":"g"}],"meta_attributes":{"nvpair":[{"name":"target-role","value":"Stopped"},{"name":"clone-max","value":"2"},{"name":"empty","value":""}]}}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "/cib/hacluster/clones/batch-clone/edit?flash=created")
	require.Len(t, *calls, 2)
	require.Equal(t, []string{"configure", "clone", "batch-clone", "g", "meta", "target-role=Stopped", "clone-max=2"}, (*calls)[1].args)
	for _, call := range *calls {
		require.False(t, call.live)
	}
	for _, body := range []string{`{}`, `{"id":"bad id","primitive":[{"id":"g"}]}`, `{"id":"c","childResource":"busy"}`} {
		*calls = nil
		response = batchTestRequest(backend, "POST", "clone/create", body)
		require.GreaterOrEqual(t, response.Code, 400)
		for _, call := range *calls {
			require.Equal(t, "cibadmin", call.name)
		}
	}
	response = httptest.NewRecorder()
	CloneNewHandler(response, httptest.NewRequest("POST", batchBase+"/clones/new", nil))
	require.Equal(t, 405, response.Code)
}
