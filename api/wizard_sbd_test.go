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

func sbdDefinition(t *testing.T) (resourceWizardDefinition, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/sbd.json")
	require.NoError(t, err)
	var d resourceWizardDefinition
	require.NoError(t, json.Unmarshal(data, &d))
	return d, data
}
func sbdForm(enabled bool) url.Values {
	return url.Values{"sbd.id": {"sbd-fencing"}, "enable:sbd.sbd-device": {strconv.FormatBool(enabled)}, "sbd.sbd-device.device": {"/dev/disk/by-id/test-device"}, "sbd.sbd-device.watchdog": {"/dev/watchdog"}}
}
func TestSBDForm(t *testing.T) {
	d, _ := sbdDefinition(t)
	view := buildNestedWizardView(d, nil, "sbd")
	require.Len(t, view.Steps, 2)
	require.Empty(t, view.Advanced)
	require.True(t, view.Steps[1].Optional)
	require.False(t, view.Steps[1].Enabled)
	for _, enabled := range []bool{false, true} {
		req := httptest.NewRequest("POST", "/", nil)
		req.Form = sbdForm(enabled)
		params := db2FormParams(req, d, view)
		require.NoError(t, validateDB2Params(params, view))
		_, present := params["sbd-device"]
		require.Equal(t, enabled, present)
		restored := buildNestedWizardView(d, params, "sbd")
		require.Equal(t, enabled, restored.Steps[1].Enabled)
		if enabled {
			params["sbd-device"].(map[string]any)["device"] = ""
			require.Error(t, validateDB2Params(params, view))
		}
	}
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": d.Shortdesc, "WizardName": "sbd", "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}))
	for _, expected := range []string{`id="edit_wizard_sbd"`, `name="sbd.id"`, `value="sbd-fencing"`, `name="enable:sbd.sbd-device"`, `value="false" selected`, `name="sbd.sbd-device.device"`, `placeholder="Example: /dev/disk/by-id/..."`, `value="/dev/watchdog"`, `id="step1"`} {
		require.Contains(t, body.String(), expected)
	}
	require.NotContains(t, body.String(), `id="step2"`)
}
func TestSBDCommand(t *testing.T) {
	params := db2Params{"id": "sbd-fencing"}
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "sbd", op, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"sbd"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var actual db2Params
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}
func TestSBDVerifyApplyAndBack(t *testing.T) {
	d, fixture := sbdDefinition(t)
	view := buildNestedWizardView(d, nil, "sbd")
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, enabled := range []bool{false, true} {
		for _, failure := range []string{"", "verify", "run", "partial"} {
			t.Run(fmt.Sprintf("%t/%s", enabled, failure), func(t *testing.T) {
				req := httptest.NewRequest("POST", "/cib/live/wizards/sbd", strings.NewReader(sbdForm(enabled).Encode()))
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
						return nil, fmt.Errorf("test execution failed")
					}
					if op == "verify" {
						preview := `{"name":"cib","text":"primitive sbd-fencing stonith:fence_sbd pcmk_delay_max=30s\nproperty stonith-enabled=true"}`
						if enabled {
							preview = "{\"name\":\"call\",\"shortdesc\":\"Create SBD device\"}\n" + preview
						}
						return []byte(preview), nil
					}
					if failure == "partial" {
						return nil, nil
					}
					result := `{"rc":true}`
					if enabled {
						result += "\n" + result
					}
					return []byte(result), nil
				}
				response := httptest.NewRecorder()
				serveNestedWizard(response, req, run, sbdConfig)
				require.Equal(t, 200, response.Code)
				require.Equal(t, []string{"show", "verify"}, operations)
				require.Contains(t, response.Body.String(), `action="/cib/live/wizards/sbd/submit"`)
				back := httptest.NewRequest("GET", "/cib/live/wizards/sbd", nil)
				for _, cookie := range response.Result().Cookies() {
					back.AddCookie(cookie)
				}
				var restored db2Params
				require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(back, "sbd"), &restored))
				require.Equal(t, expected, restored)
				operations = nil
				pa, _ := json.Marshal(expected)
				req = httptest.NewRequest("POST", "/cib/live/wizards/sbd/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response = httptest.NewRecorder()
				serveNestedWizard(response, req, run, sbdConfig)
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
}
