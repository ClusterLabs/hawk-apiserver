package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneJSONMapsPrimitive(t *testing.T) {
	const payload = `{
		"id": "clone-1",
		"primitive": [{"id": "primitive-1"}],
		"meta_attributes": {"id": "clone-1-meta_attributes", "nvpair": []}
	}`

	var clone Clone
	require.NoError(t, json.NewDecoder(strings.NewReader(payload)).Decode(&clone))
	require.Len(t, clone.Primitives, 1)
	assert.Equal(t, "primitive-1", clone.Primitives[0].ID)
}

func installFakeCrm(t *testing.T, script string) {
	t.Helper()

	binDir := t.TempDir()
	crmPath := filepath.Join(binDir, "crm")
	require.NoError(t, os.WriteFile(crmPath, []byte(script), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func installFakeCibadmin(t *testing.T, script string) {
	t.Helper()

	binDir := t.TempDir()
	cibadminPath := filepath.Join(binDir, "cibadmin")
	require.NoError(t, os.WriteFile(cibadminPath, []byte(script), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestGetCrmStatus(t *testing.T) {
	t.Run("parses XML on success", func(t *testing.T) {
		installFakeCrm(t, `#!/bin/sh
printf '%s\n' '<crm_mon><nodes><node name="node1" id="1" online="true" maintenance="false"/></nodes></crm_mon>'
`)

		status, pacemakerRC, err := GetCrmStatus()

		require.NoError(t, err)
		assert.Zero(t, pacemakerRC)
		require.Len(t, status.Nodes, 1)
		assert.Equal(t, "node1", status.Nodes[0].Name)
		assert.True(t, status.Nodes[0].Online)
	})

	t.Run("extracts pacemaker return code from stderr", func(t *testing.T) {
		installFakeCrm(t, `#!/bin/sh
printf '%s\n' 'ERROR: status: Not connected' >&2
printf '%s\n' 'crm_mon: Connection to cluster failed: Connection refused (rc=102)' >&2
exit 1
`)

		status, pacemakerRC, err := GetCrmStatus()

		require.Error(t, err)
		assert.Empty(t, status.Nodes)
		assert.Equal(t, 102, pacemakerRC)
		assert.ErrorContains(t, err, "rc=102")
	})

	t.Run("uses 102 for not connected without explicit return code", func(t *testing.T) {
		installFakeCrm(t, `#!/bin/sh
printf '%s\n' 'ERROR: status: Not connected' >&2
exit 1
`)

		_, pacemakerRC, err := GetCrmStatus()

		require.Error(t, err)
		assert.Equal(t, 102, pacemakerRC)
	})

	t.Run("leaves return code unset for unrelated command errors", func(t *testing.T) {
		installFakeCrm(t, `#!/bin/sh
printf '%s\n' 'permission denied' >&2
exit 1
`)

		_, pacemakerRC, err := GetCrmStatus()

		require.Error(t, err)
		assert.Zero(t, pacemakerRC)
		assert.ErrorContains(t, err, "permission denied")
	})

	t.Run("returns malformed XML as an error", func(t *testing.T) {
		installFakeCrm(t, `#!/bin/sh
printf '%s\n' '<crm_mon>'
`)

		_, pacemakerRC, err := GetCrmStatus()

		require.Error(t, err)
		assert.Zero(t, pacemakerRC)
	})
}

func TestGetCIBReturnsPacemakerExitCode(t *testing.T) {
	installFakeCibadmin(t, `#!/bin/sh
exit 102
`)

	_, pacemakerRC, err := GetCIB()

	require.Error(t, err)
	assert.Equal(t, 102, pacemakerRC)
}

func TestEnrichCloneMetaAttributesAllowsMissingMetaAttributes(t *testing.T) {
	installFakeCibadmin(t, `#!/bin/sh
printf '%s\n' '<clone id="clone-1"><primitive id="primitive-1"/></clone>'
`)

	metadata := FullPrimitive_CrmResourceMetadata{MetaAttributes: GetCloneDefaults()}

	require.NoError(t, enrichCloneMetaAttributesWithCibValues(&metadata, "clone-1"))
}

func TestFetchFullCloneFromCibSkipsEnrichmentForEmptyID(t *testing.T) {
	installFakeCibadmin(t, `#!/bin/sh
exit 1
`)

	metadata, err := fetchFullCloneFromCib("")

	require.NoError(t, err)
	require.Equal(t, GetCloneDefaults(), metadata.MetaAttributes)
}

func TestCloneCreateRejectsMissingChildResource(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/cib/clone/create",
		strings.NewReader(`{"id":"clone-1","primitive":[]}`))
	response := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		CloneCreateHandler(response, request)
	})
	assert.Equal(t, http.StatusBadRequest, response.Code)
}
