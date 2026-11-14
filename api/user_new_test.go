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

func TestCreateACLTarget(t *testing.T) {
	for _, tt := range []struct {
		name, body, schema string
		status, writes     int
		failure            bool
	}{
		{name: "create", body: `{"id":"alice","roles":["monitor"]}`, status: 201, writes: 1},
		{name: "legacy", body: `{"id":"alice","roles":["monitor"]}`, schema: "pacemaker-1.2", status: 201, writes: 1},
		{name: "empty ID", body: `{"id":"","roles":["monitor"]}`, status: 422},
		{name: "invalid ID", body: `{"id":"bad name","roles":["monitor"]}`, status: 422},
		{name: "no roles", body: `{"id":"alice","roles":[]}`, status: 422},
		{name: "unknown role", body: `{"id":"alice","roles":["missing"]}`, status: 422},
		{name: "injected role", body: `{"id":"alice","roles":["monitor\nprimitive dummy"]}`, status: 422},
		{name: "existing target", body: `{"id":"a-user","roles":["monitor"]}`, status: 422},
		{name: "existing role", body: `{"id":"monitor","roles":["monitor"]}`, status: 422},
		{name: "existing resource", body: `{"id":"dummy","roles":["monitor"]}`, status: 422},
		{name: "failed command", body: `{"id":"alice","roles":["monitor"]}`, status: 422, writes: 1, failure: true},
		{name: "invalid JSON", body: `{`, status: 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writes := 0
			run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
				if name == "crm" {
					writes++
					require.Equal(t, []string{"configure"}, args)
					syntax := "acl_target"
					if tt.schema != "" {
						syntax = "acl_user"
					}
					require.Equal(t, syntax+" alice monitor\n", input)
					if tt.failure {
						return nil, fmt.Errorf("Permission denied")
					}
					return nil, nil
				}
				data, err := targetsTestRunner(ctx, name, args, input)
				if name == "cibadmin" {
					data = []byte(strings.Replace(string(data), "<acls>", `<resources><primitive id="dummy"/></resources><acls>`, 1))
					if tt.schema != "" {
						data = []byte(strings.Replace(string(data), "pacemaker-3.0", tt.schema, 1))
					}
				}
				return data, err
			}
			w := httptest.NewRecorder()
			createACLTarget(w, httptest.NewRequest("POST", "/", strings.NewReader(tt.body)), run)
			require.Equal(t, tt.status, w.Code, w.Body.String())
			require.Equal(t, tt.writes, writes)
			if tt.status == 201 {
				require.Contains(t, w.Body.String(), "ACL target created successfully")
			}
			if tt.failure {
				require.Contains(t, w.Body.String(), "Unable to create: Permission denied")
			}
		})
	}
	w := httptest.NewRecorder()
	createACLTarget(w, httptest.NewRequest("GET", "/", nil), targetsTestRunner)
	require.Equal(t, 405, w.Code)
}

func TestNewACLTargetPage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	w := httptest.NewRecorder()
	UsersHandler(w, httptest.NewRequest("GET", "/cib/live/users/new", nil))
	require.Equal(t, 200, w.Code)
	for _, text := range []string{"Create ACL Target", `name="user[id]"`, `name="user[roles][]"`, `multiple size="5"`, `name="submit"`, "/static/js/user_new.js", "ACL Target"} {
		require.Contains(t, w.Body.String(), text)
	}
	w = httptest.NewRecorder()
	UsersHandler(w, httptest.NewRequest("POST", "/cib/live/users/new", nil))
	require.Equal(t, 405, w.Code)
}
