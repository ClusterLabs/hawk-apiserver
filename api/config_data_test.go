package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const configOverviewFixture = `<cib><configuration>
<nodes><node id="1" uname="node1"/><node id="2" uname="guest" type="remote"/></nodes>
<resources>
 <primitive id="dummy" class="ocf" provider="pacemaker" type="Dummy"/>
 <clone id="cl"><group id="grp"><primitive id="member" class="systemd" type="httpd"/></group></clone>
 <master id="ms"><primitive id="promoted" class="ocf" provider="heartbeat" type="Stateful"/></master>
 <template id="tmpl" class="ocf" provider="pacemaker" type="Dummy"/><primitive id="inherited" template="tmpl"/>
 <bundle id="container"><docker image="example"/><primitive id="bundled" class="ocf" provider="pacemaker" type="Dummy"/></bundle>
</resources>
<constraints>
 <rsc_location id="loc" rsc="dummy" score="100" node="node1"/>
 <rsc_colocation id="colo" rsc="member" with-rsc="dummy" score="INFINITY"/>
 <rsc_order id="ord"><resource_set id="set1"><resource_ref id="dummy"/><resource_ref id="member"/></resource_set></rsc_order>
 <rsc_ticket id="ticket-constraint" rsc="grp" ticket="ticket1"/>
</constraints>
<tags><tag id="tag1"><obj_ref id="dummy"/><obj_ref id="grp"/></tag></tags>
<alerts><alert id="alert1" path="/usr/bin/notify"><meta_attributes id="alert-meta"><nvpair id="alert-timeout" name="timeout" value="30s"/></meta_attributes><recipient id="recipient1" value="a@example.com"><instance_attributes id="recipient-attrs"><nvpair id="format" name="format" value="text"/></instance_attributes></recipient></alert></alerts>
<fencing-topology><fencing-level id="f1" target="node1" index="2" devices="fence1,fence2"/><fencing-level id="f2" target-pattern="node.*" index="3" devices="fence1"/><fencing-level id="f3" target-attribute="rack" target-value="a" index="1" devices="fence2"/></fencing-topology>
</configuration><status><node_state id="99" uname="status-only"/></status></cib>`

func configFixtureRunner(raw string) clusterConfigCommand {
	return func(_ context.Context, name string, args []string, input string) ([]byte, error) {
		if name != "cibadmin" || strings.Join(args, " ") != "-Ql" || input != "" {
			return nil, fmt.Errorf("Unexpected command: %s %v", name, args)
		}
		return []byte(raw), nil
	}
}
func configListRequest(t *testing.T, kind string, remote bool) []map[string]any {
	t.Helper()
	response := httptest.NewRecorder()
	suffix := "?remote=false"
	if remote {
		suffix = ""
	}
	configListHandler(response, httptest.NewRequest("GET", "/cib/live/"+kind+".json"+suffix, nil), configFixtureRunner(configOverviewFixture), nil)
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	var rows []map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &rows))
	return rows
}
func TestConfigOverviewRows(t *testing.T) {
	resources := configListRequest(t, "resources", false)
	ids := []string{}
	for _, row := range resources {
		ids = append(ids, row["id"].(string))
	}
	require.Equal(t, []string{"dummy", "cl", "grp", "member", "ms", "promoted", "tmpl", "inherited", "container"}, ids)
	require.Equal(t, "ocf:pacemaker:Dummy", resources[0]["sort_type"])
	require.Equal(t, "grp", resources[1]["child"])
	require.Equal(t, []any{"member"}, resources[2]["children"])
	require.Equal(t, "systemd", resources[3]["clazz"])
	require.Equal(t, "tmpl", resources[7]["template"])
	require.Equal(t, false, resources[8]["editable"])
	constraints := configListRequest(t, "constraints", false)
	require.Len(t, constraints, 4)
	require.Equal(t, "location", constraints[0]["object_type"])
	require.Equal(t, []any{"member", "dummy"}, constraints[1]["resources"])
	require.Equal(t, []any{"dummy", "member"}, constraints[2]["resources"])
	require.Equal(t, "ticket1", constraints[3]["ticket"])
	nodes := configListRequest(t, "nodes", false)
	require.Len(t, nodes, 1)
	require.Equal(t, "node1", nodes[0]["name"])
	require.Len(t, configListRequest(t, "nodes", true), 2)
	tags := configListRequest(t, "tags", false)
	require.Equal(t, []any{"dummy", "grp"}, tags[0]["refs"])
	alerts := configListRequest(t, "alerts", false)
	require.Equal(t, "/usr/bin/notify", alerts[0]["path"])
	require.Equal(t, "30s", alerts[0]["meta"].(map[string]any)["timeout"])
	recipient := alerts[0]["recipients"].([]any)[0].(map[string]any)
	require.Equal(t, "a@example.com", recipient["value"])
	require.Equal(t, "text", recipient["attributes"].(map[string]any)["format"])
	fencing := configListRequest(t, "fencing", false)
	require.Len(t, fencing, 3)
	require.Equal(t, "node", fencing[0]["type"])
	require.Equal(t, float64(2), fencing[0]["index"])
	require.Equal(t, []any{"fence1", "fence2"}, fencing[0]["devices"])
	require.Equal(t, "pattern", fencing[1]["type"])
	require.Equal(t, "attribute", fencing[2]["type"])
	require.Equal(t, "rack", fencing[2]["target"])
	require.Equal(t, "a", fencing[2]["value"])
}
func TestConfigOverviewEmptyAndErrors(t *testing.T) {
	for _, kind := range configCollections {
		rows, err := configRows(context.Background(), kind, false, configFixtureRunner(`<cib><configuration/></cib>`))
		require.NoError(t, err)
		encoded, err := json.Marshal(rows)
		require.NoError(t, err)
		require.JSONEq(t, `[]`, string(encoded))
	}
	for _, raw := range []string{`<cib>`, `<html/>`, `<cib/>`} {
		response := httptest.NewRecorder()
		configListHandler(response, httptest.NewRequest("GET", "/cib/live/resources.json", nil), configFixtureRunner(raw), nil)
		require.Equal(t, 500, response.Code)
		require.Contains(t, response.Body.String(), `"error"`)
	}
	for _, method := range []string{"POST", "DELETE"} {
		response := httptest.NewRecorder()
		configListHandler(response, httptest.NewRequest(method, "/cib/live/resources.json", nil), nil, nil)
		require.Equal(t, 405, response.Code)
	}
	response := httptest.NewRecorder()
	configListHandler(response, httptest.NewRequest("GET", "/cib/unknown/resources.json", nil), nil, nil)
	require.Equal(t, 404, response.Code)
}
func TestConfigMutations(t *testing.T) {
	for _, tt := range []struct{ kind, id string }{
		{"resources", "dummy"}, {"resources", "grp"}, {"resources", "cl"}, {"resources", "ms"}, {"resources", "tmpl"}, {"resources", "container"},
		{"constraints", "loc"}, {"constraints", "colo"}, {"constraints", "ord"}, {"constraints", "ticket-constraint"}, {"tags", "tag1"}, {"alerts", "alert1"},
	} {
		for _, rename := range []bool{true, false} {
			calls := [][]string{}
			run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
				if name == "cibadmin" {
					return configFixtureRunner(configOverviewFixture)(ctx, name, args, input)
				}
				require.Equal(t, "crm", name)
				require.Empty(t, input)
				calls = append(calls, args)
				return nil, nil
			}
			body := fmt.Sprintf(`{"id":%q}`, tt.id)
			expected := []string{"--force", "configure", "delete", tt.id}
			if rename {
				body = fmt.Sprintf(`{"oldID":%q,"newID":"renamed"}`, tt.id)
				expected = []string{"-D", "plain", "configure", "rename", tt.id, "renamed"}
			}
			request := httptest.NewRequest("POST", "/api/cib/config/action?kind="+tt.kind, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			configAction(response, request, request.Context(), rename, run)
			require.Equal(t, 200, response.Code, response.Body.String())
			require.Equal(t, [][]string{expected}, calls)
		}
	}
}
func TestConfigMutationValidation(t *testing.T) {
	for _, tt := range []struct {
		kind, body string
		rename     bool
		status     int
	}{
		{"nodes", `{"id":"1"}`, false, 400},
		{"fencing", `{"id":"f1"}`, false, 400},
		{"resources", `{"id":"loc"}`, false, 404},
		{"tags", `{"id":"dummy"}`, false, 404},
		{"constraints", `{"id":"missing"}`, false, 404},
		{"resources", `{"oldID":"dummy","newID":"grp"}`, true, 409},
		{"resources", `{"oldID":"dummy","newID":"alert-timeout"}`, true, 409},
		{"resources", `{"oldID":"dummy","newID":"dummy"}`, true, 400},
		{"resources", `{"oldID":"dummy","newID":""}`, true, 400},
		{"resources", `{"oldID":"dummy","newID":"r; touch /tmp/unsafe"}`, true, 400},
		{"resources", `{"id":"dummy","extra":true}`, false, 400},
		{"resources", `{"id":"dummy"} {}`, false, 400},
	} {
		request := httptest.NewRequest("POST", "/api/cib/config/action?kind="+tt.kind, strings.NewReader(tt.body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		// The fixture runner rejects any write, including during validation errors.
		configAction(response, request, request.Context(), tt.rename, configFixtureRunner(configOverviewFixture))
		require.Equal(t, tt.status, response.Code, response.Body.String())
	}
	request := httptest.NewRequest("POST", "/api/cib/config/delete?kind=resources", strings.NewReader(`{"id":"dummy"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	configAction(response, request, request.Context(), false, func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
		if name == "crm" {
			return nil, fmt.Errorf("crm rejected deletion")
		}
		return configFixtureRunner(configOverviewFixture)(ctx, name, args, input)
	})
	require.Equal(t, 500, response.Code)
	require.Contains(t, response.Body.String(), "crm rejected deletion")
}
func TestBatchConfigOverviewIsolation(t *testing.T) {
	backend, calls := batchTestBackend(t)
	original := backend.run
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		result, err := original(ctx, live, name, args, input)
		if name == "cibadmin" {
			return []byte(configOverviewFixture), nil
		}
		return result, err
	}
	for _, kind := range configCollections {
		response := httptest.NewRecorder()
		configListHandler(response, httptest.NewRequest("GET", batchBase+"/"+kind+".json?remote=false", nil), nil, backend)
		require.Equal(t, 200, response.Code, response.Body.String())
		if kind == "nodes" {
			require.NotContains(t, response.Body.String(), "guest")
		}
	}
	response := batchTestRequest(backend, "POST", "config/rename?kind=resources", `{"oldID":"grp","newID":"new-group"}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = batchTestRequest(backend, "POST", "config/delete?kind=alerts", `{"id":"alert1"}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	for _, call := range *calls {
		require.False(t, call.live, "Every table request and mutation must stay in the shadow CIB")
	}
	require.Equal(t, []string{"--force", "configure", "delete", "alert1"}, (*calls)[len(*calls)-1].args)
	require.NotContains(t, fmt.Sprint(*calls), "crm_simulate", "Table reads must not run simulation")
	backend.exists = func() (bool, error) { return false, nil }
	response = batchTestRequest(backend, "GET", "config/list/resources", "")
	require.Equal(t, 409, response.Code)
	response = batchTestRequest(backend, "POST", "config/delete?kind=resources", `{"id":"dummy"}`)
	require.Equal(t, 409, response.Code)
	response = batchTestRequest(backend, "POST", "config/list/resources", `{}`)
	require.Equal(t, 405, response.Code)
}
