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

const virtualIPFixture = `{"shortdesc":"Virtual IP","longdesc":"Manages virtual IPv4 and IPv6 addresses.","steps":[{"parameters":[{"name":"id","type":"resource","required":true,"shortdesc":"Identifier for the cluster resource"},{"name":"ip","type":"ip_address","required":true,"shortdesc":"IPv4 or IPv6 address"},{"name":"cidr_netmask","type":"integer","shortdesc":"CIDR netmask"},{"name":"broadcast","type":"string","shortdesc":"Broadcast address"},{"name":"nic","type":"string","advanced":true,"longdesc":"Network interface"},{"name":"arp_count","type":"integer","advanced":true,"example":"5"},{"name":"lvs_support","type":"boolean","advanced":true,"example":"false"}]}]}`

func virtualIPTestDefinition(t *testing.T) resourceWizardDefinition {
	t.Helper()
	definition, err := loadVirtualIPDefinition(context.Background(), func(_ context.Context, operation string, _ wizardParams) ([]byte, error) {
		require.Equal(t, "show", operation)
		return []byte(virtualIPFixture), nil
	})
	require.NoError(t, err)
	return definition
}

func TestVirtualIPScriptCommand(t *testing.T) {
	params := wizardParams{"id": "vip", "ip": "192.0.2.10", "cidr_netmask": "24", "broadcast": "192.0.2.255", "nic": "eth0"}
	for _, operation := range []string{"show", "verify", "run"} {
		cmd, err := resourceWizardScriptCommand(context.Background(), "virtual-ip", operation, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		data, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(data), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"virtual-ip"`, string(args[1]))
		if operation == "show" {
			require.Len(t, args, 2)
		} else {
			var actual wizardParams
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
	_, err := resourceWizardScriptCommand(context.Background(), "arbitrary-script", "run", params)
	require.Error(t, err)
}

func TestVirtualIPParameters(t *testing.T) {
	definition := virtualIPTestDefinition(t)
	require.Equal(t, "cidr netmask", definition.Steps[0].Parameters[2].Label)
	for _, ip := range []string{"192.0.2.10", "2001:db8::10"} {
		require.NoError(t, validateVirtualIPParams(wizardParams{"id": "vip", "ip": ip, "nic": "eth0", "arp_count": "5", "lvs_support": "false"}, definition))
	}
	for _, params := range []wizardParams{
		{"id": "vip"},
		{"ip": "192.0.2.10"},
		{"id": "bad id", "ip": "192.0.2.10"},
		{"id": "vip", "ip": "192.0.2.10", "cidr_netmask": "not a number"},
		{"id": "vip", "ip": "192.0.2.10", "nodes": "remote"},
		{"id": "vip", "ip": "192.0.2.10", "dry_run": "yes"},
	} {
		require.Error(t, validateVirtualIPParams(params, definition))
	}
}

func TestVirtualIPTemplate(t *testing.T) {
	definition := virtualIPTestDefinition(t)
	basic := definition.Steps[0].Parameters[:4]
	advanced := definition.Steps[0].Parameters[4:]
	tmpl, err := template.ParseFiles("../templates/wizard_virtual_ip.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": "Virtual IP", "Parameters": definition.Steps[0].Parameters, "Basic": basic, "Advanced": advanced, "Values": wizardParams{"ip": "192.0.2.10"}}))
	for _, name := range []string{"id", "ip", "cidr_netmask", "broadcast"} {
		require.Contains(t, body.String(), `id="virtual-ip.`+name+`"`)
		require.Contains(t, body.String(), `name="virtual-ip.`+name+`"`)
	}
	for _, value := range []string{`name="submit" value="Verify"`, `pattern="[0-9]+"`, `virtual-ip-advanced-fields`, `Default:`, `/static/css/highlight.css`} {
		require.Contains(t, body.String(), value)
	}
	require.NotContains(t, body.String(), `name="virtual-ip.nic"`, "advanced fields are added on demand")
	body.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": "Virtual IP", "VerifyMode": true, "ParamsJSON": `{"id":"vip","ip":"192.0.2.10"}`, "Actions": []healthAction{{Number: 1, Name: "cib", Shortdesc: "Configure cluster resources", Text: "primitive vip <unsafe>"}}}))
	for _, value := range []string{`name="submit" value="Apply"`, `action="/cib/live/wizards/virtual-ip/submit"`, `class="crmsh hljs"`, `&lt;unsafe&gt;`} {
		require.Contains(t, body.String(), value)
	}
}

func TestVirtualIPVerifyAndBack(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	expected := wizardParams{"id": "vip", "ip": "192.0.2.10", "cidr_netmask": "24", "broadcast": "192.0.2.255", "nic": "eth0", "lvs_support": "false"}
	operations := []string{}
	run := func(_ context.Context, operation string, params wizardParams) ([]byte, error) {
		operations = append(operations, operation)
		if operation == "show" {
			return []byte(virtualIPFixture), nil
		}
		require.Equal(t, "verify", operation)
		require.Equal(t, expected, params)
		return []byte(`{"name":"cib","shortdesc":"Configure cluster resources","text":"primitive vip"}`), nil
	}
	fields := url.Values{}
	for name, value := range expected {
		fields.Set("virtual-ip."+name, value)
	}
	req := httptest.NewRequest("POST", "https://cluster/cib/live/wizards/virtual-ip", strings.NewReader(fields.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	serveVirtualIPWizard(response, req, run)
	require.Equal(t, 200, response.Code)
	require.Equal(t, []string{"show", "verify"}, operations)
	require.Contains(t, response.Body.String(), `value="Apply"`)
	cookies := response.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, "hawk_wizard_virtual_ip", cookies[0].Name)
	require.True(t, cookies[0].Secure)
	req = httptest.NewRequest("GET", "https://cluster/cib/live/wizards/virtual-ip", nil)
	req.AddCookie(cookies[0])
	require.Equal(t, expected, recalledWizardValues(req, "virtual-ip"))
	require.Empty(t, recalledWizardValues(req, "mailto"))
	response = httptest.NewRecorder()
	serveVirtualIPWizard(response, req, run)
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `value="192.0.2.10"`)
}

func TestSubmitVirtualIP(t *testing.T) {
	params := wizardParams{"id": "vip", "ip": "192.0.2.10", "nic": "eth0"}
	pa, _ := json.Marshal(params)
	for _, tt := range []struct {
		name              string
		verifyErr, runErr error
		status            int
		operations        []string
	}{
		{"success", nil, nil, 200, []string{"show", "verify", "run"}},
		{"verify fails", fmt.Errorf("Invalid IP address"), nil, 422, []string{"show", "verify"}},
		{"run fails", nil, fmt.Errorf("CIB update failed"), 422, []string{"show", "verify", "run"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			operations := []string{}
			run := func(_ context.Context, operation string, actual wizardParams) ([]byte, error) {
				operations = append(operations, operation)
				if operation == "show" {
					return []byte(virtualIPFixture), nil
				}
				require.Equal(t, params, actual)
				if operation == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure cluster resources"}`), tt.verifyErr
				}
				return []byte(`{"rc":true,"shortdesc":"Configure cluster resources"}`), tt.runErr
			}
			req := httptest.NewRequest("POST", "/cib/live/wizards/virtual-ip/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			submitVirtualIPWizard(response, req, run)
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, tt.operations, operations)
		})
	}
}

func TestVirtualIPRejectsOtherRequests(t *testing.T) {
	for _, tt := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/cib/live/wizards/virtual-ip/submit", 405},
		{"DELETE", "/cib/live/wizards/virtual-ip", 405},
		{"GET", "/cib/live/wizards/virtual-ip/unknown", 404},
	} {
		response := httptest.NewRecorder()
		serveVirtualIPWizard(response, httptest.NewRequest(tt.method, tt.path, nil), func(context.Context, string, wizardParams) ([]byte, error) {
			t.Fatal("must not execute")
			return nil, nil
		})
		require.Equal(t, tt.status, response.Code)
	}
}
