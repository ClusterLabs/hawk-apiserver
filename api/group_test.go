package api

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupChildrenExcludeContainersAndSortNaturally(t *testing.T) {
	var cib CIB
	require.NoError(t, xml.Unmarshal([]byte(`<cib><configuration><resources>
      <primitive id="r10"/><primitive id="r2"/><primitive id="stonith-sbd"/>
      <group id="existing"><primitive id="grouped"/></group>
      <clone id="clone"><primitive id="cloned"/></clone>
      <master id="master"><primitive id="promotable"/></master>
    </resources></configuration></cib>`), &cib))
	require.Equal(t, []string{"r2", "r10", "stonith-sbd"}, groupChildren(cib))
	require.Empty(t, groupChildren(CIB{}))
}

func TestGroupMetaMatchesRuby(t *testing.T) {
	options := groupMetaOptions()
	require.Len(t, options, 4)
	require.Equal(t, "is-managed", options[0].Name)
	require.Equal(t, "true", options[0].DefaultValue)
	require.Equal(t, "maintenance", options[1].Name)
	require.Equal(t, "false", options[1].DefaultValue)
	require.Equal(t, "priority", options[2].Name)
	require.Equal(t, "0", options[2].DefaultValue)
	require.Equal(t, "target-role", options[3].Name)
	require.Equal(t, "Stopped", options[3].DefaultValue)
	require.Equal(t, []string{"Started", "Stopped", "Master"}, options[3].PossibleValues)
}

func TestGroupCreateValidation(t *testing.T) {
	for _, group := range []groupCreate{
		{Children: []string{"a"}},
		{ID: "bad id", Children: []string{"a"}},
		{ID: "g"},
		{ID: "g", Children: []string{"a", "a"}},
		{ID: "g", Children: []string{"g"}},
		{ID: "g", Children: []string{"a"}, Meta: map[string]string{"priority": "abc"}},
		{ID: "g", Children: []string{"a"}, Meta: map[string]string{"maintenance": "yes"}},
		{ID: "g", Children: []string{"a"}, Meta: map[string]string{"target-role": "invalid"}},
		{ID: "g", Children: []string{"a"}, Meta: map[string]string{"made-up": "value"}},
	} {
		require.Error(t, validateGroup(group))
	}
	require.NoError(t, validateGroup(groupCreate{
		ID: "group-1", Children: []string{"b", "a"},
		Meta: map[string]string{"priority": "-10", "target-role": "Stopped"},
	}))
	// Children can be pre-existing resources created outside the Ruby ID form.
	require.NoError(t, validateGroup(groupCreate{ID: "g", Children: []string{"ip:1"}}))
}

func TestGroupCreationPreservesOrderAndAppliesRole(t *testing.T) {
	for role, action := range map[string]string{"Started": "start", "Stopped": "stop", "Master": "promote"} {
		t.Run(role, func(t *testing.T) {
			var calls [][]string
			read := func() (CIB, int, error) {
				return CIB{Configuration: Configuration{Primitives: []Primitive{{ID: "a"}, {ID: "b"}}}}, 0, nil
			}
			run := func(_ context.Context, args ...string) error {
				calls = append(calls, append([]string{}, args...))
				return nil
			}
			body := fmt.Sprintf(`{"id":"g","children":["b","a"],"meta":{"target-role":%q,"priority":"10"}}`, role)
			response := httptest.NewRecorder()
			createGroup(response, httptest.NewRequest("POST", "/", strings.NewReader(body)), read, run)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			require.Equal(t, [][]string{
				{"configure", "group", "g", "b", "a", "meta", "priority=10", "target-role=" + role},
				{"-F", "resource", action, "g"},
			}, calls)
			var result map[string]string
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
			require.Equal(t, "Group created successfully", result["message"])
			require.Equal(t, "/cib/live/groups/g/edit?flash=created", result["redirect"])
		})
	}
}

func TestGroupCreationRejectsUnavailableChildAndCommandFailure(t *testing.T) {
	read := func() (CIB, int, error) {
		return CIB{Configuration: Configuration{Primitives: []Primitive{{ID: "a"}}}}, 0, nil
	}
	calls := 0
	run := func(_ context.Context, args ...string) error { calls++; return fmt.Errorf("crm rejected the group") }
	response := httptest.NewRecorder()
	createGroup(response, httptest.NewRequest("POST", "/", strings.NewReader(`{"id":"g","children":["unavailable"]}`)), read, run)
	require.Equal(t, http.StatusConflict, response.Code)
	require.Zero(t, calls)
	response = httptest.NewRecorder()
	createGroup(response, httptest.NewRequest("POST", "/", strings.NewReader(`{"id":"g","children":["a"]}`)), read, run)
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Contains(t, response.Body.String(), "crm rejected the group")
	require.Equal(t, 1, calls)
}

func TestGroupPageTemplateAndMethods(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/layout.html", "../templates/group_new.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "layout.html", map[string]any{"Title": "Create Group"}))
	require.Contains(t, body.String(), `name="group[id]"`)
	require.Contains(t, body.String(), `field-prefix="group[meta]"`)
	require.Contains(t, body.String(), `{"target-role":"Stopped"}`)
	for _, handler := range []http.HandlerFunc{FetchGroupChildren, FetchGroupMetaAttributes, GroupCreateHandler} {
		response := httptest.NewRecorder()
		handler(response, httptest.NewRequest("GET", "/", nil))
		require.Equal(t, http.StatusMethodNotAllowed, response.Code)
	}
	response := httptest.NewRecorder()
	GroupNewHandler(response, httptest.NewRequest("POST", "/cib/live/groups/new", nil))
	require.Equal(t, http.StatusMethodNotAllowed, response.Code)
}

func TestUnportedResourceDoesNotUseFallbackApplication(t *testing.T) {
	previous := Routehandler
	t.Cleanup(func() { Routehandler = previous })
	Routehandler = http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("Unported editor contacted another application") })
	response := httptest.NewRecorder()
	resourceFallback(response, httptest.NewRequest("GET", "/cib/hacluster/groups/g/edit?flash=created", nil))
	require.Equal(t, http.StatusNotFound, response.Code)
}

func TestBatchGroupForm(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/layout.html", "../templates/group_new.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "layout.html", map[string]any{
		"Title": "Create Group", "CIBBase": batchBase, "BatchMode": true, "GroupAPI": batchAPI + "/group",
		"BatchConfig": map[string]string{"base": batchBase, "api": batchAPI, "id": batchID},
	}))
	html := body.String()
	require.Contains(t, html, `data-api-base="/api/batch/hacluster/group"`)
	require.Contains(t, html, `data-children-api="/api/batch/hacluster/group/children"`)
	require.Contains(t, html, `href="/cib/hacluster/resources/types"`)
	require.Contains(t, html, `{"target-role":"Stopped"}`)
	require.Contains(t, html, `id="btn-show-batch"`)
	require.Contains(t, html, `/static/js/batch.js`)
	require.NotContains(t, html, `href="/cib/live/`)
}

func TestBatchGroupChildrenAndCreationStayScoped(t *testing.T) {
	backend, calls := batchTestBackend(t)
	original := backend.run
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		out, err := original(ctx, live, name, args, input)
		if name == "cibadmin" {
			return []byte(groupEditFixture), nil
		}
		return out, err
	}
	response := batchTestRequest(backend, "POST", "group/children", `{}`)
	require.Equal(t, 200, response.Code)
	require.JSONEq(t, `["free2","free10"]`, response.Body.String())
	require.Equal(t, []batchTestCall{{false, "cibadmin", []string{"-Ql"}, ""}}, *calls)
	*calls = nil
	response = batchTestRequest(backend, "POST", "group/create", `{"id":"batch-group","children":["free10","free2"],"meta":{"target-role":"Stopped","priority":"10"}}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "/cib/hacluster/groups/batch-group/edit?flash=created")
	require.Len(t, *calls, 3)
	require.Equal(t, []string{"configure", "group", "batch-group", "free10", "free2", "meta", "priority=10", "target-role=Stopped"}, (*calls)[1].args)
	require.Equal(t, []string{"-F", "resource", "stop", "batch-group"}, (*calls)[2].args)
	for _, call := range *calls {
		require.False(t, call.live)
	}
	*calls = nil
	response = batchTestRequest(backend, "POST", "group/create", `{"id":"batch-group","children":["busy"]}`)
	require.Equal(t, 409, response.Code)
	require.Len(t, *calls, 1)
	require.Equal(t, "cibadmin", (*calls)[0].name)
	*calls = nil
	response = batchTestRequest(backend, "POST", "group/create", `{"id":"batch-group","children":[]}`)
	require.Equal(t, 400, response.Code)
	require.Empty(t, *calls)
}
