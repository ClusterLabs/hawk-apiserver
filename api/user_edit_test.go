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

func TestUserEditAPI(t *testing.T) {
	for _, tt := range []struct {
		name, method, query, body string
		status, writes            int
		legacy, failure           bool
	}{
		{name: "load", method: "GET", query: "?id=a-user", status: 200},
		{name: "save", method: "POST", body: `{"id":"a-user","roles":["monitor"]}`, status: 200, writes: 1},
		{name: "legacy", method: "POST", body: `{"id":"old-user","roles":["monitor"]}`, status: 200, writes: 1, legacy: true},
		{name: "empty roles", method: "POST", body: `{"id":"a-user","roles":[]}`, status: 422},
		{name: "unknown role", method: "POST", body: `{"id":"a-user","roles":["gone"]}`, status: 422},
		{name: "missing", method: "GET", query: "?id=gone", status: 404},
		{name: "not target", method: "POST", body: `{"id":"monitor","roles":["monitor"]}`, status: 404},
		{name: "group", method: "POST", body: `{"id":"group1","roles":["monitor"]}`, status: 404},
		{name: "command failure", method: "POST", body: `{"id":"a-user","roles":["monitor"]}`, status: 422, writes: 1, failure: true},
		{name: "invalid JSON", method: "POST", body: `{`, status: 400},
		{name: "method", method: "DELETE", status: 405},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writes := 0
			run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
				if name == "crm" {
					writes++
					require.Equal(t, []string{"-F", "configure", "load", "update", "-"}, args)
					expected := "acl_target a-user monitor\n"
					if tt.legacy {
						expected = "acl_user old-user monitor\n"
					}
					require.Equal(t, expected, input)
					if tt.failure {
						return nil, fmt.Errorf("Permission denied")
					}
					return nil, nil
				}
				data, err := targetsTestRunner(ctx, name, args, input)
				if name == "cibadmin" && tt.legacy {
					data = []byte(strings.Replace(string(data), "pacemaker-3.0", "pacemaker-1.2", 1))
				}
				return data, err
			}
			w := httptest.NewRecorder()
			serveUserEdit(w, httptest.NewRequest(tt.method, "/"+tt.query, strings.NewReader(tt.body)), run)
			require.Equal(t, tt.status, w.Code, w.Body.String())
			require.Equal(t, tt.writes, writes)
			if tt.name == "load" {
				var data struct {
					Target  aclTarget
					Roles   []aclRole
					Enabled bool
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &data))
				require.Equal(t, "a-user", data.Target.ID)
				require.Equal(t, []string{"monitor", "operator"}, data.Target.Roles)
				require.Len(t, data.Roles, 1)
				require.False(t, data.Enabled)
			}
			if tt.name == "save" {
				require.Contains(t, w.Body.String(), "ACL target updated successfully")
			}
		})
	}
}

func TestUserEditPage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	w := httptest.NewRecorder()
	UsersHandler(w, httptest.NewRequest("GET", "/cib/live/users/a-user/edit", nil))
	require.Equal(t, 200, w.Code)
	for _, text := range []string{"Edit ACL Target", `data-target-id="a-user"`, `readonly value="a-user"`, `DeletePopup.js`, `name="user[roles][]"`} {
		require.Contains(t, w.Body.String(), text)
	}
	w = httptest.NewRecorder()
	UsersHandler(w, httptest.NewRequest("POST", "/cib/live/users/a-user/edit", nil))
	require.Equal(t, 405, w.Code)
}
