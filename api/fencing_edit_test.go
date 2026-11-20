package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const fencingResources = `<resources><primitive id="ordinary" class="ocf"/><primitive id="sbd" class="stonith"/><clone id="c"><primitive id="other" class="stonith"/></clone></resources>`

func fencingFixture(topology string) string {
	return `<cib><configuration>` + fencingResources + topology + `</configuration></cib>`
}
func TestReadFencingEdit(t *testing.T) {
	topology := `<fencing-topology><fencing-level id="a" target="qwe" index="3" devices="sbd,other"/><fencing-level id="b" target-pattern="q.*" index="1" devices="other"/><fencing-level id="c" target-attribute="rack" target-value="a&amp;b" index="2" devices="sbd"/></fencing-topology>`
	state, err := readFencingEdit(context.Background(), func(_ context.Context, name string, args []string, input string) ([]byte, error) {
		require.Equal(t, "cibadmin", name)
		require.Equal(t, []string{"-Ql"}, args)
		return []byte(fencingFixture(topology)), nil
	})
	require.NoError(t, err)
	require.True(t, state.Exists)
	require.Equal(t, []string{"other", "sbd"}, state.Devices)
	require.Equal(t, []fencingLevel{{Type: "node", Target: "qwe", Index: 3, Devices: []string{"sbd", "other"}}, {Type: "pattern", Target: "q.*", Index: 1, Devices: []string{"other"}}, {Type: "attribute", Target: "rack", Value: "a&b", Index: 2, Devices: []string{"sbd"}}}, state.Fencing)
	for _, topology := range []string{"", "<fencing-topology/>"} {
		state, err := readFencingEdit(context.Background(), func(context.Context, string, []string, string) ([]byte, error) {
			return []byte(fencingFixture(topology)), nil
		})
		require.NoError(t, err)
		require.Equal(t, topology != "", state.Exists)
		require.Empty(t, state.Fencing)
	}
}
func TestFencingSave(t *testing.T) {
	for _, tt := range []struct {
		name, topology string
		clear, fail    bool
	}{
		{name: "create"}, {name: "replace", topology: "<fencing-topology/>"}, {name: "clear", topology: "<fencing-topology/>", clear: true}, {name: "clear populated", topology: `<fencing-topology><fencing-level id="f" target="qwe" index="2" devices="sbd"/></fencing-topology>`, clear: true}, {name: "clear absent", clear: true}, {name: "delete failure", topology: "<fencing-topology/>", clear: true, fail: true}, {name: "command failure", fail: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			run := func(_ context.Context, name string, args []string, input string) ([]byte, error) {
				calls++
				require.Equal(t, "cibadmin", name)
				if calls == 1 {
					require.Equal(t, []string{"-Ql"}, args)
					return []byte(fencingFixture(tt.topology)), nil
				}
				if tt.fail {
					return nil, fmt.Errorf("permission denied")
				}
				if tt.clear {
					require.Equal(t, []string{"--delete", "--xpath", "/cib/configuration/fencing-topology"}, args)
					require.Empty(t, input)
				} else if calls == 2 && tt.topology == "" {
					require.Equal(t, []string{"-p", "-c", "--modify"}, args)
					require.Equal(t, "<cib><configuration><fencing-topology/></configuration></cib>", input)
				} else {
					require.Equal(t, []string{"-p", "--replace", "--xpath", "/cib/configuration/fencing-topology"}, args)
					var topology fencingTopologyXML
					require.NoError(t, xml.Unmarshal([]byte(input), &topology))
					if tt.clear {
						require.Empty(t, topology.Levels)
					} else {
						require.Len(t, topology.Levels, 3)
						require.Equal(t, "qwe", topology.Levels[0].Target)
						require.Equal(t, "a.*", topology.Levels[1].Pattern)
						require.Equal(t, "rack", topology.Levels[2].Attribute)
						require.Equal(t, `a&"<b`, topology.Levels[2].Value)
						require.Equal(t, "fencing-1", topology.Levels[2].ID)
						require.Equal(t, "sbd,other", topology.Levels[0].Devices)
					}
				}
				return nil, nil
			}
			levels := []fencingLevel{{Type: "node", Target: "qwe", Index: 2, Devices: []string{"sbd", "other"}}, {Type: "pattern", Target: "a.*", Index: 3, Devices: []string{"sbd"}}, {Type: "attribute", Target: "rack", Value: `a&"<b`, Index: 1, Devices: []string{"sbd"}}}
			if tt.clear {
				levels = []fencingLevel{}
			}
			body, err := json.Marshal(map[string]any{"fencing": levels})
			require.NoError(t, err)
			w := httptest.NewRecorder()
			serveFencingEdit(w, httptest.NewRequest("POST", "/cib/live/fencing/edit", strings.NewReader(string(body))), run)
			if tt.fail {
				require.Equal(t, 422, w.Code)
				require.Contains(t, w.Body.String(), "permission denied")
			} else {
				require.Equal(t, 200, w.Code, w.Body.String())
				expected := 2
				if tt.topology == "" {
					if tt.clear {
						expected = 1
					} else {
						expected = 3
					}
				}
				require.Equal(t, expected, calls)
			}
		})
	}
}
func TestFencingValidation(t *testing.T) {
	for _, body := range []string{
		`{}`, `{"fencing":null}`, `{"fencing":[]}{}`, `{"fencing":[{"type":"node","target":"qwe","index":1.5}]}`,
		`{"fencing":[{"type":"bad","target":"qwe","index":1}]}`,
		`{"fencing":[{"type":"node","target":" ","index":1}]}`,
		`{"fencing":[{"type":"attribute","target":"rack","value":"","index":1}]}`,
		`{"fencing":[{"type":"node","target":"qwe","index":0}]}`,
		`{"fencing":[{"type":"node","target":"qwe","index":10}]}`,
		`{"fencing":[{"type":"node","target":"qwe","index":1,"devices":["ordinary"]}]}`,
		`{"fencing":[{"type":"node","target":"qwe","index":1,"devices":["gone"]}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			run := func(_ context.Context, _ string, args []string, _ string) ([]byte, error) {
				require.Equal(t, []string{"-Ql"}, args, "invalid input must not write")
				return []byte(fencingFixture("")), nil
			}
			w := httptest.NewRecorder()
			serveFencingEdit(w, httptest.NewRequest("POST", "/cib/live/fencing/edit", strings.NewReader(body)), run)
			require.Contains(t, []int{400, 422}, w.Code)
		})
	}
	w := httptest.NewRecorder()
	serveFencingEdit(w, httptest.NewRequest("DELETE", "/cib/live/fencing/edit", nil), nil)
	require.Equal(t, 405, w.Code)
	w = httptest.NewRecorder()
	fetchFencingEdit(w, httptest.NewRequest("POST", "/", nil), nil)
	require.Equal(t, 405, w.Code)
}
func TestFencingFetchAndPage(t *testing.T) {
	w := httptest.NewRecorder()
	fetchFencingEdit(w, httptest.NewRequest("GET", "/", nil), func(context.Context, string, []string, string) ([]byte, error) {
		return []byte(fencingFixture("")), nil
	})
	require.Equal(t, 200, w.Code)
	require.JSONEq(t, `{"fencing":[],"devices":["other","sbd"]}`, w.Body.String())
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	w = httptest.NewRecorder()
	FencingEditHandler(w, httptest.NewRequest("GET", "/cib/live/fencing/edit", nil))
	require.Equal(t, 200, w.Code, w.Body.String())
	for _, text := range []string{"Edit Fencing Topology", "Add Level", "/cib/live/config/edit#fencing", "fencing_edit.js", `id="fencing-apply"`} {
		require.Contains(t, w.Body.String(), text)
	}
}

func TestBatchFencingEditor(t *testing.T) {
	for _, exists := range []bool{false, true} {
		backend, calls := batchTestBackend(t)
		original := backend.run
		backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
			out, err := original(ctx, live, name, args, input)
			if name == "cibadmin" && args[0] == "-Ql" {
				topology := ""
				if exists {
					topology = `<fencing-topology><fencing-level id="fencing" target="n1" index="2" devices="sbd"/></fencing-topology>`
				}
				return []byte(fencingFixture(topology)), nil
			}
			return out, err
		}
		response := batchTestRequest(backend, "GET", "fencing/fetch", "")
		require.Equal(t, 200, response.Code, response.Body.String())
		require.Contains(t, response.Body.String(), `"devices":["other","sbd"]`)
		response = batchTestRequest(backend, "POST", "fencing/update", `{"fencing":[{"type":"node","target":"n1","index":3,"devices":["sbd"]}]}`)
		require.Equal(t, 200, response.Code, response.Body.String())
		last := (*calls)[len(*calls)-1]
		require.Equal(t, []string{"-p", "--replace", "--xpath", "/cib/configuration/fencing-topology"}, last.args)
		require.Contains(t, last.input, `index="3"`)
		for _, call := range *calls {
			require.False(t, call.live)
		}
		*calls = nil
		response = batchTestRequest(backend, "POST", "fencing/update", `{"fencing":[]}`)
		require.Equal(t, 200, response.Code, response.Body.String())
		if exists {
			require.Equal(t, []string{"--delete", "--xpath", "/cib/configuration/fencing-topology"}, (*calls)[len(*calls)-1].args)
		} else {
			require.Len(t, *calls, 1)
		}
		*calls = nil
		response = batchTestRequest(backend, "POST", "fencing/update", `{"fencing":[{"type":"node","target":"n1","index":1,"devices":["ordinary"]}]}`)
		require.Equal(t, 422, response.Code, response.Body.String())
		require.Len(t, *calls, 1)
	}
}

func TestBatchFencingPage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, suffix := range []string{"", "/"} {
		response := httptest.NewRecorder()
		BatchFencingEditHandler(response, httptest.NewRequest("GET", batchBase+"/fencing/edit"+suffix, nil))
		require.Equal(t, 200, response.Code, response.Body.String())
		for _, text := range []string{`data-fetch-api="` + batchAPI + `/fencing/fetch"`, `data-submit-api="` + batchAPI + `/fencing/update"`, `href="` + batchBase + `/config/edit#fencing"`, `/static/js/batch.js`, `id="fencing-add"`} {
			require.Contains(t, response.Body.String(), text)
		}
	}
}
