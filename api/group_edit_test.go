package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const groupEditFixture = `<cib><configuration><resources><primitive id="free10"/><primitive id="free2"/><group id="g" description="Keep &amp; preserve"><primitive id="a" class="ocf" provider="pacemaker" type="Dummy"><instance_attributes id="a-params"><nvpair id="a-p" name="state" value="/tmp/state"/></instance_attributes><operations><op id="a-monitor" name="monitor" interval="10s"/></operations></primitive><primitive id="b" class="ocf" provider="pacemaker" type="Dummy"/><meta_attributes id="custom-meta"><nvpair id="old-priority" name="priority" value="3"/><nvpair id="unknown-meta" name="resource-stickiness" value="123"/></meta_attributes></group><group id="other"><primitive id="busy"/></group><clone id="clone"><group id="nested"><primitive id="nested1"/></group></clone><master id="master"><primitive id="cloned"/></master></resources></configuration></cib>`

func groupEditReader(_ context.Context, name string, args []string, input string) ([]byte, error) {
	if name == "cibadmin" && len(args) == 1 && args[0] == "-Ql" {
		return []byte(groupEditFixture), nil
	}
	return nil, fmt.Errorf("unexpected command: %s %v", name, args)
}
func TestReadGroupEdit(t *testing.T) {
	state, found, err := readGroupEdit(context.Background(), "g", groupEditReader)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"a", "b"}, state.Group.Children)
	require.Equal(t, []string{"a", "b", "free2", "free10"}, state.Available)
	require.Equal(t, map[string]string{"priority": "3", "resource-stickiness": "123"}, state.Group.Meta)
	state, found, err = readGroupEdit(context.Background(), "nested", groupEditReader)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"nested1", "free2", "free10"}, state.Available)
	_, found, err = readGroupEdit(context.Background(), "free2", groupEditReader)
	require.NoError(t, err)
	require.False(t, found)
}
func TestGroupEditUpdatePaths(t *testing.T) {
	for _, tt := range []struct {
		name, body           string
		status               int
		failure, roleFailure bool
	}{
		{name: "metadata", body: `{"id":"g","children":["a","b"],"meta":{"priority":"7"}}`, status: 200},
		{name: "reorder", body: `{"id":"g","children":["b","free2","a"],"meta":{"priority":"8"}}`, status: 200},
		{name: "target role", body: `{"id":"g","children":["a","b"],"meta":{"target-role":"Started"}}`, status: 200},
		{name: "empty", body: `{"id":"g","children":[],"meta":{}}`, status: 400},
		{name: "owned elsewhere", body: `{"id":"g","children":["busy"],"meta":{}}`, status: 409},
		{name: "missing", body: `{"id":"gone","children":["a"],"meta":{}}`, status: 404},
		{name: "command failure", body: `{"id":"g","children":["a","b"],"meta":{}}`, status: 500, failure: true},
		{name: "role failure", body: `{"id":"g","children":["a","b"],"meta":{"target-role":"Started"}}`, status: 500, roleFailure: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writes, actions := 0, 0
			run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
				if name == "cibadmin" && len(args) == 1 {
					return groupEditReader(ctx, name, args, input)
				}
				if name == "crm" && len(args) > 1 && args[1] == "resource" {
					actions++
					require.Equal(t, []string{"-F", "resource", "start", "g"}, args)
					if tt.roleFailure {
						return nil, fmt.Errorf("start failed")
					}
					return nil, nil
				}
				writes++
				if tt.name == "reorder" {
					require.Equal(t, "crm", name)
					require.Equal(t, []string{"-F", "configure", "load", "update", "-"}, args)
					require.Equal(t, "'group' 'g' 'b' 'free2' 'a' 'meta' 'priority=8' 'resource-stickiness=123'\n", input)
				} else {
					require.Equal(t, "cibadmin", name)
					require.Equal(t, []string{"-p", "--replace"}, args)
					var g groupXMLElement
					require.NoError(t, xml.Unmarshal([]byte(input), &g))
					require.Equal(t, "g", groupXMLAttr(g.Attributes, "id"))
					require.Contains(t, input, `description="Keep &amp; preserve"`)
					require.Contains(t, input, `id="a-params"`)
					require.Contains(t, input, `id="a-monitor"`)
					require.Contains(t, input, `id="unknown-meta"`)
					if tt.name == "metadata" {
						require.Contains(t, input, `id="old-priority" name="priority" value="7"`)
					}
				}
				if tt.failure {
					return nil, fmt.Errorf("write failed")
				}
				return nil, nil
			}
			w := httptest.NewRecorder()
			updateGroupEdit(w, httptest.NewRequest("POST", "/", strings.NewReader(tt.body)), run)
			require.Equal(t, tt.status, w.Code, w.Body.String())
			if tt.status < 400 || tt.failure || tt.roleFailure {
				require.Equal(t, 1, writes)
			} else {
				require.Zero(t, writes)
			}
			if tt.name == "target role" || tt.roleFailure {
				require.Equal(t, 1, actions)
			} else {
				require.Zero(t, actions)
			}
		})
	}
}
func TestCreatePrimitiveInGroup(t *testing.T) {
	reads, writes := 0, 0
	run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
		if name == "cibadmin" {
			reads++
			return groupEditReader(ctx, name, args, input)
		}
		writes++
		if writes == 1 {
			require.Equal(t, []string{"configure", "primitive", "new-child", "ocf:pacemaker:Dummy"}, args)
		} else {
			require.Equal(t, []string{"-F", "configure", "load", "update", "-"}, args)
			require.Contains(t, input, "'group' 'g' 'a' 'b' 'new-child'")
		}
		return nil, nil
	}
	w := httptest.NewRecorder()
	createPrimitiveInGroup(w, httptest.NewRequest("POST", "/", nil), Primitive{ID: "new-child", Class: "ocf", Provider: "pacemaker", Type: "Dummy", Parent: "g"}, run)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, 2, reads)
	require.Equal(t, 2, writes)
	require.Contains(t, w.Body.String(), "/cib/live/groups/g/edit?flash=child-created")
}
func TestGroupEditPage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	w := httptest.NewRecorder()
	GroupEditHandler(w, httptest.NewRequest("GET", "/cib/live/groups/g/edit", nil))
	require.Equal(t, 200, w.Code)
	for _, text := range []string{"Edit Group", `data-group-id="g"`, `readonly value="g"`, `name="revert"`, `/cib/live/primitives/new?parent=g`, "RenamePopup.js"} {
		require.Contains(t, w.Body.String(), text)
	}
}
