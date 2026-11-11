package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func nginxDefinition(t *testing.T) (resourceWizardDefinition, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/nginx.json")
	require.NoError(t, err)
	var d resourceWizardDefinition
	require.NoError(t, json.Unmarshal(data, &d))
	data, err = json.Marshal(d)
	require.NoError(t, err)
	return d, data
}
func nginxForm(view db2View, filesystem, database bool) url.Values {
	values := url.Values{}
	for _, f := range view.Fields {
		if !f.Advanced && f.Current != "" {
			values.Set(f.FieldID, f.Current)
		}
	}
	values.Set("nginx.id", "web")
	values.Set("nginx.configfile", "/etc/nginx/nginx.conf")
	values.Set("nginx.virtual-ip.ip", "192.0.2.10")
	values.Set("nginx.virtual-ip.id", "web-vip")
	values.Set("enable:nginx.filesystem", strconv.FormatBool(filesystem))
	values.Set("enable:nginx.database", strconv.FormatBool(database))
	values.Set("nginx.filesystem.id", "web-fs")
	values.Set("nginx.filesystem.device", "/dev/shared")
	values.Set("nginx.filesystem.directory", "/srv/www")
	values.Set("nginx.filesystem.fstype", "xfs")
	values.Set("nginx.database.id", "web-db")
	values.Set("nginx.database.test_passwd", "test-secret")
	return values
}
func TestNginxStepsAndParameters(t *testing.T) {
	d, _ := nginxDefinition(t)
	view := buildNestedWizardView(d, nil, "nginx")
	require.Len(t, view.Steps, 4)
	require.Len(t, view.Advanced, 4)
	require.True(t, view.Steps[2].Optional)
	require.True(t, view.Steps[3].Optional)
	require.False(t, view.Steps[1].Optional)
	for _, fs := range []bool{false, true} {
		for _, db := range []bool{false, true} {
			req := httptest.NewRequest("POST", "/", nil)
			req.Form = nginxForm(view, fs, db)
			params := db2FormParams(req, d, view)
			require.NoError(t, validateDB2Params(params, view))
			_, hasFS := params["filesystem"]
			_, hasDB := params["database"]
			require.Equal(t, fs, hasFS)
			require.Equal(t, db, hasDB)
			require.Equal(t, "false", params["install"])
			require.Equal(t, "/etc/nginx/nginx.conf", params["configfile"])
			restored := buildNestedWizardView(d, params, "nginx")
			require.Equal(t, fs, restored.Steps[2].Enabled)
			require.Equal(t, db, restored.Steps[3].Enabled)
			params["virtual-ip"].(map[string]any)["ip"] = ""
			require.Error(t, validateDB2Params(params, view))
		}
	}
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": d.Shortdesc, "WizardName": "nginx", "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}))
	for _, expected := range []string{`id="edit_wizard_nginx"`, `name="nginx.id"`, `name="nginx.install"`, `value="false" selected`, `name="nginx.virtual-ip.ip"`, `name="nginx.filesystem.device"`, `name="nginx.database.id"`, `name="enable:nginx.filesystem"`, `name="enable:nginx.database"`, `id="step3"`} {
		require.Contains(t, body.String(), expected)
	}
	require.NotContains(t, body.String(), `value="{{id}}-vip"`)
}
func TestNginxCommand(t *testing.T) {
	params := db2Params{"id": "web", "virtual-ip": map[string]any{"id": "web-vip", "ip": "192.0.2.10"}}
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "nginx", op, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"nginx"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var actual db2Params
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}
func TestNginxVerifyApplyAndBack(t *testing.T) {
	d, fixture := nginxDefinition(t)
	view := buildNestedWizardView(d, nil, "nginx")
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, failure := range []string{"", "verify", "run", "partial"} {
		t.Run(failure, func(t *testing.T) {
			form := nginxForm(view, true, true)
			req := httptest.NewRequest("POST", "/cib/live/wizards/nginx", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			require.NoError(t, req.ParseForm())
			expected := db2FormParams(req, d, view)
			operations := []string{}
			run := func(_ context.Context, op string, params db2Params) ([]byte, error) {
				operations = append(operations, op)
				if op == "show" {
					return fixture, nil
				}
				require.Equal(t, expected, params)
				if failure == op {
					return nil, fmt.Errorf("test command failed")
				}
				if op == "verify" {
					return []byte("{\"name\":\"install\",\"shortdesc\":\"Install Nginx package\"}\n{\"name\":\"cib\",\"text\":\"group g-web web-fs web-db web-vip web\"}"), nil
				}
				if failure == "partial" {
					return []byte(`{"rc":true}`), nil
				}
				return []byte("{\"rc\":true}\n{\"rc\":true}"), nil
			}
			response := httptest.NewRecorder()
			serveNestedWizard(response, req, run, nginxConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/nginx/submit"`)
			back := httptest.NewRequest("GET", "/cib/live/wizards/nginx", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			var restored db2Params
			require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(back, "nginx"), &restored))
			require.Equal(t, expected, restored)
			operations = nil
			pa, _ := json.Marshal(expected)
			req = httptest.NewRequest("POST", "/cib/live/wizards/nginx/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveNestedWizard(response, req, run, nginxConfig)
			status := 200
			if failure != "" {
				status = 422
			}
			require.Equal(t, status, response.Code)
			expectedOps := []string{"show", "verify", "run"}
			if failure == "verify" {
				expectedOps = expectedOps[:2]
			}
			require.Equal(t, expectedOps, operations)
		})
	}
}
