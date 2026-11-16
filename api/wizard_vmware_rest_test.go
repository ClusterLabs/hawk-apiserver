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

func vmwareRESTDefinition(t *testing.T) (resourceWizardDefinition, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/vmware_rest.json")
	require.NoError(t, err)
	var definition resourceWizardDefinition
	require.NoError(t, json.Unmarshal(data, &definition))
	return definition, data
}

func vmwareRESTTestParams() db2Params {
	return db2Params{"id": "vcenter-fencing", "node_name": "node1", "machine_name": "guest-node1", "server": "vcenter.example.com", "username": "administrator@vsphere.local", "password": `s"e\cret$; word`}
}

func TestVMwareRESTForm(t *testing.T) {
	definition, _ := vmwareRESTDefinition(t)
	view := buildNestedWizardView(definition, nil, "fence-vmware-rest")
	require.Len(t, view.Steps, 1)
	require.Len(t, view.Fields, 8)
	require.Len(t, view.Advanced, 1)
	params := vmwareRESTTestParams()
	require.NoError(t, validateDB2Params(params, view))
	for name := range params {
		missing := vmwareRESTTestParams()
		delete(missing, name)
		require.Error(t, validateDB2Params(missing, view))
	}
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "fence-vmware-rest", "Description": strings.Split(definition.Longdesc, "\n\n"), "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}))
	for _, value := range []string{`id="edit_wizard_fence-vmware-rest"`, `name="fence-vmware-rest.id"`, `name="fence-vmware-rest.node_name"`, `name="fence-vmware-rest.machine_name"`, `name="fence-vmware-rest.server"`, `name="fence-vmware-rest.username"`, `name="fence-vmware-rest.password"`, `type="password"`, `value="vcenter-fencing"`, `placeholder="Example: vcenter.example.com"`, "standalone ESXi host", "certificate", `id="fence-vmware-rest-password"`} {
		require.Contains(t, body.String(), value)
	}
	require.NotContains(t, body.String(), "credstore")
}

func TestVMwareRESTCommand(t *testing.T) {
	for _, operation := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "fence-vmware-rest", operation, vmwareRESTTestParams())
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"`+operation+`"`, string(args[0]))
		require.JSONEq(t, `"fence-vmware-rest"`, string(args[1]))
		if operation == "show" {
			require.Len(t, args, 2)
		} else {
			var params db2Params
			require.NoError(t, json.Unmarshal(args[2], &params))
			require.Equal(t, vmwareRESTTestParams(), params)
		}
	}
}

func TestVMwareRESTVerifyApplyAndBack(t *testing.T) {
	_, fixture := vmwareRESTDefinition(t)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, failure := range []string{"", "verify", "run", "partial"} {
		t.Run(failure, func(t *testing.T) {
			operations := []string{}
			run := func(_ context.Context, operation string, params db2Params) ([]byte, error) {
				operations = append(operations, operation)
				if operation == "show" {
					return fixture, nil
				}
				require.Equal(t, vmwareRESTTestParams(), params)
				if operation == failure {
					return nil, fmt.Errorf("REST verification failed")
				}
				if operation == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure cluster resources","text":"primitive vcenter-fencing-node1 stonith:fence_vmware_rest"}`), nil
				}
				if failure == "partial" {
					return nil, nil
				}
				return []byte(`{"rc":true}`), nil
			}
			form := url.Values{}
			for key, value := range vmwareRESTTestParams() {
				form.Set("fence-vmware-rest."+key, value.(string))
			}
			request := httptest.NewRequest("POST", "/cib/live/wizards/fence-vmware-rest", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			serveNestedWizard(response, request, run, vmwareRESTConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/fence-vmware-rest/submit"`)
			if failure == "verify" {
				require.Contains(t, response.Body.String(), "REST verification failed")
			} else {
				require.Contains(t, response.Body.String(), "stonith:fence_vmware_rest")
			}
			back := httptest.NewRequest("GET", "/cib/live/wizards/fence-vmware-rest", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			var restored db2Params
			require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(back, "fence-vmware-rest"), &restored))
			require.Equal(t, vmwareRESTTestParams(), restored)
			operations = nil
			pa, err := json.Marshal(vmwareRESTTestParams())
			require.NoError(t, err)
			request = httptest.NewRequest("POST", "/cib/live/wizards/fence-vmware-rest/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveNestedWizard(response, request, run, vmwareRESTConfig)
			if failure == "" {
				require.Equal(t, 200, response.Code)
			} else {
				require.Equal(t, 422, response.Code)
			}
			if failure == "verify" {
				require.Equal(t, []string{"show", "verify"}, operations)
			} else {
				require.Equal(t, []string{"show", "verify", "run"}, operations)
			}
		})
	}
}

func TestVMwareRESTEscapingAndValidation(t *testing.T) {
	params := vmwareRESTTestParams()
	cmd, err := vmwareRESTScriptCommand(context.Background(), "verify", params, "/tmp/rest scripts")
	require.NoError(t, err)
	require.Equal(t, []string{"/usr/sbin/crm", "--scriptdir", "/tmp/rest scripts", "-f", "/dev/stdin"}, cmd.Args)
	input, err := io.ReadAll(cmd.Stdin)
	require.NoError(t, err)
	payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
	require.NoError(t, err)
	var args []json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(payload), &args))
	var escaped db2Params
	require.NoError(t, json.Unmarshal(args[2], &escaped))
	require.Equal(t, `s\"e\\cret$; word`, escaped["password"])
	require.Equal(t, vmwareRESTTestParams(), params)
	for _, test := range []struct{ name, value string }{
		{"server", "https://vcenter.example.com"}, {"node_name", "node1\nproperty fencing-enabled=false"},
		{"machine_name", "one;two"}, {"password", "x\ncommand"}, {"ipport", "0"}, {"ipport", "65536"}, {"api_path", "/rest?x=y"}, {"credstore", "file"},
	} {
		params := vmwareRESTTestParams()
		params[test.name] = test.value
		_, err := vmwareRESTScriptCommand(context.Background(), "run", params, "/tmp/unused")
		require.Error(t, err)
	}
}
