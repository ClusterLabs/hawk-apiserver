package api

import (
	"bytes"
	"context"
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
	backend := &batchBackend{prepare: prepareBatchDirectory}
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
	_, err = backend.run(ctx, false, "crm", []string{"-F", "configure", "load", "update", "/dev/stdin"}, "primitive hawk-color-proof ocf:pacemaker:Dummy\n")
	require.NoError(t, err)
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
	unchanged, err := os.ReadFile(baseline)
	require.NoError(t, err)
	require.Equal(t, original, unchanged)
}
