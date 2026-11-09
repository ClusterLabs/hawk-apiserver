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

const db2HADRFixture = `{"name":"db2-hadr","shortdesc":"IBM DB2 Database with HADR","longdesc":"Configure DB2 as active/passive HADR, along with a Virtual IP.","steps":[{"name":"db2","required":true,"shortdesc":"DB2 settings","parameters":[{"name":"id","type":"string","value":"db2-database"},{"name":"instance","type":"string","value":"db2inst1"},{"name":"dblist","type":"string","value":"db1"},{"name":"skip_basic_sql_health_check","type":"boolean","advanced":true,"example":"false"}]},{"name":"virtual-ip","required":true,"shortdesc":"Virtual IP","steps":[{"shortdesc":"IPaddr2 agent","parameters":[{"name":"id","type":"resource","value":"db2-virtual-ip"},{"name":"ip","type":"ip_address","required":true},{"name":"cidr_netmask","type":"integer"},{"name":"broadcast","type":"string"},{"name":"nic","type":"string","advanced":true}]}]}]}`

func hadrTestDefinition(t *testing.T) resourceWizardDefinition {
	t.Helper()
	var d resourceWizardDefinition
	require.NoError(t, json.Unmarshal([]byte(db2HADRFixture), &d))
	return d
}
func validHADRParams() db2Params {
	return db2Params{"db2": map[string]any{"id": "db2-database", "instance": "db2inst1", "dblist": "db1", "skip_basic_sql_health_check": "false"}, "virtual-ip": map[string]any{"id": "db2-virtual-ip", "ip": "192.0.2.10", "cidr_netmask": "24", "broadcast": "192.0.2.255", "nic": "eth0"}}
}

func TestDB2HADRViewAndForm(t *testing.T) {
	definition := hadrTestDefinition(t)
	view := buildNestedWizardView(definition, nil, "db2-hadr")
	require.Len(t, view.Steps, 2)
	require.Len(t, view.Advanced, 2)
	require.Equal(t, "db2-hadr.db2", view.Advanced[0].Prefix)
	require.Equal(t, "db2-hadr.virtual-ip", view.Advanced[1].Prefix)
	fields := map[string]db2Field{}
	for _, f := range view.Fields {
		fields[f.FieldID] = f
	}
	for _, tt := range []struct{ name, value string }{{"db2-hadr.db2.id", "db2-database"}, {"db2-hadr.db2.instance", "db2inst1"}, {"db2-hadr.db2.dblist", "db1"}, {"db2-hadr.virtual-ip.id", "db2-virtual-ip"}} {
		require.Equal(t, tt.value, fields[tt.name].Current)
	}
	require.Equal(t, []string{"db2", "id"}, fields["db2-hadr.db2.id"].Path)
	req := httptest.NewRequest("POST", "/", nil)
	req.Form = url.Values{}
	for name, f := range fields {
		if f.Current != "" {
			req.Form.Set(name, f.Current)
		}
	}
	req.Form.Set("db2-hadr.db2.skip_basic_sql_health_check", "false")
	req.Form.Set("db2-hadr.virtual-ip.ip", "192.0.2.10")
	req.Form.Set("db2-hadr.virtual-ip.cidr_netmask", "24")
	req.Form.Set("db2-hadr.virtual-ip.broadcast", "192.0.2.255")
	req.Form.Set("db2-hadr.virtual-ip.nic", "eth0")
	params := db2FormParams(req, definition, view)
	require.Equal(t, validHADRParams(), params)
	require.NoError(t, validateDB2Params(params, view))
	restored := buildNestedWizardView(definition, params, "db2-hadr")
	require.Equal(t, "false", restored.Advanced[0].Values["skip_basic_sql_health_check"])
	require.Equal(t, "eth0", restored.Advanced[1].Values["nic"])
	for _, change := range []func(db2Params){func(p db2Params) { p["id"] = "root-id" }, func(p db2Params) { p["nodes"] = "other" }, func(p db2Params) { p["db2"].(map[string]any)["nodes"] = "other" }, func(p db2Params) { p["virtual-ip"].(map[string]any)["ip"] = "" }} {
		params := validHADRParams()
		change(params)
		require.Error(t, validateDB2Params(params, view))
	}
}

func TestDB2HADRCommand(t *testing.T) {
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "db2-hadr", op, validHADRParams())
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"db2-hadr"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var params db2Params
			require.NoError(t, json.Unmarshal(args[2], &params))
			require.Equal(t, validHADRParams(), params)
		}
	}
}

func TestDB2HADRTemplate(t *testing.T) {
	definition := hadrTestDefinition(t)
	view := buildNestedWizardView(definition, nil, "db2-hadr")
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	data := map[string]any{"Title": definition.Shortdesc, "WizardName": "db2-hadr", "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", data))
	for _, value := range []string{`id="edit_wizard_db2-hadr"`, `name="db2-hadr.db2.id"`, `name="db2-hadr.db2.instance"`, `name="db2-hadr.db2.dblist"`, `name="db2-hadr.virtual-ip.ip"`, `value="db1"`, `id="step1"`, `name="submit" value="Verify"`} {
		require.Contains(t, body.String(), value)
	}
	require.NotContains(t, body.String(), `id="step2"`)
	require.NotContains(t, body.String(), `db2-hadr.filesystem`)
	data["VerifyMode"] = true
	data["Errors"] = []string{"Referenced resource does not exist"}
	data["ParamsJSON"] = `{}`
	data["Actions"] = []healthAction{{Number: 1, Name: "cib", Text: "clone promotable-db2-database db2-database meta target-role=Stopped <unsafe>"}}
	body.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", data))
	for _, value := range []string{`action="/cib/live/wizards/db2-hadr/submit"`, `href="/cib/live/wizards/db2-hadr"`, `name="submit" value="Apply"`, `disabled`, `Referenced resource does not exist`, `target-role=Stopped &lt;unsafe&gt;`} {
		require.Contains(t, body.String(), value)
	}
}

func TestDB2HADRVerifyAndBack(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	operations := []string{}
	run := func(_ context.Context, op string, params db2Params) ([]byte, error) {
		operations = append(operations, op)
		if op == "show" {
			return []byte(db2HADRFixture), nil
		}
		require.Equal(t, "verify", op)
		require.Equal(t, validHADRParams(), params)
		return []byte(`{"name":"cib","text":"clone promotable-db2-database db2-database meta target-role=Stopped"}`), nil
	}
	fields := url.Values{"db2-hadr.db2.id": {"db2-database"}, "db2-hadr.db2.instance": {"db2inst1"}, "db2-hadr.db2.dblist": {"db1"}, "db2-hadr.db2.skip_basic_sql_health_check": {"false"}, "db2-hadr.virtual-ip.id": {"db2-virtual-ip"}, "db2-hadr.virtual-ip.ip": {"192.0.2.10"}, "db2-hadr.virtual-ip.cidr_netmask": {"24"}, "db2-hadr.virtual-ip.broadcast": {"192.0.2.255"}, "db2-hadr.virtual-ip.nic": {"eth0"}}
	req := httptest.NewRequest("POST", "https://cluster/cib/live/wizards/db2-hadr", strings.NewReader(fields.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	serveNestedWizard(response, req, run, db2HADRConfig)
	require.Equal(t, 200, response.Code)
	require.Equal(t, []string{"show", "verify"}, operations)
	cookies := response.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, "hawk_wizard_db2-hadr", cookies[0].Name)
	require.True(t, cookies[0].Secure)
	req = httptest.NewRequest("GET", "https://cluster/cib/live/wizards/db2-hadr", nil)
	req.AddCookie(cookies[0])
	var restored db2Params
	require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(req, "db2-hadr"), &restored))
	require.Equal(t, validHADRParams(), restored)
	require.Empty(t, recalledEncryptedWizardValues(req, "db2"))
	response = httptest.NewRecorder()
	serveNestedWizard(response, req, run, db2HADRConfig)
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `value="192.0.2.10"`)
}

func TestSubmitDB2HADR(t *testing.T) {
	for _, failure := range []string{"", "verify", "run", "partial"} {
		t.Run(failure, func(t *testing.T) {
			operations := []string{}
			run := func(_ context.Context, op string, params db2Params) ([]byte, error) {
				operations = append(operations, op)
				if op == "show" {
					return []byte(db2HADRFixture), nil
				}
				require.Equal(t, validHADRParams(), params)
				if op == failure {
					return nil, fmt.Errorf("crmsh failed")
				}
				if op == "verify" {
					if failure == "partial" {
						return []byte("{\"name\":\"cib\"}\n{\"name\":\"cib\"}"), nil
					}
					return []byte(`{"name":"cib"}`), nil
				}
				return []byte(`{"rc":true,"shortdesc":"Configure HADR"}`), nil
			}
			data, _ := json.Marshal(validHADRParams())
			req := httptest.NewRequest("POST", "/cib/live/wizards/db2-hadr/submit", strings.NewReader(url.Values{"pa": {string(data)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			submitNestedWizard(response, req, run, db2HADRConfig)
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

func TestDB2HADRMethods(t *testing.T) {
	for _, tt := range []struct {
		method, path string
		status       int
	}{{"GET", "/cib/live/wizards/db2-hadr/submit", 405}, {"DELETE", "/cib/live/wizards/db2-hadr", 405}, {"GET", "/cib/live/wizards/db2-hadr/unknown", 404}} {
		response := httptest.NewRecorder()
		serveNestedWizard(response, httptest.NewRequest(tt.method, tt.path, nil), func(context.Context, string, db2Params) ([]byte, error) { t.Fatal("must not execute"); return nil, nil }, db2HADRConfig)
		require.Equal(t, tt.status, response.Code)
	}
}
