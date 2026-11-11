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

func sapDBDefinition(t *testing.T) (resourceWizardDefinition, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/sap_db.json")
	require.NoError(t, err)
	var d resourceWizardDefinition
	require.NoError(t, json.Unmarshal(data, &d))
	return d, data
}
func validSAPDBParams() db2Params {
	return db2Params{"id": "grp_sapdb_NA0", "sapdb": map[string]any{"id": "rsc_sabdb_NA0", "SID": "NA0", "DBTYPE": "ADA"}, "virtual-ip": map[string]any{"id": "rsc_ip_NA0_sapna0db", "ip": "172.17.2.54", "cidr_netmask": "24", "nic": "eth0"}, "filesystem": map[string]any{"id": "rsc_fs_NA0_sapna0db", "device": "/dev/shared", "directory": "/sapdb", "fstype": "xfs", "options": "noatime,barrier=0,data=writeback", "fast_stop": "false"}}
}
func TestSAPDBForm(t *testing.T) {
	d, _ := sapDBDefinition(t)
	view := buildNestedWizardView(d, nil, "sap-db")
	require.Len(t, view.Steps, 4)
	require.Len(t, view.Advanced, 2)
	require.Equal(t, "sap-db.virtual-ip", view.Advanced[0].Prefix)
	require.Equal(t, "sap-db.filesystem", view.Advanced[1].Prefix)
	for _, step := range view.Steps {
		require.False(t, step.Optional)
	}
	req := httptest.NewRequest("POST", "/", nil)
	req.Form = url.Values{}
	for _, field := range view.Fields {
		if value := db2Value(validSAPDBParams(), field.Path); value != "" {
			req.Form.Set(field.FieldID, value)
		}
	}
	params := db2FormParams(req, d, view)
	require.Equal(t, validSAPDBParams(), params)
	require.NoError(t, validateDB2Params(params, view))
	params["filesystem"].(map[string]any)["device"] = ""
	require.Error(t, validateDB2Params(params, view))
	params = validSAPDBParams()
	params["sapdb"].(map[string]any)["SID"] = ""
	require.Error(t, validateDB2Params(params, view))
	params = validSAPDBParams()
	params["sapdb"].(map[string]any)["DBTYPE"] = ""
	require.Error(t, validateDB2Params(params, view))
	params = validSAPDBParams()
	params["nodes"] = "other"
	require.Error(t, validateDB2Params(params, view))
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": d.Shortdesc, "WizardName": "sap-db", "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}))
	for _, expected := range []string{`id="edit_wizard_sap-db"`, `name="sap-db.id"`, `name="sap-db.sapdb.SID"`, `name="sap-db.sapdb.DBTYPE"`, `value="NA0"`, `value="ADA"`, `name="sap-db.virtual-ip.ip"`, `value="172.17.2.54"`, `name="sap-db.filesystem.device"`, `value="/sapdb"`, `id="step3"`} {
		require.Contains(t, body.String(), expected)
	}
	restored := buildNestedWizardView(d, validSAPDBParams(), "sap-db")
	require.Equal(t, "eth0", restored.Advanced[0].Values["nic"])
	require.Equal(t, "false", restored.Advanced[1].Values["fast_stop"])
}
func TestSAPDBCommand(t *testing.T) {
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "sap-db", op, validSAPDBParams())
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"sap-db"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var params db2Params
			require.NoError(t, json.Unmarshal(args[2], &params))
			require.Equal(t, validSAPDBParams(), params)
		}
	}
}
func TestSAPDBVerifyApplyAndBack(t *testing.T) {
	d, fixture := sapDBDefinition(t)
	view := buildNestedWizardView(d, nil, "sap-db")
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
				require.Equal(t, validSAPDBParams(), params)
				if failure == op {
					return nil, fmt.Errorf("test command failed")
				}
				if op == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure SAP Database","text":"group grp_sapdb_NA0 rsc_ip_NA0_sapna0db rsc_fs_NA0_sapna0db rsc_sabdb_NA0 meta target-role=Stopped"}`), nil
				}
				if failure == "partial" {
					return nil, nil
				}
				return []byte(`{"rc":true}`), nil
			}
			form := url.Values{}
			for _, field := range view.Fields {
				if value := db2Value(validSAPDBParams(), field.Path); value != "" {
					form.Set(field.FieldID, value)
				}
			}
			req := httptest.NewRequest("POST", "/cib/live/wizards/sap-db", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			serveNestedWizard(response, req, run, sapDBConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/sap-db/submit"`)
			if failure == "verify" {
				require.Contains(t, response.Body.String(), `disabled`)
			} else {
				require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
			}
			back := httptest.NewRequest("GET", "/cib/live/wizards/sap-db", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			var restored db2Params
			require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(back, "sap-db"), &restored))
			require.Equal(t, validSAPDBParams(), restored)
			operations = nil
			pa, _ := json.Marshal(validSAPDBParams())
			req = httptest.NewRequest("POST", "/cib/live/wizards/sap-db/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveNestedWizard(response, req, run, sapDBConfig)
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
