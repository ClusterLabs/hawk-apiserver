package api

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateACLRole(t *testing.T) {
	for _, tt := range []struct {
		name, body     string
		status, writes int
		failure        bool
	}{
		{name: "create", body: `{"id":"monitor","rules":[{"right":"read","xpath":"//resources"},{"right":"deny","tag":"primitive","ref":"private"}]}`, status: 201, writes: 1},
		{name: "empty ID", body: `{"id":"","rules":[{"right":"read","xpath":"//resources"}]}`, status: 422},
		{name: "invalid ID", body: `{"id":"bad name","rules":[{"right":"read","xpath":"//resources"}]}`, status: 422},
		{name: "no rules", body: `{"id":"monitor","rules":[]}`, status: 422},
		{name: "no selector", body: `{"id":"monitor","rules":[{"right":"read"}]}`, status: 422},
		{name: "conflict", body: `{"id":"monitor","rules":[{"right":"read","xpath":"//resources","tag":"primitive"}]}`, status: 422},
		{name: "duplicate role", body: `{"id":"a-role","rules":[{"right":"read","xpath":"//resources"}]}`, status: 422},
		{name: "duplicate primitive", body: `{"id":"primitive1","rules":[{"right":"read","xpath":"//resources"}]}`, status: 422},
		{name: "command failure", body: `{"id":"monitor","rules":[{"right":"read","xpath":"//resources"},{"right":"deny","tag":"primitive","ref":"private"}]}`, status: 422, writes: 1, failure: true},
		{name: "malformed", body: `{`, status: 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writes := 0
			run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
				if name == "crm" {
					writes++
					require.Equal(t, []string{"configure"}, args)
					require.Equal(t, "role monitor read 'xpath://resources' deny 'tag:primitive' 'ref:private'\n", input)
					if tt.failure {
						return nil, fmt.Errorf("Permission denied")
					}
					return nil, nil
				}
				return rolesTestRunner(ctx, name, args, input)
			}
			w := httptest.NewRecorder()
			createACLRole(w, httptest.NewRequest("POST", "/", strings.NewReader(tt.body)), run)
			require.Equal(t, tt.status, w.Code, w.Body.String())
			require.Equal(t, tt.writes, writes)
			if tt.status == 201 {
				require.Contains(t, w.Body.String(), "Role created successfully")
			}
			if tt.failure {
				require.Contains(t, w.Body.String(), "Unable to create: Permission denied")
			}
		})
	}
	w := httptest.NewRecorder()
	createACLRole(w, httptest.NewRequest("GET", "/", nil), nil)
	require.Equal(t, 405, w.Code)
	// Ruby's Record validation permits numeric-leading IDs and periods.
	_, err := roleUpdateCommand(aclRole{ID: "123.monitor", Rules: []aclRoleRule{{Right: "read", XPath: "//resources"}}})
	require.NoError(t, err)
}

func TestNewRolePage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	w := httptest.NewRecorder()
	RolesHandler(w, httptest.NewRequest("GET", "/cib/live/roles/new", nil))
	require.Equal(t, 200, w.Code)
	for _, text := range []string{"Create Role", `id="new_role"`, `data-create-mode="true"`, `name="role[id]"`, `name="submit"`, `href="/cib/live/roles"`} {
		require.Contains(t, w.Body.String(), text)
	}
	require.NotContains(t, w.Body.String(), `id="delete-role"`)
	require.NotContains(t, w.Body.String(), `name="revert"`)
	w = httptest.NewRecorder()
	RolesHandler(w, httptest.NewRequest("POST", "/cib/live/roles/new", nil))
	require.Equal(t, 405, w.Code)
}
