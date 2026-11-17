package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

const batchCIBFixture = `<cib admin_epoch="0" epoch="1" num_updates="2" have-quorum="1" dc-uuid="1"><configuration><crm_config><cluster_property_set id="props"><nvpair name="cluster-name" value="batch-test"/><nvpair name="fencing-enabled" value="false"/></cluster_property_set></crm_config><nodes><node id="1" uname="n1"/></nodes><resources><primitive id="r1" class="ocf" provider="pacemaker" type="Dummy"/></resources></configuration><status><node_state id="1" uname="n1" in_ccm="true" crmd="online" join="member" expected="member"/><tickets><ticket_state id="t1" granted="false"/></tickets></status></cib>`
const batchCRMFixture = `<crm_mon><nodes><node id="1" name="n1" online="true"/></nodes><resources><resource id="r1" resource_agent="ocf::pacemaker:Dummy" role="Started" active="true" managed="true"><node id="1" name="n1"/></resource></resources></crm_mon>`

type batchTestCall struct {
	live  bool
	name  string
	args  []string
	input string
}

func batchTestBackend(t *testing.T) (*batchBackend, *[]batchTestCall) {
	t.Helper()
	calls := []batchTestCall{}
	backend := &batchBackend{exists: func() (bool, error) { return true, nil }}
	backend.run = func(_ context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		calls = append(calls, batchTestCall{live, name, append([]string{}, args...), input})
		switch name {
		case "cibadmin":
			if len(args) == 1 && args[0] == "-Ql" {
				return []byte(batchCIBFixture), nil
			}
			return nil, nil
		case "crm":
			if reflect.DeepEqual(args, []string{"status", "--as-xml"}) {
				return []byte(batchCRMFixture), nil
			}
			if reflect.DeepEqual(args, []string{"-D", "plain", "configure", "show"}) {
				if live {
					return []byte("primitive r1 Dummy\n"), nil
				}
				return []byte("primitive r1 Dummy meta target-role=Stopped\n"), nil
			}
			return []byte("ok"), nil
		case "crm_shadow", "crm_attribute":
			return []byte("ok"), nil
		case "crm_simulate":
			for i, arg := range args {
				if arg == "-G" {
					require.NoError(t, os.WriteFile(args[i+1], []byte(`<transition_graph/>`), 0600))
				}
				if arg == "-O" {
					require.NoError(t, os.WriteFile(args[i+1], []byte(batchCIBFixture), 0600))
				}
				if arg == "-D" {
					require.NoError(t, os.WriteFile(args[i+1], []byte(`digraph g {}`), 0600))
				}
			}
			return []byte("Simulated transition"), nil
		case "dot":
			return []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), nil
		}
		return nil, fmt.Errorf("Unexpected command %s", name)
	}
	return backend, &calls
}
func batchTestRequest(backend *batchBackend, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, batchAPI+"/"+path, strings.NewReader(body))
	if method == "POST" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	backend.api(response, request)
	return response
}

func TestBatchEnvironmentIsolation(t *testing.T) {
	parent := []string{"PATH=/bin", "CIB_shadow=other", "CIB_file=/tmp/live.xml", "CIB_shadow_dir=/tmp/other", "KEEP=value"}
	original := append([]string{}, parent...)
	var wait sync.WaitGroup
	for i := 0; i < 20; i++ {
		wait.Add(1)
		go func(live bool) {
			defer wait.Done()
			env := batchEnvironment(parent, live)
			if live {
				require.NotContains(t, env, "CIB_shadow="+batchID)
			} else {
				require.Contains(t, env, "CIB_shadow="+batchID)
			}
			require.NotContains(t, env, "CIB_file=/tmp/live.xml")
			require.Contains(t, env, "CIB_shadow_dir="+batchShadowDirectory)
		}(i%2 == 0)
	}
	wait.Wait()
	require.Equal(t, original, parent)
}
func TestBatchCreationAndPageScope(t *testing.T) {
	backend, calls := batchTestBackend(t)
	backend.exists = func() (bool, error) { return false, nil }
	response := httptest.NewRecorder()
	backend.page(response, httptest.NewRequest("GET", batchBase+"?init_cib=true", nil))
	require.Equal(t, 303, response.Code)
	require.Equal(t, batchBase, response.Header().Get("Location"))
	require.Equal(t, []batchTestCall{{true, "crm_shadow", []string{"-b", "-f", "-c", batchID}, ""}}, *calls)
	backend.exists = func() (bool, error) { return true, nil }
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	response = httptest.NewRecorder()
	backend.page(response, httptest.NewRequest("GET", batchBase, nil))
	require.Equal(t, 200, response.Code)
	html := response.Body.String()
	for _, text := range []string{"Batch Mode Active", `id="btn-show-batch"`, `id="btn-discard-batch"`, `id="btn-apply-batch"`, `href="/cib/hacluster/resources/types"`, `href="/cib/hacluster/crm_config/edit"`, `href="/cib/hacluster/config/edit"`, `href="/cib/hacluster/roles"`, `/static/js/batch.js`, `/api/batch/hacluster`} {
		require.Contains(t, html, text)
	}
	require.NotContains(t, html, `href="/dashboard"`)
	require.NotContains(t, html, `href="/reports"`)
	require.NotContains(t, html, `href="/cib/live/resources/types"`)
	live := httptest.NewRecorder()
	LiveStatusHandler(live, httptest.NewRequest("GET", "/cib/live", nil))
	require.Contains(t, live.Body.String(), `href="/dashboard"`)
	require.NotContains(t, live.Body.String(), "Batch Mode Active")
}
func TestBatchPreservesRubyStatusJSON(t *testing.T) {
	backend, _ := batchTestBackend(t)
	previous := Routehandler
	t.Cleanup(func() { Routehandler = previous })
	Routehandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "json", r.URL.Query().Get("format"))
		_, _ = w.Write([]byte(`{"ruby":true}`))
	})
	response := httptest.NewRecorder()
	backend.page(response, httptest.NewRequest("GET", batchBase+"?format=json", nil))
	require.JSONEq(t, `{"ruby":true}`, response.Body.String())
}
func TestBatchSimulationAndReportsStayScoped(t *testing.T) {
	backend, calls := batchTestBackend(t)
	response := batchTestRequest(backend, "POST", "simulate", `{"injections":["node n1 offline","op monitor:10000 r1 7 n1","ticket t1 grant"]}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	var state batchState
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &state))
	require.Equal(t, "batch-test", state.Details.ClusterName)
	require.Len(t, state.Resources, 1)
	require.Equal(t, []string{"t1"}, state.Tickets)
	for _, call := range *calls {
		if call.live {
			require.Equal(t, "cibadmin", call.name)
			require.Equal(t, []string{"-Ql"}, call.args)
		}
	}
	simulate := (*calls)[4]
	require.Equal(t, "crm_simulate", simulate.name)
	require.Equal(t, []string{"-S", "--xml-file"}, simulate.args[:2])
	require.Equal(t, []string{"-d", "n1", "-i", "r1_monitor_10000@n1=7", "-g", "t1"}, simulate.args[9:])
	require.NotContains(t, strings.Join(simulate.args, " "), "stonith_admin")
	for _, file := range []string{"diff", "info", "in", "out", "graph-xml", "graph"} {
		report := batchTestRequest(backend, "GET", "result?file="+file, "")
		require.Equal(t, 200, report.Code, report.Body.String())
		require.NotEmpty(t, report.Body.String())
	}
}
func TestBatchRejectsInvalidEventsAndUnsafeActions(t *testing.T) {
	for _, event := range []string{"node unknown offline", "node n1 reboot", "node n1 offline --live", "op monitor:1 unknown 7 n1", "op monitor:-1 r1 0 n1", "op stop:0 r1 10 n1", "ticket unknown grant"} {
		_, err := batchInjectionArgs([]string{event}, []byte(batchCIBFixture))
		require.Error(t, err)
	}
	backend, calls := batchTestBackend(t)
	for _, input := range []batchAction{{Kind: "node", ID: "n1", Action: "fence"}, {Kind: "resource", ID: "r1", Action: "cleanup"}, {Kind: "resource", ID: "r1\ncommit", Action: "stop"}, {Kind: "resource", ID: "unknown", Action: "start"}, {Kind: "resource", ID: "r1", Action: "migrate", Node: "unknown"}} {
		require.Error(t, backend.action(context.Background(), input))
	}
	for _, call := range *calls {
		require.False(t, call.live)
		require.Equal(t, "cibadmin", call.name)
	}
	*calls = nil
	require.NoError(t, backend.action(context.Background(), batchAction{Kind: "resource", ID: "r1", Action: "stop"}))
	require.NoError(t, backend.action(context.Background(), batchAction{Kind: "node", ID: "n1", Action: "standby"}))
	for _, call := range *calls {
		require.False(t, call.live)
	}
	require.Equal(t, []string{"-F", "resource", "stop", "r1"}, (*calls)[1].args)
	require.Equal(t, "crm_attribute", (*calls)[3].name)
}
func TestBatchApplyRequiresFreshReviewAndSuccessfulVerify(t *testing.T) {
	backend, calls := batchTestBackend(t)
	stale := batchTestRequest(backend, "POST", "apply", `{"epoch":"stale"}`)
	require.Equal(t, 409, stale.Code)
	require.Len(t, *calls, 2)
	*calls = nil
	revision, err := batchReviewRevision([]byte(batchCIBFixture), []byte("primitive r1 Dummy\n"))
	require.NoError(t, err)
	body := fmt.Sprintf(`{"epoch":%q}`, revision)
	success := batchTestRequest(backend, "POST", "apply", body)
	require.Equal(t, 200, success.Code, success.Body.String())
	require.Contains(t, success.Header().Get("Set-Cookie"), "hawk-batch-events=; Path=/; Max-Age=0")
	require.Len(t, *calls, 5)
	require.Equal(t, "crm_shadow", (*calls)[4].name)
	require.Equal(t, []string{"-b", "-f", "-D", batchID}, (*calls)[4].args)
	require.Equal(t, []string{"-F", "configure", "verify"}, (*calls)[2].args)
	require.False(t, (*calls)[3].live)
	require.Equal(t, []string{"-F", "-f", "-", "configure"}, (*calls)[3].args)
	require.Equal(t, "cib commit hacluster\n", (*calls)[3].input)
	original := backend.run
	*calls = nil
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		if name == "crm" && len(args) > 2 && args[2] == "verify" {
			return nil, fmt.Errorf("Invalid configuration")
		}
		return original(ctx, live, name, args, input)
	}
	failed := batchTestRequest(backend, "POST", "apply", body)
	require.Equal(t, 422, failed.Code)
	require.Empty(t, failed.Header().Get("Set-Cookie"))
	for _, call := range *calls {
		require.NotContains(t, call.input, "cib commit")
		require.NotEqual(t, "crm_shadow", call.name)
	}
}
func TestBatchAPIRejectsMissingSessionsAndNonJSONWrites(t *testing.T) {
	backend, calls := batchTestBackend(t)
	bad := httptest.NewRequest("POST", batchAPI+"/apply", strings.NewReader(`{"epoch":"0:1:2"}`))
	response := httptest.NewRecorder()
	backend.api(response, bad)
	require.Equal(t, 415, response.Code)
	require.Empty(t, *calls)
	response = batchTestRequest(backend, "GET", "apply", "")
	require.Equal(t, 405, response.Code)
	backend.exists = func() (bool, error) { return false, nil }
	response = batchTestRequest(backend, "GET", "state", "")
	require.Equal(t, 409, response.Code)
	require.Empty(t, *calls)
}

func TestBatchReviewIgnoresSimulatedStatusChanges(t *testing.T) {
	baseline := []byte("primitive r1 Dummy\n")
	before, err := batchReviewRevision([]byte(batchCIBFixture), baseline)
	require.NoError(t, err)
	changed := strings.Replace(batchCIBFixture, `num_updates="2"`, `num_updates="100"`, 1)
	changed = strings.Replace(changed, `crmd="online"`, `crmd="offline"`, 1)
	after, err := batchReviewRevision([]byte(changed), baseline)
	require.NoError(t, err)
	require.Equal(t, before, after)
	after, err = batchReviewRevision([]byte(changed), []byte("primitive r2 Dummy\n"))
	require.NoError(t, err)
	require.NotEqual(t, before, after)
	changed = strings.Replace(batchCIBFixture, `type="Dummy"`, `type="Stateful"`, 1)
	after, err = batchReviewRevision([]byte(changed), baseline)
	require.NoError(t, err)
	require.NotEqual(t, before, after)
}

func TestBatchDiscardDeletesOnlyShadow(t *testing.T) {
	backend, calls := batchTestBackend(t)
	backend.artifacts.Summary = []byte("Old simulation")
	response := batchTestRequest(backend, "GET", "discard", "")
	require.Equal(t, 405, response.Code)
	require.Empty(t, *calls)
	response = batchTestRequest(backend, "POST", "discard", `{}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	require.JSONEq(t, `{"redirect":"/cib/live"}`, response.Body.String())
	require.Equal(t, []batchTestCall{{false, "crm_shadow", []string{"-b", "-f", "-D", batchID}, ""}}, *calls)
	require.Empty(t, backend.artifacts.Summary)
	require.Contains(t, response.Header().Get("Set-Cookie"), "hawk-batch-events=; Path=/; Max-Age=0")
}

func TestBatchFailedCommitPreservesShadow(t *testing.T) {
	backend, calls := batchTestBackend(t)
	original := backend.run
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		if strings.Contains(input, "cib commit") {
			return nil, fmt.Errorf("Commit failed")
		}
		return original(ctx, live, name, args, input)
	}
	revision, err := batchReviewRevision([]byte(batchCIBFixture), []byte("primitive r1 Dummy\n"))
	require.NoError(t, err)
	response := batchTestRequest(backend, "POST", "apply", fmt.Sprintf(`{"epoch":%q}`, revision))
	require.Equal(t, 422, response.Code)
	require.Empty(t, response.Header().Get("Set-Cookie"))
	for _, call := range *calls {
		require.NotEqual(t, "crm_shadow", call.name)
	}
}
