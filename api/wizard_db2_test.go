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

const db2Fixture = `{"name":"db2","shortdesc":"IBM DB2 Database","longdesc":"Configure DB2 with a Virtual IP and a filesystem. The group is stopped initially.","steps":[{"required":true,"shortdesc":"DB2 settings","longdesc":"Resource Agent for DB2.\n\nStandard and HADR modes.","parameters":[{"name":"id","type":"string","value":"db2-database","shortdesc":"DB2 Resource ID"},{"name":"instance","type":"string","value":"db2inst1"},{"name":"dblist","type":"string","advanced":true},{"name":"skip_basic_sql_health_check","type":"boolean","advanced":true,"example":"false"}]},{"name":"virtual-ip","required":true,"shortdesc":"Virtual IP","longdesc":"Virtual IP description.","steps":[{"shortdesc":"IPaddr2 agent","parameters":[{"name":"id","type":"resource","value":"db2-virtual-ip"},{"name":"ip","type":"ip_address","required":true},{"name":"cidr_netmask","type":"integer"},{"name":"nic","type":"string","advanced":true}]}]},{"name":"filesystem","required":true,"shortdesc":"File System (mount point)","steps":[{"shortdesc":"Filesystem agent","parameters":[{"name":"id","type":"resource","value":"db2-fs"},{"name":"device","type":"string","required":true},{"name":"directory","type":"string","value":"/db2/db2inst1"},{"name":"fstype","type":"string","value":"xfs"},{"name":"options","type":"string"},{"name":"fast_stop","type":"boolean","advanced":true,"example":"no"}]}]}]}`

func db2TestDefinition(t *testing.T) resourceWizardDefinition {
	t.Helper()
	d, err := loadDB2Definition(context.Background(), func(_ context.Context, op string, _ db2Params) ([]byte, error) {
		require.Equal(t, "show", op)
		return []byte(db2Fixture), nil
	})
	require.NoError(t, err)
	return d
}
func validDB2Params() db2Params {
	return db2Params{"id": "db2-database", "instance": "db2inst1", "virtual-ip": map[string]any{"id": "db2-virtual-ip", "ip": "192.0.2.10", "cidr_netmask": "24", "nic": "eth0"}, "filesystem": map[string]any{"id": "db2-fs", "device": "/dev/example", "directory": "/db2/db2inst1", "fstype": "xfs"}}
}
func TestDB2ViewAndNestedForm(t *testing.T) {
	definition := db2TestDefinition(t)
	view := buildDB2View(definition, nil)
	require.Len(t, view.Steps, 3)
	require.Equal(t, "Virtual IP", view.Steps[1].Title)
	require.Len(t, view.Advanced, 3)
	require.Equal(t, "db2", view.Advanced[0].Prefix)
	require.Equal(t, "db2.virtual-ip", view.Advanced[1].Prefix)
	require.Equal(t, "db2.filesystem", view.Advanced[2].Prefix)
	require.Equal(t, "false", view.Advanced[2].Parameters[0].Example)
	fields := map[string]db2Field{}
	for _, f := range view.Fields {
		fields[f.FieldID] = f
	}
	require.Equal(t, "db2-database", fields["db2.id"].Current)
	require.Equal(t, "db2inst1", fields["db2.instance"].Current)
	require.Equal(t, "db2-virtual-ip", fields["db2.virtual-ip.id"].Current)
	require.Equal(t, "xfs", fields["db2.filesystem.fstype"].Current)
	require.Equal(t, []string{"virtual-ip", "ip"}, fields["db2.virtual-ip.ip"].Path)
	req := httptest.NewRequest("POST", "/", nil)
	req.Form = url.Values{}
	for name, f := range fields {
		if !f.Advanced && f.Current != "" {
			req.Form.Set(name, f.Current)
		}
	}
	req.Form.Set("db2.virtual-ip.ip", "192.0.2.10")
	req.Form.Set("db2.virtual-ip.cidr_netmask", "24")
	req.Form.Set("db2.virtual-ip.nic", "eth0")
	req.Form.Set("db2.filesystem.device", "/dev/example")
	params := db2FormParams(req, definition, view)
	require.Equal(t, validDB2Params(), params)
	require.NoError(t, validateDB2Params(params, view))
	restored := buildDB2View(definition, params)
	require.Equal(t, "eth0", restored.Advanced[1].Values["nic"])
}
func TestDB2RejectsInvalidParams(t *testing.T) {
	view := buildDB2View(db2TestDefinition(t), nil)
	for _, change := range []func(db2Params){
		func(p db2Params) { p["nodes"] = "other" },
		func(p db2Params) { p["virtual-ip"].(map[string]any)["nodes"] = "other" },
		func(p db2Params) { p["virtual-ip"].(map[string]any)["ip"] = "" },
		func(p db2Params) { p["filesystem"].(map[string]any)["device"] = "" },
		func(p db2Params) { p["virtual-ip"].(map[string]any)["cidr_netmask"] = "bad" },
		func(p db2Params) { p["instance"] = true },
		func(p db2Params) { p["id"] = map[string]any{"injected": "value"} },
	} {
		params := validDB2Params()
		change(params)
		require.Error(t, validateDB2Params(params, view))
	}
}
func TestDB2Command(t *testing.T) {
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "db2", op, validDB2Params())
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"db2"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var params db2Params
			require.NoError(t, json.Unmarshal(args[2], &params))
			require.Equal(t, validDB2Params(), params)
		}
	}
}
func TestDB2Template(t *testing.T) {
	d := db2TestDefinition(t)
	view := buildDB2View(d, nil)
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	data := map[string]any{"Title": d.Shortdesc, "WizardName": "db2", "Description": []string{d.Longdesc}, "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", data))
	for _, value := range []string{`name="db2.id"`, `name="db2.instance"`, `name="db2.virtual-ip.ip"`, `name="db2.filesystem.device"`, `value="db2-database"`, `value="db2inst1"`, `value="xfs"`, `id="step0"`, `id="step1"`, `id="step2"`, `id="db2-next"`, `id="db2-previous"`, `name="submit" value="Verify"`} {
		require.Contains(t, body.String(), value)
	}
	data["VerifyMode"] = true
	data["ParamsJSON"] = `{}`
	data["Actions"] = []healthAction{{Number: 1, Name: "cib", Text: "group g-db2-database meta target-role=Stopped <unsafe>"}}
	body.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", data))
	for _, value := range []string{`action="/cib/live/wizards/db2/submit"`, `name="submit" value="Apply"`, `class="crmsh hljs"`, `target-role=Stopped &lt;unsafe&gt;`} {
		require.Contains(t, body.String(), value)
	}
}
func TestDB2VerifyAndBack(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	operations := []string{}
	run := func(_ context.Context, op string, params db2Params) ([]byte, error) {
		operations = append(operations, op)
		if op == "show" {
			return []byte(db2Fixture), nil
		}
		require.Equal(t, "verify", op)
		require.Equal(t, validDB2Params(), params)
		return []byte(`{"name":"cib","text":"group g-db2-database meta target-role=Stopped"}`), nil
	}
	fields := url.Values{"db2.id": {"db2-database"}, "db2.instance": {"db2inst1"}, "db2.virtual-ip.id": {"db2-virtual-ip"}, "db2.virtual-ip.ip": {"192.0.2.10"}, "db2.virtual-ip.cidr_netmask": {"24"}, "db2.virtual-ip.nic": {"eth0"}, "db2.filesystem.id": {"db2-fs"}, "db2.filesystem.device": {"/dev/example"}, "db2.filesystem.directory": {"/db2/db2inst1"}, "db2.filesystem.fstype": {"xfs"}}
	req := httptest.NewRequest("POST", "https://cluster/cib/live/wizards/db2", strings.NewReader(fields.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	serveDB2Wizard(response, req, run)
	require.Equal(t, 200, response.Code)
	require.Equal(t, []string{"show", "verify"}, operations)
	require.Contains(t, response.Body.String(), "target-role=Stopped")
	cookies := response.Result().Cookies()
	require.Len(t, cookies, 1)
	require.True(t, cookies[0].Secure)
	req = httptest.NewRequest("GET", "https://cluster/cib/live/wizards/db2", nil)
	req.AddCookie(cookies[0])
	var restored db2Params
	require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(req, "db2"), &restored))
	require.Equal(t, validDB2Params(), restored)
	response = httptest.NewRecorder()
	serveDB2Wizard(response, req, run)
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `value="192.0.2.10"`)
}
func TestSubmitDB2(t *testing.T) {
	for _, failure := range []string{"", "verify", "run", "partial"} {
		t.Run(failure, func(t *testing.T) {
			operations := []string{}
			run := func(_ context.Context, op string, params db2Params) ([]byte, error) {
				operations = append(operations, op)
				if op == "show" {
					return []byte(db2Fixture), nil
				}
				require.Equal(t, validDB2Params(), params)
				if op == failure {
					return nil, fmt.Errorf("crmsh failed")
				}
				if op == "verify" {
					if failure == "partial" {
						return []byte("{\"name\":\"cib\"}\n{\"name\":\"cib\"}"), nil
					}
					return []byte(`{"name":"cib"}`), nil
				}
				return []byte(`{"rc":true,"shortdesc":"Configure DB2"}`), nil
			}
			data, _ := json.Marshal(validDB2Params())
			req := httptest.NewRequest("POST", "/cib/live/wizards/db2/submit", strings.NewReader(url.Values{"pa": {string(data)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			submitDB2Wizard(response, req, run)
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
func TestDB2Methods(t *testing.T) {
	for _, tt := range []struct {
		method, path string
		status       int
	}{{"GET", "/cib/live/wizards/db2/submit", 405}, {"DELETE", "/cib/live/wizards/db2", 405}, {"GET", "/cib/live/wizards/db2/unknown", 404}} {
		response := httptest.NewRecorder()
		serveDB2Wizard(response, httptest.NewRequest(tt.method, tt.path, nil), func(context.Context, string, db2Params) ([]byte, error) { t.Fatal("must not execute"); return nil, nil })
		require.Equal(t, tt.status, response.Code)
	}
}
