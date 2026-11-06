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
	for _, value := range []string{`data-location-id="location1"`, `readonly`, `>Apply</button>`, `id="revert-location-button"`, `/cib/live/config/edit#constraints`} {
		require.Contains(t, body.String(), value)
	}
	response := httptest.NewRecorder()
	LocationRubyHandler(response, httptest.NewRequest("POST", "/cib/live/locations/location1/edit", nil))
	require.Equal(t, 405, response.Code)
}
