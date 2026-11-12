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

func cryptctlDefinition(t *testing.T) (resourceWizardDefinition, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/cryptctl.json")
	require.NoError(t, err)
	var d resourceWizardDefinition
	require.NoError(t, json.Unmarshal(data, &d))
	return d, data
}
func validCryptctlParams() db2Params {
	return db2Params{"id": "cryptctl", "cert-path": "/etc/cryptctl/server.crt", "cert-key-path": "/etc/cryptctl/server.key", "virtual-ip": map[string]any{"id": "cryptctl-vip", "ip": "192.0.2.10", "cidr_netmask": "24", "nic": "eth0"}, "filesystem": map[string]any{"id": "cryptctl-filesystem", "device": "/dev/shared", "directory": "/var/lib/cryptctl/keydb", "fstype": "xfs", "fast_stop": "false"}, "cryptctl-server": map[string]any{"id": "cryptctl-server-service"}}
}
func TestCryptctlForm(t *testing.T) {
	d, _ := cryptctlDefinition(t)
	view := buildNestedWizardView(d, nil, "cryptctl")
	require.Len(t, view.Steps, 4)
	require.Len(t, view.Advanced, 2)
	for _, step := range view.Steps {
		require.False(t, step.Optional)
	}
	req := httptest.NewRequest("POST", "/", nil)
	req.Form = url.Values{}
	for _, field := range view.Fields {
		if value := db2Value(validCryptctlParams(), field.Path); value != "" {
			req.Form.Set(field.FieldID, value)
		}
	}
	params := db2FormParams(req, d, view)
	require.Equal(t, validCryptctlParams(), params)
	require.NoError(t, validateDB2Params(params, view))
	for _, name := range []string{"cert-path", "cert-key-path"} {
		invalid := validCryptctlParams()
		delete(invalid, name)
		require.Error(t, validateDB2Params(invalid, view))
	}
	params["filesystem"].(map[string]any)["device"] = ""
	require.Error(t, validateDB2Params(params, view))
	params = validCryptctlParams()
	params["nodes"] = "other"
	require.Error(t, validateDB2Params(params, view))
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": d.Shortdesc, "WizardName": "cryptctl", "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}))
	for _, expected := range []string{`id="edit_wizard_cryptctl"`, `name="cryptctl.id"`, `name="cryptctl.cert-path"`, `name="cryptctl.cert-key-path"`, `name="cryptctl.virtual-ip.ip"`, `name="cryptctl.filesystem.device"`, `name="cryptctl.cryptctl-server.id"`, `value="/var/lib/cryptctl/keydb"`, `value="cryptctl-server-service"`, `id="step3"`} {
		require.Contains(t, body.String(), expected)
	}
	require.NotContains(t, body.String(), `value="{{id}}-vip"`)
	restored := buildNestedWizardView(d, validCryptctlParams(), "cryptctl")
	require.Equal(t, "eth0", restored.Advanced[0].Values["nic"])
	require.Equal(t, "false", restored.Advanced[1].Values["fast_stop"])
}
func TestCryptctlCommand(t *testing.T) {
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "cryptctl", op, validCryptctlParams())
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"cryptctl"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var params db2Params
			require.NoError(t, json.Unmarshal(args[2], &params))
			require.Equal(t, validCryptctlParams(), params)
		}
	}
}
func TestCryptctlVerifyApplyAndBack(t *testing.T) {
	d, fixture := cryptctlDefinition(t)
	view := buildNestedWizardView(d, nil, "cryptctl")
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, failure := range []string{"", "verify", "run", "partial"} {
		t.Run(failure, func(t *testing.T) {
			operations := []string{}
			run := func(_ context.Context, op string, params db2Params) ([]byte, error) {
				operations = append(operations, op)
				if op == "show" {
					return fixture, nil
				}
				require.Equal(t, validCryptctlParams(), params)
				if failure == op {
					return nil, fmt.Errorf("test command failed")
				}
				if op == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure cryptctl","text":"group group-cryptctl cryptctl-vip cryptctl-filesystem cryptctl-server-service"}`), nil
				}
				if failure == "partial" {
					return nil, nil
				}
				return []byte(`{"rc":true}`), nil
			}
			form := url.Values{}
			for _, field := range view.Fields {
				if value := db2Value(validCryptctlParams(), field.Path); value != "" {
					form.Set(field.FieldID, value)
				}
			}
			req := httptest.NewRequest("POST", "/cib/live/wizards/cryptctl", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			serveNestedWizard(response, req, run, cryptctlConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/cryptctl/submit"`)
			if failure == "verify" {
				require.Contains(t, response.Body.String(), `disabled`)
			} else {
				require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
			}
			back := httptest.NewRequest("GET", "/cib/live/wizards/cryptctl", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			var restored db2Params
			require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(back, "cryptctl"), &restored))
			require.Equal(t, validCryptctlParams(), restored)
			operations = nil
			pa, _ := json.Marshal(validCryptctlParams())
			req = httptest.NewRequest("POST", "/cib/live/wizards/cryptctl/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveNestedWizard(response, req, run, cryptctlConfig)
			status := 200
			if failure != "" {
				status = 422
			}
			require.Equal(t, status, response.Code)
			expected := []string{"show", "verify", "run"}
			if failure == "verify" {
				expected = expected[:2]
			}
			require.Equal(t, expected, operations)
		})
	}
}
