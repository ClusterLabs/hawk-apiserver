package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Opt-in integration test: every write goes to a disposable CIB_file. It never
// creates the real hacluster shadow, commits to the live CIB, or runs an agent.
func TestBatchOfflineSmoke(t *testing.T) {
	if os.Getenv("HAWK_BATCH_OFFLINE_TEST") != "1" {
		t.Skip("Set HAWK_BATCH_OFFLINE_TEST=1 to exercise installed crmsh against disposable CIB files")
	}
	ctx := context.Background()
	// Confirm the production account can read the real CIB; all mutations below
	// still use the disposable files through the injected command runner.
	if os.Geteuid() == 0 {
		_, err := runBatchCommand(ctx, true, "cibadmin", []string{"-Ql"}, "")
		require.NoError(t, err)
	}
	read := exec.Command("cibadmin", "-Ql")
	read.Env = batchEnvironment(os.Environ(), true)
	original, err := read.Output()
	require.NoError(t, err)
	directory := t.TempDir()
	liveFile, shadowFile := filepath.Join(directory, "live.xml"), filepath.Join(directory, "shadow.xml")
	require.NoError(t, os.WriteFile(liveFile, original, 0600))
	require.NoError(t, os.WriteFile(shadowFile, original, 0600))
	backend := &batchBackend{exists: func() (bool, error) { return true, nil }}
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		if strings.Contains(input, "cib commit") {
			return nil, fmt.Errorf("Live commit is prohibited in this integration test")
		}
		if live && !(name == "cibadmin" && reflectStrings(args, []string{"-Ql"}) || name == "crm" && reflectStrings(args, []string{"-D", "plain", "configure", "show"})) {
			return nil, fmt.Errorf("Only reads of the baseline are permitted")
		}
		source := shadowFile
		if live {
			source = liveFile
		}
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Env = append(batchEnvironment(os.Environ(), true), "CIB_file="+source)
		cmd.Stdin = strings.NewReader(input)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return out, fmt.Errorf("%s failed: %w: %s", name, err, stderr.String())
		}
		return out, nil
	}
	_, raw, err := backend.readCIB(ctx, false)
	require.NoError(t, err)
	require.Equal(t, original, raw)
	_, err = backend.run(ctx, false, "crm", []string{"-F", "configure", "load", "update", "/dev/stdin"}, "primitive hawk-batch-proof ocf:pacemaker:Dummy\n")
	require.NoError(t, err)
	// Exercise the status consumed by the shared circle renderer, not just
	// the requested target-role in configuration.
	for _, action := range []string{"start", "stop", "start", "stop"} {
		require.NoError(t, backend.action(ctx, batchAction{Kind: "resource", ID: "hawk-batch-proof", Action: action}))
		require.NoError(t, backend.simulate(ctx, nil))
		state, err := backend.state(ctx)
		require.NoError(t, err)
		found := false
		for _, resource := range state.CRM.Resources {
			if resource.ID == "hawk-batch-proof" {
				found = true
				expected := "Stopped"
				if action == "start" {
					expected = "Started"
				}
				require.Equal(t, expected, resource.Role)
				require.Equal(t, action == "start", resource.Active)
			}
		}
		require.True(t, found, "Simulated resources must remain visible")
		baseline, err := os.ReadFile(liveFile)
		require.NoError(t, err)
		require.Equal(t, original, baseline, "Staging must never change the live CIB")
	}

	state, err := backend.state(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, state.Nodes)
	node := state.Nodes[0].Uname
	require.NoError(t, backend.simulate(ctx, []string{"op monitor:10000 hawk-batch-proof 0 " + node}))
	_, injected, err := backend.readCIB(ctx, false)
	require.NoError(t, err)
	require.Contains(t, string(injected), `operation_key="hawk-batch-proof_monitor_10000"`)
	// Reading the page state must leave the injected history intact.
	_, err = backend.state(ctx)
	require.NoError(t, err)
	_, persisted, err := backend.readCIB(ctx, false)
	require.NoError(t, err)
	require.Equal(t, injected, persisted)
	require.NoError(t, backend.simulate(ctx, []string{"node " + state.Nodes[0].Uname + " offline"}))
	state, err = backend.state(ctx)
	require.NoError(t, err)
	found := false
	for _, resource := range state.Resources {
		if resource.ID == "hawk-batch-proof" {
			found = true
			require.Equal(t, "Stopped", resource.TargetRole)
		}
	}
	require.True(t, found)
	diff, _, err := backend.diff(ctx)
	require.NoError(t, err)
	require.Contains(t, diff, "hawk-batch-proof")
	unchanged, err := os.ReadFile(liveFile)
	require.NoError(t, err)
	require.Equal(t, original, unchanged)
	require.NotEmpty(t, backend.artifacts.Summary)
	require.NotEmpty(t, backend.artifacts.Graph)
}
func reflectStrings(a, b []string) bool { return strings.Join(a, "\x00") == strings.Join(b, "\x00") }

// Cover CIB_shadow as well as CIB_file: Pacemaker handles --save-output
// differently when a shadow remains selected during simulation.
func TestBatchShadowStartStopStatus(t *testing.T) {
	if os.Getenv("HAWK_BATCH_OFFLINE_TEST") != "1" {
		t.Skip("Set HAWK_BATCH_OFFLINE_TEST=1 to test a disposable shadow")
	}
	ctx := context.Background()
	read := exec.Command("cibadmin", "-Ql")
	read.Env = batchEnvironment(os.Environ(), true)
	original, err := read.Output()
	require.NoError(t, err)
	directory := t.TempDir()
	baseline := filepath.Join(directory, "baseline.xml")
	shadow := filepath.Join(directory, "shadow.proof")
	require.NoError(t, os.WriteFile(baseline, original, 0600))
	require.NoError(t, os.WriteFile(shadow, original, 0600))
	backend := &batchBackend{prepare: prepareBatchDirectory, exists: func() (bool, error) { return true, nil }}
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		if name == "crm_simulate" {
			// Exercise production credentials and shadow-environment isolation.
			require.Contains(t, args, "--xml-file")
			return runBatchCommand(ctx, false, name, args, input)
		}
		if strings.Contains(input, "cib commit") {
			return nil, fmt.Errorf("Live commits are prohibited")
		}
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Env = batchEnvironment(os.Environ(), true)
		if live {
			cmd.Env = append(cmd.Env, "CIB_file="+baseline)
		} else {
			cmd.Env = append(cmd.Env, "CIB_shadow=proof", "CIB_shadow_dir="+directory)
		}
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return out, fmt.Errorf("%s: %w: %s", name, err, out)
		}
		return out, nil
	}
	response := batchTestRequest(backend, "POST", "primitive/create", `{"id":"hawk-color-proof","class":"ocf","provider":"pacemaker","type":"Dummy"}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	_, created, err := backend.readCIB(ctx, false)
	require.NoError(t, err)
	require.Contains(t, string(created), `id="hawk-color-proof"`)
	for _, action := range []string{"start", "stop", "start", "stop"} {
		require.NoError(t, backend.action(ctx, batchAction{Kind: "resource", ID: "hawk-color-proof", Action: action}))
		require.NoError(t, backend.simulate(ctx, nil))
		state, err := backend.state(ctx)
		require.NoError(t, err)
		found := false
		for _, resource := range state.CRM.Resources {
			if resource.ID == "hawk-color-proof" {
				found = true
				expected := "Stopped"
				if action == "start" {
					expected = "Started"
				}
				require.Equal(t, expected, resource.Role)
				require.Equal(t, action == "start", resource.Active)
			}
		}
		require.True(t, found)
	}
	response = batchTestRequest(backend, "POST", "group/create", `{"id":"hawk-group-proof","children":["hawk-color-proof"],"meta":{"target-role":"Stopped"}}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = batchTestRequest(backend, "POST", "group/children", `{}`)
	require.Equal(t, 200, response.Code)
	require.NotContains(t, response.Body.String(), "hawk-color-proof", "Grouped primitives are no longer available children")
	cib, _, err := backend.readCIB(ctx, false)
	require.NoError(t, err)
	foundGroup := false
	for _, group := range cib.Configuration.Groups {
		if group.ID == "hawk-group-proof" {
			foundGroup = true
			require.Len(t, group.Primitives, 1)
			require.Equal(t, "hawk-color-proof", group.Primitives[0].ID)
		}
	}
	require.True(t, foundGroup)
	response = batchTestRequest(backend, "POST", "clone/create", `{"id":"hawk-clone-proof","childResource":"hawk-group-proof","primitive":[{"id":"hawk-group-proof"}],"meta_attributes":{"nvpair":[{"name":"target-role","value":"Stopped"}]}}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	cib, raw, err := backend.readCIB(ctx, false)
	require.NoError(t, err)
	children := cloneChildren(cib)
	require.NotContains(t, children, "hawk-group-proof")
	require.NotContains(t, children, "hawk-color-proof")
	require.Contains(t, string(raw), `<clone id="hawk-clone-proof">`)
	require.Contains(t, string(raw), `<group id="hawk-group-proof">`)
	response = batchTestRequest(backend, "POST", "tag/objects", `{}`)
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), "hawk-color-proof", "Nested resources must remain taggable")
	response = batchTestRequest(backend, "POST", "tag/create", `{"id":"hawk-tag-proof","refs":["hawk-color-proof","hawk-group-proof"]}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	_, raw, err = backend.readCIB(ctx, false)
	require.NoError(t, err)
	require.Contains(t, string(raw), `<tag id="hawk-tag-proof">`)
	require.Contains(t, string(raw), `<obj_ref id="hawk-color-proof"/>`)
	require.Contains(t, string(raw), `<obj_ref id="hawk-group-proof"/>`)
	response = batchTestRequest(backend, "GET", "tag/fetch?id=hawk-tag-proof", "")
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `"refs":["hawk-color-proof","hawk-group-proof"]`)
	response = batchTestRequest(backend, "POST", "tag/update", `{"id":"hawk-tag-proof","refs":["hawk-color-proof"]}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = batchTestRequest(backend, "POST", "tag/rename", `{"oldID":"hawk-tag-proof","newID":"hawk-tag-renamed"}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = batchTestRequest(backend, "GET", "tag/fetch?id=hawk-tag-renamed", "")
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `"refs":["hawk-color-proof"]`)
	response = batchTestRequest(backend, "POST", "tag/delete", `"hawk-tag-renamed"`)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = batchTestRequest(backend, "GET", "tag/fetch?id=hawk-tag-renamed", "")
	require.Equal(t, 404, response.Code)
	response = batchTestRequest(backend, "POST", "alert/create", `{"id":"hawk-alert-proof","path":"/bin/true","params":{"message":"cluster event"},"meta":{"timeout":"30s"},"recipients":[{"id":"proof-recipient","value":"user@example.com","params":{"subject":"proof"},"meta":{"timeout":"10s"}}]}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	alert, found, err := loadAlert(ctx, "hawk-alert-proof", func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
		return backend.run(ctx, false, name, args, input)
	})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "/bin/true", alert.Path)
	require.Equal(t, "cluster event", alert.Params["message"])
	require.Equal(t, "30s", alert.Meta["timeout"])
	require.Len(t, alert.Recipients, 1)
	require.Equal(t, "user@example.com", alert.Recipients[0].Value)
	require.Equal(t, "proof", alert.Recipients[0].Params["subject"])
	require.Equal(t, "10s", alert.Recipients[0].Meta["timeout"])
	response = batchTestRequest(backend, "GET", "alert/fetch?id=hawk-alert-proof", "")
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), "user@example.com")
	response = batchTestRequest(backend, "POST", "alert/update", `{"id":"hawk-alert-proof","path":"/bin/true","params":{"message":"updated event"},"meta":{"timeout":"60s"},"recipients":[{"id":"updated-recipient","value":"ops@example.com","params":{"subject":"updated"},"meta":{"timeout":"15s"}}]}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	alert, found, err = loadAlert(ctx, "hawk-alert-proof", func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
		return backend.run(ctx, false, name, args, input)
	})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "updated event", alert.Params["message"])
	require.Equal(t, "60s", alert.Meta["timeout"])
	require.Len(t, alert.Recipients, 1)
	require.Equal(t, "ops@example.com", alert.Recipients[0].Value)
	require.Equal(t, "updated", alert.Recipients[0].Params["subject"])
	require.Equal(t, "15s", alert.Recipients[0].Meta["timeout"])
	response = batchTestRequest(backend, "POST", "alert/rename", `{"oldID":"hawk-alert-proof","newID":"hawk-alert-renamed"}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = batchTestRequest(backend, "GET", "alert/fetch?id=hawk-alert-renamed", "")
	require.Equal(t, 200, response.Code)
	response = batchTestRequest(backend, "POST", "alert/delete", `"hawk-alert-renamed"`)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = batchTestRequest(backend, "GET", "alert/fetch?id=hawk-alert-renamed", "")
	require.Equal(t, 404, response.Code)
	response = batchTestRequest(backend, "POST", "location/options", `{}`)
	require.Equal(t, 200, response.Code)
	var options struct {
		Resources []string `json:"resources"`
		Nodes     []string `json:"nodes"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &options))
	require.Contains(t, options.Resources, "hawk-clone-proof")
	require.NotContains(t, options.Resources, "hawk-color-proof")
	require.NotEmpty(t, options.Nodes)
	locationPayload := fmt.Sprintf(`{"id":"hawk-location-proof","resources":["hawk-clone-proof"],"rules":[{"score":"100","expressions":[{"attribute":"#uname","operation":"eq","value":%q,"kind":"uname"}]}]}`, options.Nodes[0])
	response = batchTestRequest(backend, "POST", "location/create", locationPayload)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = batchTestRequest(backend, "POST", "location/create", `{"id":"hawk-location-rule-proof","resources":["hawk-clone-proof"],"discovery":"always","rules":[{"score":"100","operator":"and","expressions":[{"attribute":"rack","operation":"eq","value":"a","type":"string","kind":"attr-val"}]}]}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	_, raw, err = backend.readCIB(ctx, false)
	require.NoError(t, err)
	require.Contains(t, string(raw), `id="hawk-location-proof"`)
	require.Contains(t, string(raw), `id="hawk-location-rule-proof"`)
	require.Contains(t, string(raw), `attribute="rack"`)
	response = batchTestRequest(backend, "POST", "location/fetch?id=hawk-location-proof", `{}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	var location locationCreate
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &location))
	require.Equal(t, "100", location.Rules[0].Score)
	locationPayload = fmt.Sprintf(`{"id":"hawk-location-proof","resources":["hawk-clone-proof"],"rules":[{"score":"200","expressions":[{"attribute":"#uname","operation":"eq","value":%q,"kind":"uname"}]}]}`, options.Nodes[0])
	response = batchTestRequest(backend, "POST", "location/update", locationPayload)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = batchTestRequest(backend, "POST", "location/fetch?id=hawk-location-proof", `{}`)
	require.Equal(t, 200, response.Code)
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &location))
	require.Equal(t, "200", location.Rules[0].Score)
	response = batchTestRequest(backend, "POST", "location/update", `{"id":"hawk-location-rule-proof","resources":["hawk-clone-proof"],"discovery":"always","rules":[{"score":"50","operator":"or","expressions":[{"attribute":"rack","operation":"eq","value":"b","type":"string","kind":"attr-val"},{"attribute":"disk","operation":"defined","kind":"attr-def"}]}]}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = batchTestRequest(backend, "POST", "location/fetch?id=hawk-location-rule-proof", `{}`)
	require.Equal(t, 200, response.Code)
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &location))
	require.Equal(t, "50", location.Rules[0].Score)
	require.Equal(t, "or", location.Rules[0].Operator)
	require.Len(t, location.Rules[0].Expressions, 2)
	unchanged, err := os.ReadFile(baseline)
	require.NoError(t, err)
	require.Equal(t, original, unchanged)
}
