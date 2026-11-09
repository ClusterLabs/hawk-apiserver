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

const drbdFixture = `{"category": "file system", "longdesc": "Distributed Replicated Block Device. Configure a DRBD cluster\nresource. Also creates a multistate resource managing the state of\nDRBD. Does not create or modify the referenced DRBD configuration.", "name": "drbd", "shortdesc": "DRBD Block Device", "steps": [{"longdesc": "", "parameters": [{"advanced": false, "longdesc": "", "name": "id", "required": true, "shortdesc": "DRBD Cluster Resource ID", "type": "resource", "unique": false, "value": "drbd-data"}, {"advanced": false, "longdesc": "", "name": "drbd_resource", "required": true, "shortdesc": "DRBD Resource Name", "type": "string", "unique": false, "value": "drbd0"}, {"advanced": false, "longdesc": "", "name": "drbdconf", "required": false, "shortdesc": "", "type": "string", "unique": false, "value": "/etc/drbd.conf"}, {"advanced": false, "longdesc": "", "name": "install", "required": false, "shortdesc": "Install packages for DRBD", "type": "boolean", "unique": false, "value": false}], "required": true, "shortdesc": ""}]}`

func TestDRBDForm(t *testing.T) {
	definition, err := loadResourceWizardDefinition(context.Background(), func(_ context.Context, op string, _ wizardParams) ([]byte, error) {
		require.Equal(t, "show", op)
		return []byte(drbdFixture), nil
	}, drbdConfig.Title)
	require.NoError(t, err)
	require.Len(t, definition.Steps, 1)
	parameters := definition.Steps[0].Parameters
	require.Len(t, parameters, 4)
	tmpl, err := template.ParseFiles("../templates/wizard_resource.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "drbd", "Basic": parameters, "Parameters": parameters, "Values": wizardParams{}}))
	for _, field := range parameters {
		require.Contains(t, body.String(), `name="drbd.`+field.Name+`"`)
		if field.Type == "boolean" {
			require.Contains(t, body.String(), `value="false" selected`)
		} else {
			require.Contains(t, body.String(), `value="`+field.Value+`"`)
		}
	}
	require.Contains(t, body.String(), `id="edit_wizard_drbd"`)
	require.NotContains(t, body.String(), `id="wizard-advanced"`)
	params := wizardParams{"id": "drbd-data", "drbd_resource": "drbd0", "drbdconf": "/etc/drbd.conf", "install": "false"}
	require.NoError(t, validateResourceWizardParams(params, definition, drbdConfig.Title))
	for _, name := range []string{"id", "drbd_resource"} {
		missing := wizardParams{}
		for key, value := range params {
			if key != name {
				missing[key] = value
			}
		}
		require.Error(t, validateResourceWizardParams(missing, definition, drbdConfig.Title))
	}
	params["nodes"] = "node1"
	require.Error(t, validateResourceWizardParams(params, definition, drbdConfig.Title))
}

func TestDRBDScriptCommand(t *testing.T) {
	params := wizardParams{"id": "drbd-data", "drbd_resource": "drbd0", "drbdconf": "/etc/drbd.conf", "install": "false"}
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := resourceWizardScriptCommand(context.Background(), "drbd", op, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		stdin, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(stdin), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"drbd"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var actual wizardParams
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}

func TestDRBDVerifyAndApply(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	params := wizardParams{"id": "drbd-data", "drbd_resource": "drbd0", "drbdconf": "/etc/drbd.conf", "install": "false"}
	for _, install := range []string{"false", "true"} {
		params["install"] = install
		for _, failure := range []string{"", "verify", "run", "partial"} {
			t.Run(install+"/"+failure, func(t *testing.T) {
				operations := []string{}
				run := func(_ context.Context, op string, actual wizardParams) ([]byte, error) {
					operations = append(operations, op)
					if op == "show" {
						return []byte(drbdFixture), nil
					}
					require.Equal(t, params, actual)
					if op == failure {
						return nil, fmt.Errorf("DRBD test failure")
					}
					if op == "verify" {
						preview := `{"name":"cib","shortdesc":"Configure cluster resources","text":"primitive drbd-data ocf:linbit:drbd"}`
						if install == "true" || failure == "partial" {
							preview = "{\"name\":\"install\",\"shortdesc\":\"Install packages for DRBD\"}\n" + preview
						}
						return []byte(preview), nil
					}
					result := `{"rc":true,"shortdesc":"Configure cluster resources"}`
					if install == "true" && failure != "partial" {
						result = "{\"rc\":true,\"shortdesc\":\"Install packages for DRBD\"}\n" + result
					}
					return []byte(result), nil
				}
				form := url.Values{}
				for key, value := range params {
					form.Set("drbd."+key, value)
				}
				req := httptest.NewRequest("POST", "/cib/live/wizards/drbd", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response := httptest.NewRecorder()
				serveResourceWizard(response, req, run, drbdConfig)
				require.Equal(t, 200, response.Code)
				require.Equal(t, []string{"show", "verify"}, operations)
				require.Contains(t, response.Body.String(), `action="/cib/live/wizards/drbd/submit"`)
				if failure == "verify" {
					require.Contains(t, response.Body.String(), `disabled`)
				} else {
					require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
				}
				back := httptest.NewRequest("GET", "/cib/live/wizards/drbd", nil)
				for _, cookie := range response.Result().Cookies() {
					back.AddCookie(cookie)
				}
				require.Equal(t, params, recalledWizardValues(back, "drbd"))
				operations = nil
				pa, _ := json.Marshal(params)
				req = httptest.NewRequest("POST", "/cib/live/wizards/drbd/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response = httptest.NewRecorder()
				serveResourceWizard(response, req, run, drbdConfig)
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
}
