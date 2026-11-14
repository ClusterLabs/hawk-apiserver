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

const aclTargetsFixture = `<cib validate-with="pacemaker-3.0"><configuration><acls><acl_role id="monitor"><acl_permission kind="read" xpath="//resources"/></acl_role><acl_target id="z-user"><role id="monitor"/></acl_target><acl_target id="a-user"><role_ref id="monitor"/><role id="operator"/></acl_target><acl_user id="old-user"><role_ref id="monitor"/><read xpath="//nodes"/></acl_user><acl_group id="group1"><role id="monitor"/></acl_group></acls></configuration></cib>`

func targetsTestRunner(_ context.Context, name string, args []string, input string) ([]byte, error) {
	if name == "pacemakerd" {
		return []byte("Pacemaker 3.0.3 Supporting: corosync"), nil
	}
	if name == "cibadmin" {
		return []byte(aclTargetsFixture), nil
	}
	return nil, fmt.Errorf("unexpected command: %s", name)
}

func TestLoadACLTargets(t *testing.T) {
	state, err := loadACLRoles(context.Background(), targetsTestRunner)
	require.NoError(t, err)
	require.True(t, state.Supported)
	require.False(t, state.Enabled)
	require.Len(t, state.Targets, 3)
	require.Equal(t, "a-user", state.Targets[0].ID)
	require.Equal(t, []string{"monitor", "operator"}, state.Targets[0].Roles)
	require.Equal(t, "old-user", state.Targets[1].ID)
	require.Equal(t, []string{"monitor"}, state.Targets[1].Roles)
	require.Len(t, state.Roles, 1)
}

func TestDeleteACLTarget(t *testing.T) {
	for _, tt := range []struct {
		id             string
		failure        bool
		status, writes int
	}{
		{"a-user", false, 200, 1}, {"old-user", false, 200, 1}, {"a-user", true, 422, 1},
		{"monitor", false, 404, 0}, {"group1", false, 404, 0}, {"missing", false, 404, 0}, {"", false, 400, 0},
	} {
		t.Run(tt.id+fmt.Sprint(tt.failure), func(t *testing.T) {
			writes := 0
			run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
				if name == "crm" {
					writes++
					require.Equal(t, []string{"--force", "configure", "delete", tt.id}, args)
					if tt.failure {
						return nil, fmt.Errorf("Permission denied")
					}
					return nil, nil
				}
				return targetsTestRunner(ctx, name, args, input)
			}
			w := httptest.NewRecorder()
			deleteACLRecord(w, httptest.NewRequest("POST", "/", strings.NewReader(fmt.Sprintf(`{"id":%q}`, tt.id))), run, true)
			require.Equal(t, tt.status, w.Code)
			require.Equal(t, tt.writes, writes)
			if tt.status == 200 {
				require.Contains(t, w.Body.String(), "ACL target deleted successfully")
			}
		})
	}
	// A target must not be deletable through the roles endpoint either.
	w := httptest.NewRecorder()
	deleteACLRole(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"id":"a-user"}`)), targetsTestRunner)
	require.Equal(t, 404, w.Code)
}

func TestUsersPage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	w := httptest.NewRecorder()
	UsersHandler(w, httptest.NewRequest("GET", "/cib/live/users", nil))
	require.Equal(t, 200, w.Code)
	for _, text := range []string{"Targets", "ACL Target", "/cib/live/users/new", "ACLOverview.js", "users.js", `data-config-table="users"`} {
		require.Contains(t, w.Body.String(), text)
	}
	w = httptest.NewRecorder()
	UsersHandler(w, httptest.NewRequest("PUT", "/cib/live/users", nil))
	require.Equal(t, 405, w.Code)
}
