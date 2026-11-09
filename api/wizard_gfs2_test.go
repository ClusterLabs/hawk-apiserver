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

const gfs2Fixture = `{"category": "file system", "longdesc": "This template generates a cloned instance of the GFS2 file system. The\nfile system should be on the device, unless cLVM is used.", "name": "gfs2", "shortdesc": "GFS2 File System (Cloned)", "steps": [{"longdesc": "", "parameters": [{"advanced": false, "example": "bigfs", "longdesc": "NB: The clone is going to be named c-<id> (e.g. c-bigfs)", "name": "id", "required": true, "shortdesc": "File System Resource ID", "type": "resource", "unique": false}, {"advanced": false, "example": "/mnt/bigfs", "longdesc": "", "name": "directory", "required": true, "shortdesc": "Mount Point", "type": "string", "unique": false}, {"advanced": false, "longdesc": "", "name": "device", "required": true, "shortdesc": "Device", "type": "string", "unique": false}, {"advanced": false, "longdesc": "", "name": "options", "required": false, "shortdesc": "Mount Options", "type": "string", "unique": false}, {"advanced": false, "longdesc": "If set, create the DLM resource and cloned resource group.", "name": "dlm", "required": false, "shortdesc": "Create DLM Resource and Cloned Group", "type": "boolean", "unique": false, "value": true}, {"advanced": false, "longdesc": "ID of cloned group", "name": "group", "required": false, "shortdesc": "Cloned Group Resource ID", "type": "resource", "unique": false, "value": "g-dlm"}], "required": true, "shortdesc": ""}]}`

func TestGFS2Form(t *testing.T) {
	definition, err := loadResourceWizardDefinition(context.Background(), func(_ context.Context, op string, _ wizardParams) ([]byte, error) {
		require.Equal(t, "show", op)
		return []byte(gfs2Fixture), nil
	}, gfs2Config.Title)
	require.NoError(t, err)
	require.Len(t, definition.Steps, 1)
	parameters := definition.Steps[0].Parameters
	require.Len(t, parameters, 6)
	tmpl, err := template.ParseFiles("../templates/wizard_resource.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "gfs2", "Basic": parameters, "Parameters": parameters, "Values": wizardParams{}}))
	for _, field := range parameters {
		require.Contains(t, body.String(), `name="gfs2.`+field.Name+`"`)
		if field.Type == "boolean" {
			require.Contains(t, body.String(), `value="true" selected`)
		} else {
			require.Contains(t, body.String(), `value="`+field.Value+`"`)
			if field.Example != "" {
				require.Contains(t, body.String(), `placeholder="Example: `+field.Example+`"`)
			}
		}
	}
	require.Contains(t, body.String(), `id="edit_wizard_gfs2"`)
	require.NotContains(t, body.String(), `id="wizard-advanced"`)
	params := wizardParams{"id": "bigfs", "directory": "/mnt/bigfs", "device": "/dev/shared", "options": "noatime", "dlm": "true", "group": "g-dlm"}
	require.NoError(t, validateResourceWizardParams(params, definition, gfs2Config.Title))
	for _, name := range []string{"id", "directory", "device"} {
		missing := wizardParams{}
		for key, value := range params {
			if key != name {
				missing[key] = value
			}
		}
		require.Error(t, validateResourceWizardParams(missing, definition, gfs2Config.Title))
	}
	params["nodes"] = "node1"
	require.Error(t, validateResourceWizardParams(params, definition, gfs2Config.Title))
}

func TestGFS2ScriptCommand(t *testing.T) {
	params := wizardParams{"id": "bigfs", "directory": "/mnt/bigfs", "device": "/dev/shared", "options": "noatime", "dlm": "true", "group": "g-dlm"}
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := resourceWizardScriptCommand(context.Background(), "gfs2", op, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		stdin, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(stdin), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"gfs2"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var actual wizardParams
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}

func TestGFS2VerifyAndApply(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	params := wizardParams{"id": "bigfs", "directory": "/mnt/bigfs", "device": "/dev/shared", "options": "noatime", "dlm": "true", "group": "g-dlm"}
	for _, dlm := range []string{"false", "true"} {
		params["dlm"] = dlm
		for _, failure := range []string{"", "verify", "run", "partial"} {
			t.Run(dlm+"/"+failure, func(t *testing.T) {
				operations := []string{}
				run := func(_ context.Context, op string, actual wizardParams) ([]byte, error) {
					operations = append(operations, op)
					if op == "show" {
						return []byte(gfs2Fixture), nil
					}
					require.Equal(t, params, actual)
					if op == failure {
						return nil, fmt.Errorf("GFS2 test failure")
					}
					if op == "verify" {
						preview := `{"name":"cib","shortdesc":"Configure cluster resources","text":"primitive bigfs ocf:heartbeat:Filesystem"}`
						if dlm == "true" {
							preview = "{\"name\":\"cib\",\"shortdesc\":\"Create DLM resource\"}\n" + preview
						}
						preview += "\n" + `{"name":"crm","shortdesc":"Add the GFS2 File System to the Cloned Group","text":"configure modgroup g-dlm add bigfs"}`
						return []byte(preview), nil
					}
					result := `{"rc":true,"shortdesc":"Configure cluster resources"}`
					if dlm == "true" {
						result = "{\"rc\":true,\"shortdesc\":\"Create DLM resource\"}\n" + result
					}
					if failure != "partial" {
						result += "\n" + `{"rc":true,"shortdesc":"Add the GFS2 File System to the Cloned Group"}`
					}
					return []byte(result), nil
				}
				form := url.Values{}
				for key, value := range params {
					form.Set("gfs2."+key, value)
				}
				req := httptest.NewRequest("POST", "/cib/live/wizards/gfs2", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response := httptest.NewRecorder()
				serveResourceWizard(response, req, run, gfs2Config)
				require.Equal(t, 200, response.Code)
				require.Equal(t, []string{"show", "verify"}, operations)
				require.Contains(t, response.Body.String(), `action="/cib/live/wizards/gfs2/submit"`)
				if failure == "verify" {
					require.Contains(t, response.Body.String(), `disabled`)
				} else {
					require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
					require.Contains(t, response.Body.String(), `configure modgroup g-dlm add bigfs`)
				}
				back := httptest.NewRequest("GET", "/cib/live/wizards/gfs2", nil)
				for _, cookie := range response.Result().Cookies() {
					back.AddCookie(cookie)
				}
				require.Equal(t, params, recalledWizardValues(back, "gfs2"))
				operations = nil
				pa, _ := json.Marshal(params)
				req = httptest.NewRequest("POST", "/cib/live/wizards/gfs2/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response = httptest.NewRecorder()
				serveResourceWizard(response, req, run, gfs2Config)
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
