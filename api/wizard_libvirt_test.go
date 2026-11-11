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

const libvirtFixture = `{"category": "stonith", "longdesc": "Uses libvirt as a STONITH device to fence a guest node. Create a\nseparate resource for each guest node in the cluster. Note that the\nrecommended fencing mechanism is SBD whenever a shared storage device\n(like a SAN) is available.", "name": "libvirt", "shortdesc": "STONITH for libvirt (kvm / Xen)", "steps": [{"longdesc": "", "parameters": [{"advanced": false, "example": "stonith-libvirt", "longdesc": "", "name": "id", "required": true, "shortdesc": "Resource ID (Name)", "type": "resource", "unique": false}, {"advanced": false, "longdesc": "", "name": "target", "required": true, "shortdesc": "Node to Manage With STONITH Device", "type": "resource", "unique": false}, {"advanced": false, "longdesc": "The optional domain_id defaults to the hostname.", "name": "hostlist", "required": true, "shortdesc": "List of controlled hosts: hostname[:domain_id]..", "type": "string", "unique": false}, {"advanced": false, "example": "qemu+ssh://my_kvm_server.example.com/system", "longdesc": "URI for connection to the hypervisor. driver[+transport]://[username@]\n[hostlist][:port]/[path][?extraparameters] e.g.\nqemu+ssh://my_kvm_server.mydomain.my/system   (uses ssh for root)\nxen://my_kvm_server.mydomain.my/              (uses TLS for client)\nvirsh must be installed (e.g. libvirt-client package) and access\ncontrol must be configured for your selected URI.", "name": "hypervisor_uri", "required": true, "shortdesc": "", "type": "string", "unique": false}, {"advanced": false, "example": "power_cycle", "longdesc": "A guest reset may be done by a sequence of off and on commands\n(power_cycle) or by the reboot command. Which method works depend on\nthe hypervisor and guest configuration management.", "name": "reset_method", "required": false, "shortdesc": "Guest Reset Method", "type": "string", "unique": false}, {"advanced": false, "longdesc": "", "name": "install", "required": false, "shortdesc": "Enable to Install Required Packages", "type": "boolean", "unique": false, "value": false}], "required": true, "shortdesc": ""}]}`

func TestLibvirtForm(t *testing.T) {
	definition, err := loadResourceWizardDefinition(context.Background(), func(_ context.Context, op string, _ wizardParams) ([]byte, error) {
		require.Equal(t, "show", op)
		return []byte(libvirtFixture), nil
	}, libvirtConfig.Title)
	require.NoError(t, err)
	require.Len(t, definition.Steps, 1)
	parameters := definition.Steps[0].Parameters
	require.Len(t, parameters, 6)
	tmpl, err := template.ParseFiles("../templates/wizard_resource.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "libvirt", "Basic": parameters, "Parameters": parameters, "Values": wizardParams{}}))
	for _, field := range parameters {
		require.Contains(t, body.String(), `name="libvirt.`+field.Name+`"`)
		if field.Type == "boolean" {
			require.Contains(t, body.String(), `value="false" selected`)
		} else {
			require.Contains(t, body.String(), `value="`+field.Value+`"`)
		}
	}
	require.Contains(t, body.String(), `id="edit_wizard_libvirt"`)
	require.NotContains(t, body.String(), `id="wizard-advanced"`)
	require.Contains(t, body.String(), `placeholder="Example: qemu&#43;ssh://my_kvm_server.example.com/system"`)
	require.Contains(t, body.String(), `placeholder="Example: power_cycle"`)
	params := wizardParams{"id": "stonith-libvirt", "target": "node1", "hostlist": "node1:guest1", "hypervisor_uri": "qemu+ssh://hypervisor.example.com/system", "reset_method": "power_cycle", "install": "false"}
	require.NoError(t, validateResourceWizardParams(params, definition, libvirtConfig.Title))
	for _, name := range []string{"id", "target", "hostlist", "hypervisor_uri"} {
		missing := wizardParams{}
		for key, value := range params {
			if key != name {
				missing[key] = value
			}
		}
		require.Error(t, validateResourceWizardParams(missing, definition, libvirtConfig.Title))
	}
	params["nodes"] = "node1"
	require.Error(t, validateResourceWizardParams(params, definition, libvirtConfig.Title))
}

func TestLibvirtScriptCommand(t *testing.T) {
	params := wizardParams{"id": "stonith-libvirt", "target": "node1", "hostlist": "node1:guest1", "hypervisor_uri": "qemu+ssh://hypervisor.example.com/system", "reset_method": "power_cycle", "install": "false"}
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := resourceWizardScriptCommand(context.Background(), "libvirt", op, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		stdin, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(stdin), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"libvirt"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var actual wizardParams
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}

func TestLibvirtVerifyAndApply(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	params := wizardParams{"id": "stonith-libvirt", "target": "node1", "hostlist": "node1:guest1", "hypervisor_uri": "qemu+ssh://hypervisor.example.com/system", "reset_method": "power_cycle", "install": "false"}
	for _, install := range []string{"false", "true"} {
		params["install"] = install
		for _, failure := range []string{"", "verify", "run", "partial"} {
			t.Run(install+"/"+failure, func(t *testing.T) {
				operations := []string{}
				run := func(_ context.Context, op string, actual wizardParams) ([]byte, error) {
					operations = append(operations, op)
					if op == "show" {
						return []byte(libvirtFixture), nil
					}
					require.Equal(t, params, actual)
					if op == failure {
						return nil, fmt.Errorf("Libvirt test failure")
					}
					if op == "verify" {
						preview := `{"name":"cib","shortdesc":"Configure cluster resources","text":"primitive stonith-libvirt-node1 stonith:external/libvirt"}`
						if install == "true" || failure == "partial" {
							preview = "{\"name\":\"install\",\"shortdesc\":\"Install libvirt packages\"}\n" + preview
						}
						return []byte(preview), nil
					}
					result := `{"rc":true,"shortdesc":"Configure cluster resources"}`
					if install == "true" && failure != "partial" {
						result = "{\"rc\":true,\"shortdesc\":\"Install libvirt packages\"}\n" + result
					}
					return []byte(result), nil
				}
				form := url.Values{}
				for key, value := range params {
					form.Set("libvirt."+key, value)
				}
				req := httptest.NewRequest("POST", "/cib/live/wizards/libvirt", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response := httptest.NewRecorder()
				serveResourceWizard(response, req, run, libvirtConfig)
				require.Equal(t, 200, response.Code)
				require.Equal(t, []string{"show", "verify"}, operations)
				require.Contains(t, response.Body.String(), `action="/cib/live/wizards/libvirt/submit"`)
				if failure == "verify" {
					require.Contains(t, response.Body.String(), `disabled`)
				} else {
					require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
				}
				back := httptest.NewRequest("GET", "/cib/live/wizards/libvirt", nil)
				for _, cookie := range response.Result().Cookies() {
					back.AddCookie(cookie)
				}
				require.Equal(t, params, recalledWizardValues(back, "libvirt"))
				operations = nil
				pa, _ := json.Marshal(params)
				req = httptest.NewRequest("POST", "/cib/live/wizards/libvirt/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response = httptest.NewRecorder()
				serveResourceWizard(response, req, run, libvirtConfig)
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
