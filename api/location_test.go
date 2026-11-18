package api

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocationResources(t *testing.T) {
	ids, err := locationResourcesFromXML([]byte(`<cib><configuration><resources><primitive id="r10"/><primitive id="r2"/><group id="g1"><primitive id="nested"/></group><clone id="c1"/><master id="m1"/><bundle id="b1"/><template id="t1"/><tag id="tag1"/></resources></configuration></cib>`))
	require.NoError(t, err)
	require.Equal(t, []string{"b1", "c1", "g1", "m1", "r2", "r10"}, ids)
}

func TestLocationCreateSimple(t *testing.T) {
	location := locationCreate{ID: "prefer-node1", Resources: []string{"r1"}, Rules: []locationRule{{Score: "INFINITY", Expressions: []locationExpression{{Attribute: "#uname", Operation: "eq", Value: "node1"}}}}}
	args, err := locationCreateArgs(location)
	require.NoError(t, err)
	require.Equal(t, []string{"configure", "location", "prefer-node1", "r1", "INFINITY:", "node1"}, args)
	payload, err := json.Marshal(location)
	require.NoError(t, err)
	response := httptest.NewRecorder()
	calls := 0
	read := func() (CIB, int, error) { return CIB{LocationResources: []string{"r1"}}, 0, nil }
	run := func(_ context.Context, got ...string) error { calls++; require.Equal(t, args, got); return nil }
	createLocation(response, httptest.NewRequest("POST", "/", strings.NewReader(string(payload))), read, run)
	require.Equal(t, 200, response.Code)
	require.Equal(t, 1, calls)
	require.Contains(t, response.Body.String(), "/cib/live/locations/prefer-node1/edit?flash=created")
	location.Rules[0].Score = "bad"
	_, err = locationCreateArgs(location)
	require.ErrorContains(t, err, "Invalid score")
	location.Rules[0].Expressions = nil
	_, err = locationCreateArgs(location)
	require.ErrorContains(t, err, "No expressions")
	location.Resources = nil
	_, err = locationCreateArgs(location)
	require.ErrorContains(t, err, "No resource")
}

func TestLocationCreateAdvanced(t *testing.T) {
	location := locationCreate{ID: "advanced", Resources: []string{"r1", "g1"}, Discovery: "exclusive", Rules: []locationRule{
		{Score: "100", Role: "Master", Operator: "or", Expressions: []locationExpression{{Attribute: "site", Operation: "eq", Value: "main site", Type: "string"}, {Attribute: "maintenance", Operation: "not_defined"}}},
		{Score: "preference", Expressions: []locationExpression{{Attribute: "capacity", Operation: "gte", Value: "5", Type: "integer"}}},
	}}
	args, err := locationCreateArgs(location)
	require.NoError(t, err)
	require.Equal(t, []string{"configure", "location", "advanced", "{", "r1", "g1", "}", "resource-discovery=exclusive", "rule", "$role=Master", "100:", "site", "string:eq", "main site", "or", "not_defined", "maintenance", "rule", "preference:", "capacity", "integer:gte", "5"}, args)
	location.Discovery = "invalid"
	_, err = locationCreateArgs(location)
	require.ErrorContains(t, err, "discovery")
}

func TestLocationTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/location_new.html")
	require.NoError(t, err)
	var body strings.Builder
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", nil))
	for _, text := range []string{`name="location[id]"`, `name="location[resources][]"`, `id="locsimple"`, `id="locadvanced"`, `href="/cib/live/constraints/types"`} {
		require.Contains(t, body.String(), text)
	}
	response := httptest.NewRecorder()
	LocationNewHandler(response, httptest.NewRequest("POST", "/cib/live/locations/new", nil))
	require.Equal(t, 405, response.Code)
}

func TestLocationFromXML(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		location, err := locationFromXML([]byte(`<rsc_location id="location1" rsc="r1" node="node1" score="INFINITY"/>`))
		require.NoError(t, err)
		require.Equal(t, "location1", location.ID)
		require.Equal(t, []string{"r1"}, location.Resources)
		require.True(t, simpleLocation(location.Rules))
		args, err := locationCreateArgs(location)
		require.NoError(t, err)
		require.Equal(t, []string{"configure", "location", "location1", "r1", "INFINITY:", "node1"}, args)
	})
	t.Run("advanced", func(t *testing.T) {
		location, err := locationFromXML([]byte(`<rsc_location id="location1" resource-discovery="exclusive"><resource_set id="set1"><resource_ref id="r1"/><resource_ref id="r2"/></resource_set><rule id="rule1" score-attribute="capacity" role="Master" boolean-op="or"><expression id="e1" attribute="site" operation="eq" value="main site" type="string"/><expression id="e2" attribute="maintenance" operation="not_defined"/></rule></rsc_location>`))
		require.NoError(t, err)
		require.Equal(t, []string{"r1", "r2"}, location.Resources)
		require.Equal(t, "exclusive", location.Discovery)
		require.Equal(t, "capacity", location.Rules[0].Score)
		require.Equal(t, "or", location.Rules[0].Operator)
		require.Equal(t, "attr-def", location.Rules[0].Expressions[1].Kind)
		_, err = locationCreateArgs(location)
		require.NoError(t, err)
	})
	for _, child := range []string{`<rule id="nested"/>`, `<date_expression operation="in_range"/>`} {
		_, err := locationFromXML([]byte(`<rsc_location id="location1" rsc="r1"><rule score="100">` + child + `</rule></rsc_location>`))
		require.ErrorContains(t, err, "too complex")
	}
}

func TestLocationEditTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/location_new.html")
	require.NoError(t, err)
	var body strings.Builder
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"EditMode": true, "LocationID": "location1"}))
	for _, value := range []string{`data-location-id="location1"`, `readonly`, `>Apply</button>`, `id="revert-location-button"`, `id="rename-location"`, `id="delete-location"`, `href="/cib/live/locations/new"`, `data-rename-api="/api/cib/resource/rename"`, `data-delete-api="/api/cib/resource/delete"`, `/cib/live/config/edit#constraints`} {
		require.Contains(t, body.String(), value)
	}
	response := httptest.NewRecorder()
	LocationRubyHandler(response, httptest.NewRequest("POST", "/cib/live/locations/location1/edit", nil))
	require.Equal(t, 405, response.Code)
}

func TestBatchLocationForm(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/layout.html", "../templates/location_new.html")
	require.NoError(t, err)
	var body strings.Builder
	require.NoError(t, tmpl.ExecuteTemplate(&body, "layout.html", map[string]any{
		"Title": "Create Location Constraint", "CIBBase": batchBase, "BatchMode": true, "LocationAPI": batchAPI + "/location",
		"BatchConfig": map[string]string{"base": batchBase, "api": batchAPI, "id": batchID},
	}))
	html := body.String()
	for _, text := range []string{`data-api-base="/api/batch/hacluster/location"`, `data-options-api="/api/batch/hacluster/location/options"`, `href="/cib/hacluster/constraints/types"`, `id="btn-show-batch"`, `data-location-tab="locsimple"`, `data-location-tab="locadvanced"`, `LocationRules.js`} {
		require.Contains(t, html, text)
	}
	require.NotContains(t, html, `href="/cib/live/`)
}

func TestBatchLocationOptionsAndCreationStayScoped(t *testing.T) {
	backend, calls := batchTestBackend(t)
	original := backend.run
	fixture := strings.Replace(groupEditFixture, "<configuration>", `<configuration><nodes><node id="1" uname="shadow-node"/></nodes>`, 1)
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		out, err := original(ctx, live, name, args, input)
		if name == "cibadmin" {
			return []byte(fixture), nil
		}
		return out, err
	}
	response := batchTestRequest(backend, "POST", "location/options", `{}`)
	require.Equal(t, 200, response.Code)
	require.JSONEq(t, `{"resources":["clone","free2","free10","g","master","other"],"nodes":["shadow-node"]}`, response.Body.String())
	require.Equal(t, []batchTestCall{{false, "cibadmin", []string{"-Ql"}, ""}}, *calls)
	for _, tt := range []struct {
		body string
		args []string
	}{
		{`{"id":"prefer-group","resources":["g"],"rules":[{"score":"100","expressions":[{"attribute":"#uname","operation":"eq","value":"shadow-node","kind":"uname"}]}]}`, []string{"configure", "location", "prefer-group", "g", "100:", "shadow-node"}},
		{`{"id":"avoid-rack","resources":["free2","free10"],"discovery":"never","rules":[{"score":"-INFINITY","role":"Started","operator":"and","expressions":[{"attribute":"rack","operation":"eq","value":"a","type":"string","kind":"attr-val"}]}]}`, []string{"configure", "location", "avoid-rack", "{", "free2", "free10", "}", "resource-discovery=never", "rule", "$role=Started", "-INFINITY:", "rack", "string:eq", "a"}},
	} {
		*calls = nil
		response = batchTestRequest(backend, "POST", "location/create", tt.body)
		require.Equal(t, 200, response.Code, response.Body.String())
		require.Contains(t, response.Body.String(), "/cib/hacluster/locations/")
		require.Contains(t, response.Body.String(), "/edit?flash=created")
		require.Len(t, *calls, 2)
		require.Equal(t, tt.args, (*calls)[1].args)
		for _, call := range *calls {
			require.False(t, call.live)
		}
	}
	*calls = nil
	response = batchTestRequest(backend, "POST", "location/create", `{"id":"invalid-resource","resources":["a"],"rules":[{"score":"100","expressions":[{"attribute":"#uname","operation":"eq","value":"shadow-node"}]}]}`)
	require.Equal(t, 409, response.Code)
	require.Len(t, *calls, 1)
	require.Equal(t, "cibadmin", (*calls)[0].name)
}

func TestBatchLocationEditTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/layout.html", "../templates/location_new.html")
	require.NoError(t, err)
	var body strings.Builder
	require.NoError(t, tmpl.ExecuteTemplate(&body, "layout.html", map[string]any{
		"Title": "Edit Location Constraint", "EditMode": true, "LocationID": "location1", "CIBBase": batchBase, "BatchMode": true, "LocationAPI": batchAPI + "/location",
		"BatchConfig": map[string]string{"base": batchBase, "api": batchAPI, "id": batchID},
	}))
	html := body.String()
	for _, text := range []string{`data-location-id="location1"`, `data-api-base="/api/batch/hacluster/location"`, `readonly`, `>Apply</button>`, `id="revert-location-button"`, `href="/cib/hacluster/config/edit#constraints"`, `href="/cib/hacluster/locations/new"`, `data-rename-api="/api/batch/hacluster/location/rename"`, `data-delete-api="/api/batch/hacluster/location/delete"`, `id="rename-location"`, `id="delete-location"`, `id="btn-show-batch"`} {
		require.Contains(t, html, text)
	}
	require.NotContains(t, html, `href="/cib/live/`)
}

func TestBatchLocationFetchAndUpdateStayScoped(t *testing.T) {
	backend, calls := batchTestBackend(t)
	original := backend.run
	locationXML := `<rsc_location id="location1" rsc="r1" node="shadow-node" score="100"/>`
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		out, err := original(ctx, live, name, args, input)
		if name == "cibadmin" && len(args) > 1 {
			return []byte(locationXML), nil
		}
		return out, err
	}
	response := batchTestRequest(backend, "POST", "location/fetch?id=location1", `{}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	var data locationCreate
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &data))
	require.Equal(t, "100", data.Rules[0].Score)
	require.Equal(t, "shadow-node", data.Rules[0].Expressions[0].Value)
	require.False(t, (*calls)[0].live)
	for _, body := range []string{
		`{"id":"location1","resources":["r1"],"rules":[{"score":"200","expressions":[{"attribute":"#uname","operation":"eq","value":"shadow-node2"}]}]}`,
		`{"id":"location1","resources":["r1"],"discovery":"always","rules":[{"score":"100","operator":"or","expressions":[{"attribute":"rack","operation":"eq","value":"a","type":"string","kind":"attr-val"},{"attribute":"disk","operation":"defined","kind":"attr-def"}]}]}`,
	} {
		*calls = nil
		response = batchTestRequest(backend, "POST", "location/update", body)
		require.Equal(t, 200, response.Code, response.Body.String())
		require.Contains(t, response.Body.String(), "/cib/hacluster/locations/location1/edit?flash=updated")
		require.Len(t, *calls, 3)
		require.Equal(t, []string{"-F", "configure", "load", "update", "-"}, (*calls)[2].args)
		require.Contains(t, (*calls)[2].input, `"location" "location1" "r1"`)
		for _, call := range *calls {
			require.False(t, call.live)
		}
	}
	locationXML = `<rsc_location id="location1" rsc="r1"><rule score="100"><date_expression operation="in_range"/></rule></rsc_location>`
	*calls = nil
	response = batchTestRequest(backend, "POST", "location/update", `{"id":"location1","resources":["r1"],"rules":[{"score":"200","expressions":[{"attribute":"#uname","operation":"eq","value":"shadow-node"}]}]}`)
	require.Equal(t, 409, response.Code)
	require.Contains(t, response.Body.String(), "too complex")
	require.Len(t, *calls, 1)
	require.Equal(t, "cibadmin", (*calls)[0].name)
}

func TestBatchLocationHeaderActionsStayScoped(t *testing.T) {
	backend, calls := batchTestBackend(t)
	original := backend.run
	fixture := `<cib><configuration><resources><primitive id="r1"/></resources><constraints><rsc_location id="location1" rsc="r1"><rule score="100"><date_expression operation="in_range"/></rule></rsc_location><rsc_order id="order1"/></constraints></configuration></cib>`
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		out, err := original(ctx, live, name, args, input)
		if name == "cibadmin" {
			return []byte(fixture), nil
		}
		return out, err
	}
	for _, tt := range []struct {
		operation, body string
		args            []string
	}{
		{"location/rename", `{"oldID":"location1","newID":"renamed"}`, []string{"-D", "plain", "configure", "rename", "location1", "renamed"}},
		{"location/delete", `"location1"`, []string{"--force", "configure", "delete", "location1"}},
	} {
		*calls = nil
		response := batchTestRequest(backend, "POST", tt.operation, tt.body)
		require.Equal(t, 200, response.Code, response.Body.String())
		require.Len(t, *calls, 2)
		require.Equal(t, tt.args, (*calls)[1].args)
		for _, call := range *calls {
			require.False(t, call.live)
		}
	}
	for _, body := range []string{`"r1"`, `"order1"`, `"missing"`} {
		*calls = nil
		response := batchTestRequest(backend, "POST", "location/delete", body)
		require.Equal(t, 404, response.Code)
		require.Len(t, *calls, 1)
		require.Equal(t, "cibadmin", (*calls)[0].name)
	}
}
