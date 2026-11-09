package api

import (
	"bytes"
	"context"
	"encoding/base64"
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
	"time"

	"github.com/stretchr/testify/require"
)

const databaseFixture = `{"name":"database","shortdesc":"MySQL/MariaDB Database","longdesc":"Configure a MySQL or MariaDB SQL Database.","steps":[{"longdesc":"Resource script for MySQL.","parameters":[{"name":"install","type":"boolean","value":false,"shortdesc":"Enable to install required packages"},{"name":"id","type":"resource","required":true,"shortdesc":"Identifier for the cluster resource"},{"name":"test_table","type":"string","value":"","example":"mysql.user","shortdesc":"MySQL test table"},{"name":"datadir","type":"string","advanced":true,"example":"/var/lib/mysql"},{"name":"test_passwd","type":"string","advanced":true,"example":""},{"name":"enable_creation","type":"boolean","advanced":true,"example":"0"},{"name":"max_slave_lag","type":"integer","advanced":true,"example":"3600"}]}]}`

func databaseTestDefinition(t *testing.T) resourceWizardDefinition {
	t.Helper()
	definition, err := loadResourceWizardDefinition(context.Background(), func(_ context.Context, op string, _ wizardParams) ([]byte, error) {
		require.Equal(t, "show", op)
		return []byte(databaseFixture), nil
	}, "Database")
	require.NoError(t, err)
	return definition
}
func TestDatabaseDefinitionAndTemplate(t *testing.T) {
	definition := databaseTestDefinition(t)
	parameters := definition.Steps[0].Parameters
	require.Equal(t, "false", parameters[0].Value)
	require.Empty(t, parameters[2].Value)
	require.Equal(t, "false", parameters[5].Example)
	tmpl, err := template.ParseFiles("../templates/wizard_resource.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "database", "Basic": parameters[:3], "Advanced": parameters[3:], "Parameters": parameters, "Values": wizardParams{}}))
	for _, value := range []string{`id="edit_wizard_database"`, `name="database.id"`, `name="database.install"`, `name="database.test_table"`, `value="false" selected`, `placeholder="Example: mysql.user"`, `name="submit" value="Verify"`, `id="wizard-advanced-fields"`} {
		require.Contains(t, body.String(), value)
	}
	require.NotContains(t, body.String(), `value="mysql.user"`)
	body.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "database", "Basic": parameters[:3], "Values": wizardParams{"install": "true"}}))
	require.Contains(t, body.String(), `value="true" selected`)
	body.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "database", "VerifyMode": true, "ParamsJSON": `{"id":"db","install":"false"}`, "Actions": []healthAction{{Number: 1, Name: "cib", Text: "primitive db <unsafe>"}}}))
	for _, value := range []string{`action="/cib/live/wizards/database/submit"`, `name="submit" value="Apply"`, `href="/cib/live/wizards/database"`, `class="crmsh hljs"`, `&lt;unsafe&gt;`} {
		require.Contains(t, body.String(), value)
	}
}
func TestDatabaseScriptCommand(t *testing.T) {
	params := wizardParams{"id": "db", "install": "false", "test_passwd": "<secret> \"quoted\" $HOME\nnext"}
	for _, operation := range []string{"show", "verify", "run"} {
		cmd, err := resourceWizardScriptCommand(context.Background(), "database", operation, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"database"`, string(args[1]))
		if operation == "show" {
			require.Len(t, args, 2)
		} else {
			var actual wizardParams
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}
func TestDatabaseRememberedValues(t *testing.T) {
	params := wizardParams{"id": "db", "test_passwd": "secret-password", "replication_passwd": "replication-secret"}
	req := httptest.NewRequest("GET", "https://cluster/cib/live/wizards/database", nil)
	response := httptest.NewRecorder()
	rememberWizardValues(response, req, "database", params)
	cookies := response.Result().Cookies()
	require.Len(t, cookies, 1)
	require.True(t, cookies[0].Secure)
	require.True(t, cookies[0].HttpOnly)
	require.Equal(t, 3600, cookies[0].MaxAge)
	encrypted, err := base64.RawURLEncoding.DecodeString(cookies[0].Value)
	require.NoError(t, err)
	require.NotContains(t, string(encrypted), "secret-password")
	req.AddCookie(cookies[0])
	require.Equal(t, params, recalledWizardValues(req, "database"))
	encrypted[len(encrypted)-1] ^= 1
	bad := httptest.NewRequest("GET", "/cib/live/wizards/database", nil)
	tampered := *cookies[0]
	tampered.Value = base64.RawURLEncoding.EncodeToString(encrypted)
	bad.AddCookie(&tampered)
	require.Empty(t, recalledWizardValues(bad, "database"))
	// An authenticated cookie must still expire after the one-hour cache window.
	data, _ := json.Marshal(struct {
		Params  wizardParams
		Expires int64
	}{params, time.Now().Add(-time.Hour).Unix()})
	nonce := make([]byte, databaseStateCipher.NonceSize())
	expired := databaseStateCipher.Seal(nonce, nonce, data, []byte("database"))
	bad = httptest.NewRequest("GET", "/cib/live/wizards/database", nil)
	tampered.Value = base64.RawURLEncoding.EncodeToString(expired)
	bad.AddCookie(&tampered)
	require.Empty(t, recalledWizardValues(bad, "database"))
}
func TestDatabaseVerifyAndBack(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	operations := []string{}
	expected := wizardParams{"install": "false", "id": "db1", "test_passwd": "secret"}
	run := func(_ context.Context, operation string, params wizardParams) ([]byte, error) {
		operations = append(operations, operation)
		if operation == "show" {
			return []byte(databaseFixture), nil
		}
		require.Equal(t, "verify", operation)
		require.Equal(t, expected, params)
		return []byte(`{"name":"cib","shortdesc":"Configure cluster resources","text":"primitive db1 ocf:heartbeat:mysql"}`), nil
	}
	req := httptest.NewRequest("POST", "https://cluster/cib/live/wizards/database", strings.NewReader(url.Values{"database.install": {"false"}, "database.id": {"db1"}, "database.test_table": {""}, "database.test_passwd": {"secret"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	serveResourceWizard(response, req, run, databaseConfig)
	require.Equal(t, 200, response.Code)
	require.Equal(t, []string{"show", "verify"}, operations)
	require.Contains(t, response.Body.String(), `value="Apply"`)
	cookies := response.Result().Cookies()
	require.Len(t, cookies, 1)
	req = httptest.NewRequest("GET", "https://cluster/cib/live/wizards/database", nil)
	req.AddCookie(cookies[0])
	require.Equal(t, expected, recalledDatabaseValues(req))
	response = httptest.NewRecorder()
	serveResourceWizard(response, req, run, databaseConfig)
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `value="db1"`)
}
func TestSubmitDatabase(t *testing.T) {
	for _, install := range []string{"false", "true"} {
		for _, failure := range []string{"", "verify", "run", "partial"} {
			t.Run(install+"/"+failure, func(t *testing.T) {
				params := wizardParams{"id": "db", "install": install, "datadir": "/srv/mysql"}
				data, _ := json.Marshal(params)
				operations := []string{}
				run := func(_ context.Context, op string, actual wizardParams) ([]byte, error) {
					operations = append(operations, op)
					if op == "show" {
						return []byte(databaseFixture), nil
					}
					require.Equal(t, params, actual)
					if op == failure {
						return nil, fmt.Errorf("crmsh failure")
					}
					if op == "verify" {
						if failure == "partial" {
							return []byte("{\"name\":\"install\"}\n{\"name\":\"cib\"}\n"), nil
						}
						return []byte(`{"name":"cib"}`), nil
					}
					return []byte(`{"rc":true,"shortdesc":"Configured"}`), nil
				}
				req := httptest.NewRequest("POST", "/cib/live/wizards/database/submit", strings.NewReader(url.Values{"pa": {string(data)}}.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response := httptest.NewRecorder()
				submitResourceWizard(response, req, run, databaseConfig)
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
	definition := databaseTestDefinition(t)
	require.Error(t, validateResourceWizardParams(wizardParams{"install": "false"}, definition, "Database"))
	require.Error(t, validateResourceWizardParams(wizardParams{"id": "db", "dry_run": "yes"}, definition, "Database"))
}
