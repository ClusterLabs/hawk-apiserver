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

func sapSimpleStackDefinition(t *testing.T) (resourceWizardDefinition, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/sap_simple_stack.json")
	require.NoError(t, err)
	var d resourceWizardDefinition
	require.NoError(t, json.Unmarshal(data, &d))
	return d, data
}
func validSAPSimpleStackParams() db2Params {
	var params db2Params
	_ = json.Unmarshal([]byte(`{"id": "grp_sap_NA0", "raid1": {"id": "raid1", "raidconf": "/etc/mdadm.conf", "raiddev": "/dev/md0"}, "lvm": {"id": "lvm", "vgname": "sapvg", "vg_access_mode": "system_id"}, "filesystem-sapmnt": {"id": "rsc_fs_NA0_sapmnt", "device": "/dev/sapvg/filesystem-sapmnt", "directory": "/sapmnt", "fstype": "xfs", "options": "noatime,barrier=0,data=writeback"}, "filesystem-usrsap": {"id": "rsc_fs_NA0_usrsap", "device": "/dev/sapvg/filesystem-usrsap", "directory": "/usr/sap", "fstype": "xfs", "options": "noatime,barrier=0,data=writeback"}, "sapdb": {"id": "rsc_sabdb_NA0", "SID": "NA0", "DBTYPE": "ADA"}, "virtual-ip-db": {"id": "rsc_ip_NA0_sapna0db", "ip": "172.17.2.54", "nic": "eth0", "cidr_netmask": "24"}, "filesystem-db": {"id": "rsc_fs_NA0_sapna0db", "device": "/dev/sapvg/filesystem-db", "directory": "/sapdb", "fstype": "xfs", "options": "noatime,barrier=0,data=writeback"}, "sapinstance-as": {"id": "rsc_sapinst_NA0_ASCS00_sapna0as", "InstanceName": "NA0_ASCS00_sapna0as", "START_PROFILE": "/usr/sap/NA0/SYS/profile/START_ASCS00_sapna0as", "AUTOMATIC_RECOVER": "true"}, "virtual-ip-as": {"id": "rsc_ip_NA0_sapna0as", "ip": "172.17.2.53", "nic": "eth0", "cidr_netmask": "24"}, "sapinstance-ci": {"id": "rsc_sapinst_NA0_DVEBMGS01_sapna0ci", "InstanceName": "NA0_DVEBMGS01_sapna0ci", "START_PROFILE": "/usr/sap/NA0/SYS/profile/START_DVEBMGS01_sapna0ci", "AUTOMATIC_RECOVER": "true"}, "virtual-ip-ci": {"id": "rsc_ip_NA0_sapna0ci", "ip": "172.17.2.55", "nic": "eth0", "cidr_netmask": "24"}}`), &params)
	return params
}
func TestSAPSimpleStackForm(t *testing.T) {
	d, _ := sapSimpleStackDefinition(t)
	view := buildNestedWizardView(d, nil, "sap-simple-stack")
	require.Len(t, view.Steps, 12)
	require.Len(t, view.Advanced, 8)
	for _, step := range view.Steps {
		require.False(t, step.Optional)
	}
	req := httptest.NewRequest("POST", "/", nil)
	req.Form = url.Values{}
	for _, field := range view.Fields {
		if value := db2Value(validSAPSimpleStackParams(), field.Path); value != "" {
			req.Form.Set(field.FieldID, value)
		}
	}
	params := db2FormParams(req, d, view)
	require.Equal(t, validSAPSimpleStackParams(), params)
	require.NoError(t, validateDB2Params(params, view))
	for _, scope := range []string{"filesystem-sapmnt", "filesystem-usrsap", "filesystem-db"} {
		invalid := validSAPSimpleStackParams()
		invalid[scope].(map[string]any)["device"] = ""
		require.Error(t, validateDB2Params(invalid, view))
	}
	params = validSAPSimpleStackParams()
	params["nodes"] = "other"
	require.Error(t, validateDB2Params(params, view))
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": d.Shortdesc, "WizardName": "sap-simple-stack", "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}))
	for _, expected := range []string{`id="edit_wizard_sap-simple-stack"`, `name="sap-simple-stack.id"`, `name="sap-simple-stack.lvm.vgname"`, `name="sap-simple-stack.filesystem-sapmnt.device"`, `name="sap-simple-stack.filesystem-usrsap.device"`, `name="sap-simple-stack.filesystem-db.device"`, `name="sap-simple-stack.sapinstance-as.InstanceName"`, `name="sap-simple-stack.sapinstance-ci.InstanceName"`, `name="sap-simple-stack.virtual-ip-as.ip"`, `name="sap-simple-stack.virtual-ip-ci.ip"`, `name="sap-simple-stack.virtual-ip-db.ip"`, `value="/sapmnt"`, `id="step11"`} {
		require.Contains(t, body.String(), expected)
	}
	restored := buildNestedWizardView(d, validSAPSimpleStackParams(), "sap-simple-stack")
	require.Len(t, restored.Steps, 12)
	for _, group := range restored.Advanced {
		if strings.Contains(group.Prefix, "virtual-ip") {
			require.Equal(t, "eth0", group.Values["nic"])
		}
	}
}
func TestSAPSimpleStackCommand(t *testing.T) {
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "sap-simple-stack", op, validSAPSimpleStackParams())
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"sap-simple-stack"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var params db2Params
			require.NoError(t, json.Unmarshal(args[2], &params))
			require.Equal(t, validSAPSimpleStackParams(), params)
		}
	}
}
func TestSAPSimpleStackVerifyApplyAndBack(t *testing.T) {
	d, fixture := sapSimpleStackDefinition(t)
	view := buildNestedWizardView(d, nil, "sap-simple-stack")
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
				require.Equal(t, validSAPSimpleStackParams(), params)
				if failure == op {
					return nil, fmt.Errorf("test command failed")
				}
				if op == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure SAP SimpleStack","text":"group grp_sap_NA0 raid1 lvm meta target-role=Stopped"}`), nil
				}
				if failure == "partial" {
					return nil, nil
				}
				return []byte(`{"rc":true}`), nil
			}
			form := url.Values{}
			for _, field := range view.Fields {
				if value := db2Value(validSAPSimpleStackParams(), field.Path); value != "" {
					form.Set(field.FieldID, value)
				}
			}
			req := httptest.NewRequest("POST", "/cib/live/wizards/sap-simple-stack", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			serveNestedWizard(response, req, run, sapSimpleStackConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/sap-simple-stack/submit"`)
			if failure == "verify" {
				require.Contains(t, response.Body.String(), `disabled`)
			} else {
				require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
			}
			back := httptest.NewRequest("GET", "/cib/live/wizards/sap-simple-stack", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			var restored db2Params
			require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(back, "sap-simple-stack"), &restored))
			require.Equal(t, validSAPSimpleStackParams(), restored)
			operations = nil
			pa, _ := json.Marshal(validSAPSimpleStackParams())
			req = httptest.NewRequest("POST", "/cib/live/wizards/sap-simple-stack/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveNestedWizard(response, req, run, sapSimpleStackConfig)
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
