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

func nfsServerDefinition(t *testing.T) (resourceWizardDefinition, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/nfsserver.json")
	require.NoError(t, err)
	var d resourceWizardDefinition
	require.NoError(t, json.Unmarshal(data, &d))
	addExportFSExamples(&d)
	data, err = json.Marshal(d)
	require.NoError(t, err)
	return d, data
}
func nfsServerForm(view db2View, root, vip bool) url.Values {
	values := url.Values{}
	for _, field := range view.Fields {
		if !field.Advanced && field.Current != "" {
			values.Set(field.FieldID, field.Current)
		}
	}
	values.Set("enable:nfsserver.rootfs", strconv.FormatBool(root))
	values.Set("enable:nfsserver.virtual-ip", strconv.FormatBool(vip))
	values.Set("nfsserver.rootfs.clientspec", "192.0.2.0/24")
	values.Set("nfsserver.exportfs.clientspec", "192.0.2.0/24")
	values.Set("nfsserver.exportfs.unlock_on_stop", "false")
	values.Set("nfsserver.virtual-ip.id", "nfs-ip")
	values.Set("nfsserver.virtual-ip.ip", "192.0.2.10")
	return values
}
func TestNFSServerStepsAndParameters(t *testing.T) {
	d, _ := nfsServerDefinition(t)
	view := buildNestedWizardView(d, nil, "nfsserver")
	require.Len(t, view.Steps, 4)
	require.Len(t, view.Advanced, 3)
	require.True(t, view.Steps[1].Optional)
	require.True(t, view.Steps[3].Optional)
	require.False(t, view.Steps[2].Optional)
	for _, root := range []bool{false, true} {
		for _, vip := range []bool{false, true} {
			req := httptest.NewRequest("POST", "/", nil)
			req.Form = nfsServerForm(view, root, vip)
			params := db2FormParams(req, d, view)
			require.NoError(t, validateDB2Params(params, view))
			_, hasRoot := params["rootfs"]
			_, hasVIP := params["virtual-ip"]
			require.Equal(t, root, hasRoot)
			require.Equal(t, vip, hasVIP)
			require.Equal(t, "base-fs", params["base-id"])
			require.Equal(t, "false", db2Value(params, []string{"exportfs", "unlock_on_stop"}))
			if root {
				require.Equal(t, "0", db2Value(params, []string{"rootfs", "fsid"}))
			}
			restored := buildNestedWizardView(d, params, "nfsserver")
			require.Equal(t, root, restored.Steps[1].Enabled)
			require.Equal(t, vip, restored.Steps[3].Enabled)
			params["exportfs"].(map[string]any)["clientspec"] = ""
			require.Error(t, validateDB2Params(params, view))
		}
	}
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": d.Shortdesc, "WizardName": "nfsserver", "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}))
	for _, expected := range []string{`id="edit_wizard_nfsserver"`, `name="nfsserver.base-id"`, `name="nfsserver.rootfs.clientspec"`, `name="nfsserver.exportfs.clientspec"`, `name="nfsserver.virtual-ip.ip"`, `name="enable:nfsserver.rootfs"`, `name="enable:nfsserver.virtual-ip"`, `value="0"`, `value="rw,crossmnt"`, `value="/srv/nfs/example"`, `placeholder="Example: 192.0.2.0/24"`, `id="step3"`} {
		require.Contains(t, body.String(), expected)
	}
}
func TestNFSServerCommand(t *testing.T) {
	params := db2Params{"base-id": "base-fs", "exportfs": map[string]any{"clientspec": "192.0.2.0/24"}}
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "nfsserver", op, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"nfsserver"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var actual db2Params
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}
func TestNFSServerVerifyApplyAndBack(t *testing.T) {
	d, fixture := nfsServerDefinition(t)
	view := buildNestedWizardView(d, nil, "nfsserver")
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, failure := range []string{"", "verify", "run", "partial"} {
		t.Run(failure, func(t *testing.T) {
			form := nfsServerForm(view, true, true)
			req := httptest.NewRequest("POST", "/cib/live/wizards/nfsserver", strings.NewReader(form.Encode()))
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
					return []byte("{\"name\":\"install\",\"shortdesc\":\"Install NFS Packages\"}\n{\"name\":\"cib\",\"text\":\"group g-nfs exportfs nfs-ip\"}"), nil
				}
				if failure == "partial" {
					return []byte(`{"rc":true}`), nil
				}
				return []byte("{\"rc\":true}\n{\"rc\":true}"), nil
			}
			response := httptest.NewRecorder()
			serveNestedWizard(response, req, run, nfsServerConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/nfsserver/submit"`)
			back := httptest.NewRequest("GET", "/cib/live/wizards/nfsserver", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			var restored db2Params
			require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(back, "nfsserver"), &restored))
			require.Equal(t, expected, restored)
			operations = nil
			pa, _ := json.Marshal(expected)
			req = httptest.NewRequest("POST", "/cib/live/wizards/nfsserver/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveNestedWizard(response, req, run, nfsServerConfig)
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
