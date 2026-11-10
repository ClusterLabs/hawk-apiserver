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

func raidLVMDefinition(t *testing.T) (resourceWizardDefinition, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/raid_lvm_tumbleweed.json")
	require.NoError(t, err)
	var d resourceWizardDefinition
	require.NoError(t, json.Unmarshal(data, &d))
	return d, data
}
func validRAIDLVMParams() db2Params {
	return db2Params{"id": "g-raid", "raid1": map[string]any{"id": "raid1", "raidconf": "/etc/mdadm.conf", "raiddev": "/dev/md0", "force_stop": "false"}, "lvm": map[string]any{"id": "lvm", "vgname": "vg1", "volgrpname": "vg1", "vg_access_mode": "system_id", "tag": "pacemaker"}}
}
func TestRAIDLVMForm(t *testing.T) {
	d, _ := raidLVMDefinition(t)
	view := buildNestedWizardView(d, nil, "raid-lvm")
	require.Len(t, view.Steps, 3)
	require.Len(t, view.Advanced, 2)
	require.Equal(t, "raid-lvm.raid1", view.Advanced[0].Prefix)
	require.Equal(t, "raid-lvm.lvm", view.Advanced[1].Prefix)
	for _, step := range view.Steps {
		require.False(t, step.Optional)
	}
	form := url.Values{}
	for _, field := range view.Fields {
		if field.Current != "" {
			form.Set(field.FieldID, field.Current)
		}
	}
	for key, value := range map[string]string{"raid-lvm.raid1.force_stop": "false", "raid-lvm.lvm.vgname": "vg1", "raid-lvm.lvm.volgrpname": "vg1", "raid-lvm.lvm.vg_access_mode": "system_id", "raid-lvm.lvm.tag": "pacemaker"} {
		form.Set(key, value)
	}
	req := httptest.NewRequest("POST", "/", nil)
	req.Form = form
	params := db2FormParams(req, d, view)
	require.Equal(t, validRAIDLVMParams(), params)
	require.NoError(t, validateDB2Params(params, view))
	params["lvm"].(map[string]any)["vgname"] = ""
	require.Error(t, validateDB2Params(params, view))
	params = validRAIDLVMParams()
	params["nodes"] = "other"
	require.Error(t, validateDB2Params(params, view))
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": d.Shortdesc, "WizardName": "raid-lvm", "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}))
	for _, expected := range []string{`id="edit_wizard_raid-lvm"`, `name="raid-lvm.id"`, `name="raid-lvm.raid1.raidconf"`, `name="raid-lvm.raid1.raiddev"`, `value="g-raid"`, `value="/etc/mdadm.conf"`, `value="/dev/md0"`, `name="raid-lvm.lvm.vgname"`, `id="step2"`} {
		require.Contains(t, body.String(), expected)
	}
	require.NotContains(t, body.String(), `id="step3"`)
	restored := buildNestedWizardView(d, validRAIDLVMParams(), "raid-lvm")
	require.Equal(t, "false", restored.Advanced[0].Values["force_stop"])
	require.Equal(t, "pacemaker", restored.Advanced[1].Values["tag"])
}
func TestRAIDLVMCommand(t *testing.T) {
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "raid-lvm", op, validRAIDLVMParams())
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"raid-lvm"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var params db2Params
			require.NoError(t, json.Unmarshal(args[2], &params))
			require.Equal(t, validRAIDLVMParams(), params)
		}
	}
}
func TestRAIDLVMVerifyApplyAndBack(t *testing.T) {
	d, fixture := raidLVMDefinition(t)
	view := buildNestedWizardView(d, nil, "raid-lvm")
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
				require.Equal(t, validRAIDLVMParams(), params)
				if failure == op {
					return nil, fmt.Errorf("test command failed")
				}
				if op == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure RAID and LVM","text":"group g-raid raid1 lvm meta target-role=stopped"}`), nil
				}
				if failure == "partial" {
					return nil, nil
				}
				return []byte(`{"rc":true}`), nil
			}
			form := url.Values{}
			for _, field := range view.Fields {
				if value := db2Value(validRAIDLVMParams(), field.Path); value != "" {
					form.Set(field.FieldID, value)
				}
			}
			req := httptest.NewRequest("POST", "/cib/live/wizards/raid-lvm", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			serveNestedWizard(response, req, run, raidLVMConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/raid-lvm/submit"`)
			if failure == "verify" {
				require.Contains(t, response.Body.String(), `disabled`)
			} else {
				require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
			}
			back := httptest.NewRequest("GET", "/cib/live/wizards/raid-lvm", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			var restored db2Params
			require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(back, "raid-lvm"), &restored))
			require.Equal(t, validRAIDLVMParams(), restored)
			operations = nil
			pa, _ := json.Marshal(validRAIDLVMParams())
			req = httptest.NewRequest("POST", "/cib/live/wizards/raid-lvm/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveNestedWizard(response, req, run, raidLVMConfig)
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
