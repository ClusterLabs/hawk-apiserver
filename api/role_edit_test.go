package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoleUpdateCommand(t *testing.T) {
	role := aclRole{ID: "a-role", Rules: []aclRoleRule{{Right: "read", XPath: `//primitive[@id="dummy"]`}, {Right: "deny", Tag: "primitive", Ref: "dummy"}}}
	command, err := roleUpdateCommand(role)
	require.NoError(t, err)
	require.Equal(t, "role a-role read 'xpath://primitive[@id=\"dummy\"]' deny 'tag:primitive' 'ref:dummy'\n", command)
	for _, rule := range []aclRoleRule{
		{Right: "read"}, {Right: "other", Tag: "primitive"},
		{Right: "write", XPath: "//resources", Tag: "primitive"},
		{Right: "read", XPath: "//resources", Attribute: "id"},
		{Right: "read", Tag: "primitive\nprimitive bad"},
	} {
		_, err := roleUpdateCommand(aclRole{ID: "a-role", Rules: []aclRoleRule{rule}})
		require.Error(t, err)
	}
	_, err = roleUpdateCommand(aclRole{ID: "a-role"})
	require.ErrorContains(t, err, "At least one rule")
	_, err = roleUpdateCommand(aclRole{ID: "bad\nrole", Rules: role.Rules})
	require.Error(t, err)
}

func TestRoleEditAPI(t *testing.T) {
	for _, tt := range []struct {
		name, method, query, body string
		status, writes            int
		failure                   bool
	}{
		{name: "load", method: "GET", query: "?id=a-role", status: 200},
		{name: "save", method: "POST", body: `{"id":"a-role","rules":[{"right":"read","xpath":"//resources"}]}`, status: 200, writes: 1},
		{name: "missing", method: "GET", query: "?id=missing", status: 404},
		{name: "not a role", method: "POST", body: `{"id":"primitive1","rules":[{"right":"read","xpath":"//resources"}]}`, status: 404},
		{name: "empty rules", method: "POST", body: `{"id":"a-role","rules":[]}`, status: 422},
		{name: "invalid combination", method: "POST", body: `{"id":"a-role","rules":[{"right":"read","ref":"primitive1"}]}`, status: 422},
		{name: "command failure", method: "POST", body: `{"id":"a-role","rules":[{"right":"read","xpath":"//resources"}]}`, status: 422, writes: 1, failure: true},
		{name: "invalid json", method: "POST", body: `{`, status: 400},
		{name: "method", method: "DELETE", status: 405},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writes := 0
			run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
				if name == "crm" {
					writes++
					require.Equal(t, []string{"-F", "configure", "load", "update", "/dev/stdin"}, args)
					require.Equal(t, "role a-role read 'xpath://resources'\n", input)
					if tt.failure {
						return nil, fmt.Errorf("Permission denied")
					}
					return nil, nil
				}
				return rolesTestRunner(ctx, name, args, input)
			}
			response := httptest.NewRecorder()
			serveRoleEdit(response, httptest.NewRequest(tt.method, "/api/cib/acl/role"+tt.query, strings.NewReader(tt.body)), run)
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, tt.writes, writes)
			if tt.name == "load" {
				var data struct {
					Role    aclRole
					Enabled bool
					Legacy  bool
				}
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &data))
				require.Equal(t, "a-role", data.Role.ID)
				require.True(t, data.Enabled)
				require.False(t, data.Legacy)
			}
		})
	}
}

func TestRoleEditPage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	response := httptest.NewRecorder()
	RolesHandler(response, httptest.NewRequest("GET", "/cib/live/roles/a-role/edit", nil))
	require.Equal(t, 200, response.Code)
	for _, text := range []string{"Edit Role", `data-role-id="a-role"`, `name="role[id]"`, `name="submit"`, `name="revert"`, "/static/js/role_edit.js"} {
		require.Contains(t, response.Body.String(), text)
	}
}
