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

const haproxyFixture = `{"category": "server", "longdesc": "HAProxy is a free, very fast and reliable solution offering high\navailability, load balancing, and proxying for TCP and HTTP-based\napplications. It is particularly suited for very high traffic web\nsites and powers quite a number of the world's most visited ones.\n\nNOTE: Installs a basic haproxy.cfg configuration file. This will\noverwrite any existing haproxy.cfg.", "name": "haproxy", "shortdesc": "HAProxy", "steps": [{"longdesc": "     haproxy.service", "parameters": [{"advanced": false, "longdesc": "", "name": "install", "required": false, "shortdesc": "Install and configure HAProxy packages", "type": "boolean", "unique": false, "value": false}, {"longdesc": "", "name": "id", "required": true, "shortdesc": "Identifier for the cluster resource", "type": "resource", "unique": true}], "required": true, "shortdesc": "systemd unit file for haproxy"}]}`

func TestHAProxyForm(t *testing.T) {
	definition, err := loadResourceWizardDefinition(context.Background(), func(_ context.Context, op string, _ wizardParams) ([]byte, error) {
		require.Equal(t, "show", op)
		return []byte(haproxyFixture), nil
	}, haProxyConfig.Title)
	require.NoError(t, err)
	require.Len(t, definition.Steps, 1)
	parameters := definition.Steps[0].Parameters
	require.Len(t, parameters, 2)
	tmpl, err := template.ParseFiles("../templates/wizard_resource.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "haproxy", "Basic": parameters, "Parameters": parameters, "Values": wizardParams{}}))
	for _, field := range parameters {
		require.Contains(t, body.String(), `name="haproxy.`+field.Name+`"`)
		if field.Type == "boolean" {
			require.Contains(t, body.String(), `value="false" selected`)
		} else {
			require.Contains(t, body.String(), `value="`+field.Value+`"`)
		}
	}
	require.Contains(t, body.String(), `id="edit_wizard_haproxy"`)
	require.NotContains(t, body.String(), `id="wizard-advanced"`)
	params := wizardParams{"id": "proxy", "install": "false"}
	require.NoError(t, validateResourceWizardParams(params, definition, haProxyConfig.Title))
	for _, name := range []string{"id"} {
		missing := wizardParams{}
		for key, value := range params {
			if key != name {
				missing[key] = value
			}
		}
		require.Error(t, validateResourceWizardParams(missing, definition, haProxyConfig.Title))
	}
	params["nodes"] = "node1"
	require.Error(t, validateResourceWizardParams(params, definition, haProxyConfig.Title))
}

func TestHAProxyScriptCommand(t *testing.T) {
	params := wizardParams{"id": "proxy", "install": "false"}
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := resourceWizardScriptCommand(context.Background(), "haproxy", op, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		stdin, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(stdin), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"haproxy"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var actual wizardParams
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}

func TestHAProxyVerifyAndApply(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	params := wizardParams{"id": "proxy", "install": "false"}
	for _, install := range []string{"false", "true"} {
		params["install"] = install
		for _, failure := range []string{"", "verify", "run", "partial"} {
			t.Run(install+"/"+failure, func(t *testing.T) {
				operations := []string{}
				run := func(_ context.Context, op string, actual wizardParams) ([]byte, error) {
					operations = append(operations, op)
					if op == "show" {
						return []byte(haproxyFixture), nil
					}
					require.Equal(t, params, actual)
					if op == failure {
						return nil, fmt.Errorf("HAProxy test failure")
					}
					if op == "verify" {
						preview := `{"name":"cib","shortdesc":"Configure cluster resources","text":"primitive proxy systemd:haproxy"}`
						if install == "true" || failure == "partial" {
							preview = "{\"name\":\"install\",\"shortdesc\":\"Install HAProxy\"}\n{\"name\":\"service\"}\n{\"name\":\"copy\"}\n" + preview
						}
						return []byte(preview), nil
					}
					result := `{"rc":true,"shortdesc":"Configure cluster resources"}`
					if install == "true" && failure != "partial" {
						result = "{\"rc\":true,\"shortdesc\":\"Install HAProxy\"}\n{\"rc\":true}\n{\"rc\":true}\n" + result
					}
					return []byte(result), nil
				}
				form := url.Values{}
				for key, value := range params {
					form.Set("haproxy."+key, value)
				}
				req := httptest.NewRequest("POST", "/cib/live/wizards/haproxy", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response := httptest.NewRecorder()
				serveResourceWizard(response, req, run, haProxyConfig)
				require.Equal(t, 200, response.Code)
				require.Equal(t, []string{"show", "verify"}, operations)
				require.Contains(t, response.Body.String(), `action="/cib/live/wizards/haproxy/submit"`)
				if failure == "verify" {
					require.Contains(t, response.Body.String(), `disabled`)
				} else {
					require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
				}
				back := httptest.NewRequest("GET", "/cib/live/wizards/haproxy", nil)
				for _, cookie := range response.Result().Cookies() {
					back.AddCookie(cookie)
				}
				require.Equal(t, params, recalledWizardValues(back, "haproxy"))
				operations = nil
				pa, _ := json.Marshal(params)
				req = httptest.NewRequest("POST", "/cib/live/wizards/haproxy/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response = httptest.NewRecorder()
				serveResourceWizard(response, req, run, haProxyConfig)
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
