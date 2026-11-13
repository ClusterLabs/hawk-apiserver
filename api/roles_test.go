package api

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const aclRolesFixture = `<cib><configuration><crm_config><cluster_property_set><nvpair name="enable-acl" value="true"/></cluster_property_set></crm_config><resources><primitive id="primitive1"/></resources><acls><acl_role id="z-role"><acl_permission kind="read" xpath="//resources"/></acl_role><acl_role id="a-role"><acl_permission kind="write" object-type="primitive" reference="primitive1" attribute="description"/></acl_role><acl_target id="user"><role id="a-role"/></acl_target></acls></configuration></cib>`

func rolesTestRunner(_ context.Context, name string, args []string, input string) ([]byte, error) {
	if name == "pacemakerd" {
		return []byte("Pacemaker 3: acls cibsecrets corosync"), nil
	}
	if name == "cibadmin" {
		return []byte(aclRolesFixture), nil
	}
	return nil, fmt.Errorf("unexpected command: %s %v", name, args)
}
func TestLoadACLRoles(t *testing.T) {
	s, e := loadACLRoles(context.Background(), rolesTestRunner)
	require.NoError(t, e)
	require.True(t, s.Supported)
	require.True(t, s.Enabled)
	require.Len(t, s.Roles, 2)
	require.Equal(t, "a-role", s.Roles[0].ID)
	require.Equal(t, "write", s.Roles[0].Rules[0].Right)
	require.Equal(t, "primitive1", s.Roles[0].Rules[0].Ref)
	require.Equal(t, "//resources", s.Roles[1].Rules[0].XPath)
	old := func(_ context.Context, name string, args []string, input string) ([]byte, error) {
		if name == "pacemakerd" {
			return []byte("acls"), nil
		}
		return []byte(`<cib><configuration><acls><acl_role id="legacy"><read ref="primitive1" tag="primitive"/></acl_role></acls></configuration></cib>`), nil
	}
	s, e = loadACLRoles(context.Background(), old)
	require.NoError(t, e)
	require.False(t, s.Enabled)
	require.Equal(t, "read", s.Roles[0].Rules[0].Right)
	require.Equal(t, "primitive", s.Roles[0].Rules[0].Tag)
	s, e = loadACLRoles(context.Background(), func(context.Context, string, []string, string) ([]byte, error) { return []byte("corosync"), nil })
	require.NoError(t, e)
	require.False(t, s.Supported)
	require.Empty(t, s.Roles)
}
func TestACLRolesVersionFallback(t *testing.T) {
	for _, tt := range []struct {
		version   string
		supported bool
	}{
		{"2.0.5", false}, {"2.1.0", true}, {"2.1.9+build", true}, {"3.0.3+20260916.98b92cfb9b-1.1", true},
	} {
		t.Run(tt.version, func(t *testing.T) {
			run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
				if name == "pacemakerd" {
					return []byte("Pacemaker " + tt.version + " (Build: test) Supporting: corosync"), nil
				}
				return rolesTestRunner(ctx, name, args, input)
			}
			state, err := loadACLRoles(context.Background(), run)
			require.NoError(t, err)
			require.Equal(t, tt.supported, state.Supported)
			if tt.supported {
				require.Len(t, state.Roles, 2)
			}
		})
	}
}

func TestDeleteACLRole(t *testing.T) {
	for _, tt := range []struct {
		id            string
		failure       bool
		status, calls int
	}{{"a-role", false, 200, 1}, {"a-role", true, 422, 1}, {"primitive1", false, 404, 0}, {"missing", false, 404, 0}, {"", false, 400, 0}} {
		calls := 0
		run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
			if name == "crm" {
				calls++
				require.Equal(t, []string{"--force", "configure", "delete", tt.id}, args)
				if tt.failure {
					return nil, fmt.Errorf("Permission denied")
				}
				return nil, nil
			}
			return rolesTestRunner(ctx, name, args, input)
		}
		response := httptest.NewRecorder()
		deleteACLRole(response, httptest.NewRequest("POST", "/", strings.NewReader(fmt.Sprintf(`{"id":%q}`, tt.id))), run)
		require.Equal(t, tt.status, response.Code)
		require.Equal(t, tt.calls, calls)
	}
}
func TestRolesEndpoints(t *testing.T) {
	response := httptest.NewRecorder()
	serveRolesList(response, httptest.NewRequest("GET", "/", nil), rolesTestRunner)
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `"enabled":true`)
	require.Contains(t, response.Body.String(), `"id":"a-role"`)
	response = httptest.NewRecorder()
	serveRolesList(response, httptest.NewRequest("POST", "/", nil), rolesTestRunner)
	require.Equal(t, 405, response.Code)
	response = httptest.NewRecorder()
	deleteACLRole(response, httptest.NewRequest("GET", "/", nil), rolesTestRunner)
	require.Equal(t, 405, response.Code)
	cwd, e := os.Getwd()
	require.NoError(t, e)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	response = httptest.NewRecorder()
	RolesHandler(response, httptest.NewRequest("GET", "/cib/live/roles", nil))
	require.Equal(t, 200, response.Code)
	for _, value := range []string{"Access Control Lists", "/cib/live/roles/new", "/static/js/classes/ConfigTable.js", "/static/js/classes/DeletePopup.js"} {
		require.Contains(t, response.Body.String(), value)
	}
}
