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

func sapSimpleStackPlusDefinition(t *testing.T) (resourceWizardDefinition, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/sap_simple_stack_plus.json")
	require.NoError(t, err)
	var d resourceWizardDefinition
	require.NoError(t, json.Unmarshal(data, &d))
	return d, data
}
func validSAPSimpleStackPlusParams() db2Params {
	var params db2Params
	_ = json.Unmarshal([]byte(`{"id": "grp_sap_NA0", "raid1": {"id": "raid1", "raidconf": "/etc/mdadm.conf", "raiddev": "/dev/md0"}, "lvm": {"id": "lvm", "vgname": "sapvg", "vg_access_mode": "system_id"}, "filesystem-sapmnt": {"id": "rsc_fs_NA0_sapmnt", "device": "/dev/sapvg/filesystem-sapmnt", "directory": "/sapmnt", "fstype": "xfs", "options": "noatime,barrier=0,data=writeback"}, "filesystem-usrsap": {"id": "rsc_fs_NA0_usrsap", "device": "/dev/sapvg/filesystem-usrsap", "directory": "/usr/sap", "fstype": "xfs", "options": "noatime,barrier=0,data=writeback"}, "sapdb": {"id": "rsc_sabdb_NA0", "SID": "NA0", "DBTYPE": "ADA"}, "virtual-ip-db": {"id": "rsc_ip_NA0_sapna0db", "ip": "172.17.2.54", "nic": "eth0", "cidr_netmask": "24"}, "filesystem-db": {"id": "rsc_fs_NA0_sapna0db", "device": "/dev/sapvg/filesystem-db", "directory": "/sapdb", "fstype": "xfs", "options": "noatime,barrier=0,data=writeback"}, "sapinstance-as": {"id": "rsc_sapinst_NA0_ASCS00_sapna0as", "InstanceName": "NA0_ASCS00_sapna0as", "START_PROFILE": "/usr/sap/NA0/SYS/profile/START_ASCS00_sapna0as", "AUTOMATIC_RECOVER": "true"}, "virtual-ip-as": {"id": "rsc_ip_NA0_sapna0as", "ip": "172.17.2.53", "nic": "eth0", "cidr_netmask": "24"}, "filesystem-as": {"id": "rsc_fs_NA0_sapna0as", "device": "/dev/sapvg/filesystem-as", "directory": "/usr/sap/NA0/ASCS00", "fstype": "xfs", "options": "noatime,barrier=0,data=writeback"}, "sapinstance-ci": {"id": "rsc_sapinst_NA0_DVEBMGS01_sapna0ci", "InstanceName": "NA0_DVEBMGS01_sapna0ci", "START_PROFILE": "/usr/sap/NA0/SYS/profile/START_DVEBMGS01_sapna0ci", "AUTOMATIC_RECOVER": "true"}, "virtual-ip-ci": {"id": "rsc_ip_NA0_sapna0ci", "ip": "172.17.2.55", "nic": "eth0", "cidr_netmask": "24"}, "filesystem-ci": {"id": "rsc_fs_NA0_sapna0ci", "device": "/dev/sapvg/filesystem-ci", "directory": "/usr/sap/NA0/DVEBMGS01", "fstype": "xfs", "options": "noatime,barrier=0,data=writeback"}}`), &params)
	return params
}
func TestSAPSimpleStackPlusForm(t *testing.T) {
	d, _ := sapSimpleStackPlusDefinition(t)
	view := buildNestedWizardView(d, nil, "sap-simple-stack-plus")
	require.Len(t, view.Steps, 14)
	require.Len(t, view.Advanced, 10)
	for _, step := range view.Steps {
		require.False(t, step.Optional)
	}
	req := httptest.NewRequest("POST", "/", nil)
	req.Form = url.Values{}
	for _, field := range view.Fields {
		if value := db2Value(validSAPSimpleStackPlusParams(), field.Path); value != "" {
			req.Form.Set(field.FieldID, value)
		}
	}
	params := db2FormParams(req, d, view)
	require.Equal(t, validSAPSimpleStackPlusParams(), params)
	require.NoError(t, validateDB2Params(params, view))
	for _, scope := range []string{"filesystem-sapmnt", "filesystem-usrsap", "filesystem-db", "filesystem-as", "filesystem-ci"} {
		invalid := validSAPSimpleStackPlusParams()
		invalid[scope].(map[string]any)["device"] = ""
		require.Error(t, validateDB2Params(invalid, view))
	}
	params = validSAPSimpleStackPlusParams()
	params["nodes"] = "other"
	require.Error(t, validateDB2Params(params, view))
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": d.Shortdesc, "WizardName": "sap-simple-stack-plus", "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}))
	for _, expected := range []string{`id="edit_wizard_sap-simple-stack-plus"`, `name="sap-simple-stack-plus.id"`, `name="sap-simple-stack-plus.lvm.vgname"`, `name="sap-simple-stack-plus.filesystem-sapmnt.device"`, `name="sap-simple-stack-plus.filesystem-usrsap.device"`, `name="sap-simple-stack-plus.filesystem-db.device"`, `name="sap-simple-stack-plus.filesystem-as.device"`, `name="sap-simple-stack-plus.filesystem-ci.device"`, `name="sap-simple-stack-plus.sapinstance-as.InstanceName"`, `name="sap-simple-stack-plus.sapinstance-ci.InstanceName"`, `name="sap-simple-stack-plus.virtual-ip-as.ip"`, `name="sap-simple-stack-plus.virtual-ip-ci.ip"`, `name="sap-simple-stack-plus.virtual-ip-db.ip"`, `value="/sapmnt"`, `id="step13"`} {
		require.Contains(t, body.String(), expected)
	}
	restored := buildNestedWizardView(d, validSAPSimpleStackPlusParams(), "sap-simple-stack-plus")
	require.Len(t, restored.Steps, 14)
	for _, group := range restored.Advanced {
		if strings.Contains(group.Prefix, "virtual-ip") {
			require.Equal(t, "eth0", group.Values["nic"])
		}
	}
}
func TestSAPSimpleStackPlusCommand(t *testing.T) {
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "sap-simple-stack-plus", op, validSAPSimpleStackPlusParams())
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"sap-simple-stack-plus"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var params db2Params
			require.NoError(t, json.Unmarshal(args[2], &params))
			require.Equal(t, validSAPSimpleStackPlusParams(), params)
		}
	}
}
func TestSAPSimpleStackPlusVerifyApplyAndBack(t *testing.T) {
	d, fixture := sapSimpleStackPlusDefinition(t)
	view := buildNestedWizardView(d, nil, "sap-simple-stack-plus")
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
				require.Equal(t, validSAPSimpleStackPlusParams(), params)
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
				if value := db2Value(validSAPSimpleStackPlusParams(), field.Path); value != "" {
					form.Set(field.FieldID, value)
				}
			}
			req := httptest.NewRequest("POST", "/cib/live/wizards/sap-simple-stack-plus", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			serveNestedWizard(response, req, run, sapSimpleStackPlusConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/sap-simple-stack-plus/submit"`)
			if failure == "verify" {
				require.Contains(t, response.Body.String(), `disabled`)
			} else {
				require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
			}
			back := httptest.NewRequest("GET", "/cib/live/wizards/sap-simple-stack-plus", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			var restored db2Params
			require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(back, "sap-simple-stack-plus"), &restored))
			require.Equal(t, validSAPSimpleStackPlusParams(), restored)
			operations = nil
			pa, _ := json.Marshal(validSAPSimpleStackPlusParams())
			req = httptest.NewRequest("POST", "/cib/live/wizards/sap-simple-stack-plus/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveNestedWizard(response, req, run, sapSimpleStackPlusConfig)
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
