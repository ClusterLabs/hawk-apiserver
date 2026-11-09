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

const oracleFixture = `{"category": "database", "longdesc": "Configure an Oracle Database cluster resource.", "name": "oracle", "shortdesc": "Oracle Database", "steps": [{"longdesc": "", "parameters": [{"advanced": false, "longdesc": "Unique ID for the database cluster resource.", "name": "id", "required": true, "shortdesc": "Resource ID", "type": "resource", "unique": false, "value": "oracle"}, {"advanced": false, "longdesc": "", "name": "sid", "required": true, "shortdesc": "Database SID", "type": "string", "unique": false, "value": "OracleDB"}, {"advanced": false, "longdesc": "", "name": "listener", "required": true, "shortdesc": "Listener.", "type": "string", "unique": false, "value": "LISTENER"}, {"advanced": false, "longdesc": "", "name": "home", "required": true, "shortdesc": "Database Home.", "type": "string", "unique": false, "value": "/srv/oracledb"}, {"advanced": false, "longdesc": "", "name": "user", "required": true, "shortdesc": "Database User.", "type": "string", "unique": false, "value": "oracle"}], "required": true, "shortdesc": ""}]}`

func TestOracleForm(t *testing.T) {
	definition, err := loadResourceWizardDefinition(context.Background(), func(_ context.Context, op string, _ wizardParams) ([]byte, error) {
		require.Equal(t, "show", op)
		return []byte(oracleFixture), nil
	}, oracleConfig.Title)
	require.NoError(t, err)
	require.Len(t, definition.Steps, 1)
	parameters := definition.Steps[0].Parameters
	require.Len(t, parameters, 5)
	tmpl, err := template.ParseFiles("../templates/wizard_resource.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "oracle", "Basic": parameters, "Parameters": parameters, "Values": wizardParams{}}))
	for _, field := range parameters {
		require.Contains(t, body.String(), `name="oracle.`+field.Name+`"`)
		require.Contains(t, body.String(), `value="`+field.Value+`"`)
	}
	require.Contains(t, body.String(), `id="edit_wizard_oracle"`)
	require.NotContains(t, body.String(), `id="wizard-advanced"`)
	params := wizardParams{"id": "oracle", "sid": "OracleDB", "listener": "LISTENER", "home": "/srv/oracledb", "user": "oracle"}
	require.NoError(t, validateResourceWizardParams(params, definition, oracleConfig.Title))
	for name := range params {
		missing := wizardParams{}
		for key, value := range params {
			if key != name {
				missing[key] = value
			}
		}
		require.Error(t, validateResourceWizardParams(missing, definition, oracleConfig.Title))
	}
	params["nodes"] = "node1"
	require.Error(t, validateResourceWizardParams(params, definition, oracleConfig.Title))
}

func TestOracleScriptCommand(t *testing.T) {
	params := wizardParams{"id": "oracle1", "sid": "DB", "home": "/srv/db with spaces", "listener": "LISTENER", "user": "oracle"}
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := resourceWizardScriptCommand(context.Background(), "oracle", op, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		stdin, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(stdin), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"oracle"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var actual wizardParams
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}

func TestOracleVerifyAndApply(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	params := wizardParams{"id": "oracle1", "sid": "DB", "home": "/srv/oracle", "listener": "LISTENER", "user": "oracle"}
	for _, failure := range []string{"", "verify", "run"} {
		t.Run(failure, func(t *testing.T) {
			operations := []string{}
			run := func(_ context.Context, op string, actual wizardParams) ([]byte, error) {
				operations = append(operations, op)
				if op == "show" {
					return []byte(oracleFixture), nil
				}
				require.Equal(t, params, actual)
				if op == failure {
					return nil, fmt.Errorf("Oracle test failure")
				}
				if op == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure cluster resources","text":"primitive oracle1 ocf:heartbeat:oracle"}`), nil
				}
				return []byte(`{"rc":true,"shortdesc":"Configure cluster resources"}`), nil
			}
			form := url.Values{}
			for key, value := range params {
				form.Set("oracle."+key, value)
			}
			req := httptest.NewRequest("POST", "/cib/live/wizards/oracle", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			serveResourceWizard(response, req, run, oracleConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/oracle/submit"`)
			if failure == "verify" {
				require.Contains(t, response.Body.String(), `disabled`)
			} else {
				require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
			}
			back := httptest.NewRequest("GET", "/cib/live/wizards/oracle", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			require.Equal(t, params, recalledWizardValues(back, "oracle"))
			operations = nil
			pa, _ := json.Marshal(params)
			req = httptest.NewRequest("POST", "/cib/live/wizards/oracle/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveResourceWizard(response, req, run, oracleConfig)
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
