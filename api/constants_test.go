package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRscDefaultsMatchCrmshResourceMetaAttributes(t *testing.T) {
	expectedNames := []string{
		"allow-migrate", "maintenance", "is-managed", "interval-origin",
		"migration-threshold", "priority", "multiple-active",
		"failure-timeout", "resource-stickiness", "target-role",
		"restart-type", "description", "remote-node", "requires",
		"provides", "remote-port", "remote-addr", "remote-connect-timeout",
		"critical", "allow-unhealthy-nodes", "container-attribute-target",
	}

	byName := make(map[string]MetaParameter, len(rscDefaults))
	for _, parameter := range GetRscDefaults() {
		require.NotEmpty(t, parameter.Name)
		require.NotEmpty(t, parameter.Content.Type, parameter.Name)
		_, duplicate := byName[parameter.Name]
		require.False(t, duplicate, "duplicate resource meta-attribute %q", parameter.Name)
		byName[parameter.Name] = parameter
	}

	require.Len(t, byName, len(expectedNames))
	for _, name := range expectedNames {
		require.Contains(t, byName, name)
	}

	for _, name := range []string{
		"interval-origin", "provides", "critical",
		"allow-unhealthy-nodes", "container-attribute-target",
	} {
		require.NotEmpty(t, byName[name].Longdesc, name)
	}
}

func TestAddedRscDefaultValueSchemas(t *testing.T) {
	byName := make(map[string]MetaParameter, len(rscDefaults))
	for _, parameter := range GetRscDefaults() {
		byName[parameter.Name] = parameter
	}

	require.Equal(t,
		[]string{"block", "stop_only", "stop_start", "stop_unexpected"},
		byName["multiple-active"].Content.PossibleValues)
	require.Equal(t,
		[]string{"Started", "Stopped", "Unpromoted", "Promoted"},
		byName["target-role"].Content.PossibleValues)
	require.Equal(t,
		[]string{"nothing", "quorum", "fencing", "unfencing"},
		byName["requires"].Content.PossibleValues)
	require.Equal(t, []string{"unfencing"}, byName["provides"].Content.PossibleValues)
}

func TestRoleValueSchemasUseCurrentTerminology(t *testing.T) {
	cloneByName := make(map[string]MetaParameter, len(cloneDefaults))
	for _, parameter := range GetCloneDefaults() {
		cloneByName[parameter.Name] = parameter
	}

	require.Equal(t,
		[]string{"Started", "Stopped", "Unpromoted", "Promoted"},
		cloneByName["target-role"].Content.PossibleValues)
	require.Equal(t,
		[]string{"Stopped", "Started", "Unpromoted", "Promoted"},
		actionDefaults.Role.Content.PossibleValues)
}
