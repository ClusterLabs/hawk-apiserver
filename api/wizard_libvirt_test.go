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

func libvirtDefinition(t *testing.T) (resourceWizardDefinition, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/libvirt.json")
	require.NoError(t, err)
	var definition resourceWizardDefinition
	require.NoError(t, json.Unmarshal(data, &definition))
	return definition, data
}

func libvirtTestParams() db2Params {
	return db2Params{"id": "stonith-libvirt", "target": "node1", "ip": "192.168.122.1", "username": "root", "plug": "guest-node1", "install": "false"}
}

func TestLibvirtForm(t *testing.T) {
	definition, _ := libvirtDefinition(t)
	view := buildNestedWizardView(definition, nil, "fence-virsh")
	require.Len(t, view.Steps, 1)
	require.Len(t, view.Advanced, 1)
	params := libvirtTestParams()
	require.NoError(t, validateDB2Params(params, view))
	require.NoError(t, validateLibvirtParams(params))
	tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "fence-virsh", "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}))
	for _, value := range []string{`id="edit_wizard_fence-virsh"`, `name="fence-virsh.ip"`, `name="fence-virsh.target"`, `name="fence-virsh.plug"`, `name="fence-virsh.username"`, `value="root"`, `placeholder="Example: 192.168.122.1"`, `name="fence-virsh.install"`, `value="false" selected`} {
		require.Contains(t, body.String(), value)
	}
	require.NotContains(t, body.String(), "hypervisor_uri")
	require.NotContains(t, body.String(), "hostlist")
}

func TestLibvirtScriptCommand(t *testing.T) {
	for _, operation := range []string{"show", "verify", "run"} {
		cmd, err := libvirtScriptCommand(context.Background(), operation, libvirtTestParams(), "/tmp/scripts with spaces")
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "--scriptdir", "/tmp/scripts with spaces", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"`+operation+`"`, string(args[0]))
		require.JSONEq(t, `"libvirt"`, string(args[1]))
		if operation == "show" {
			require.Len(t, args, 2)
		} else {
			var params db2Params
			require.NoError(t, json.Unmarshal(args[2], &params))
			require.Equal(t, libvirtTestParams(), params)
		}
	}
	_, err := libvirtScriptCommand(context.Background(), "invalid", libvirtTestParams(), "/tmp/scripts")
	require.Error(t, err)
}

func TestLibvirtRejectsInvalidParameters(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
	}{
		{"id", ""}, {"target", "node1\nproperty fencing-enabled=false"}, {"plug", "guest;reboot"},
		{"plug", "--help"}, {"ip", "host;command"}, {"ip", `host"`}, {"identity_file", "key\ncommand"},
		{"ipport", "0"}, {"ipport", "65536"}, {"ipport", "x"}, {"install", "maybe"}, {"hypervisor_uri", "qemu:///system"}, {"username", true},
	} {
		t.Run(test.name+fmt.Sprint(test.value), func(t *testing.T) {
			params := libvirtTestParams()
			params[test.name] = test.value
			_, err := libvirtScriptCommand(context.Background(), "run", params, "/tmp/unused")
			require.Error(t, err)
		})
	}
	for _, ip := range []string{"192.168.122.1", "hypervisor.example.com", "2001:db8::1"} {
		params := libvirtTestParams()
		params["ip"] = ip
		require.NoError(t, validateLibvirtParams(params))
	}
}

func TestLibvirtVerifyApplyAndBack(t *testing.T) {
	_, fixture := libvirtDefinition(t)
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
				require.Equal(t, libvirtTestParams(), params)
				if operation == failure {
					return nil, fmt.Errorf("execution failed")
				}
				if operation == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure cluster resources","text":"primitive stonith-libvirt-node1 stonith:fence_virsh"}`), nil
				}
				if failure == "partial" {
					return nil, nil
				}
				return []byte(`{"rc":true}`), nil
			}
			form := url.Values{}
			for key, value := range libvirtTestParams() {
				form.Set("fence-virsh."+key, value.(string))
			}
			request := httptest.NewRequest("POST", "/cib/live/wizards/fence-virsh", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			serveNestedWizard(response, request, run, fenceVirshConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/fence-virsh/submit"`)
			back := httptest.NewRequest("GET", "/cib/live/wizards/fence-virsh", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			var restored db2Params
			require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(back, "fence-virsh"), &restored))
			require.Equal(t, libvirtTestParams(), restored)
			operations = nil
			pa, err := json.Marshal(libvirtTestParams())
			require.NoError(t, err)
			request = httptest.NewRequest("POST", "/cib/live/wizards/fence-virsh/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveNestedWizard(response, request, run, fenceVirshConfig)
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
