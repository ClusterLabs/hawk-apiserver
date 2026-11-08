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

func TestMailtoScriptCommand(t *testing.T) {
	params := mailtoParams{"id": "mail1", "email": "admin@example.test", "subject": "Report \"quoted\" $HOME\nnext line"}
	for _, operation := range []string{"show", "verify", "run"} {
		cmd, err := mailtoScriptCommand(context.Background(), operation, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"`+operation+`"`, string(args[0]))
		require.JSONEq(t, `"mailto"`, string(args[1]))
		if operation == "show" {
			require.Len(t, args, 2)
		} else {
			var actual mailtoParams
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
	_, err := mailtoScriptCommand(context.Background(), "delete", params)
	require.Error(t, err)
}

func TestMailtoTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/wizard_mailto.html")
	require.NoError(t, err)
	params := []mailtoParameter{{Name: "id", Type: "resource", Required: true, Shortdesc: "Identifier"}, {Name: "email", Type: "email", Required: true, Shortdesc: "Email address"}, {Name: "subject", Type: "string", Example: "Resource Group"}}
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": "E-Mail", "Parameters": params, "Values": mailtoParams{"id": "mail1", "email": "admin@example.test", "subject": "<unsafe>"}}))
	for _, value := range []string{`id="edit_wizard_mailto"`, `id="mailto.id"`, `name="mailto.email"`, `name="mailto.subject"`, `type="email"`, `placeholder="Example: Resource Group"`, `value="&lt;unsafe&gt;"`, `name="submit" value="Verify"`} {
		require.Contains(t, body.String(), value)
	}
	body.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": "E-Mail", "Parameters": params, "Values": mailtoParams{}}))
	require.NotContains(t, body.String(), `value="Resource Group"`)
	body.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": "E-Mail", "VerifyMode": true, "ParamsJSON": `{"id":"mail1","email":"admin@example.test"}`, "Errors": []string{"Invalid configuration"}, "Actions": []healthAction{{Number: 1, Shortdesc: "Configure mail", Text: "<unsafe>"}}}))
	for _, value := range []string{`action="/cib/live/wizards/mailto/submit"`, `name="pa"`, `value="Apply"`, `disabled`, `&lt;unsafe&gt;`, `href="/cib/live/wizards/mailto"`} {
		require.Contains(t, body.String(), value)
	}
}

func TestMailtoRememberedValues(t *testing.T) {
	params := mailtoParams{"id": "mail1", "email": "admin@example.test"}
	req := httptest.NewRequest("GET", "https://cluster/cib/live/wizards/mailto", nil)
	response := httptest.NewRecorder()
	rememberMailto(response, req, params)
	cookies := response.Result().Cookies()
	require.Len(t, cookies, 1)
	require.True(t, cookies[0].HttpOnly)
	require.True(t, cookies[0].Secure)
	require.Equal(t, 3600, cookies[0].MaxAge)
	req.AddCookie(cookies[0])
	require.Equal(t, params, recalledMailto(req))
}

func TestSubmitMailto(t *testing.T) {
	params := mailtoParams{"id": "mail1", "email": "admin@example.test", "subject": "Cluster takeover"}
	data, _ := json.Marshal(params)
	for _, tt := range []struct {
		name              string
		verifyErr, runErr error
		status            int
		operations        []string
	}{
		{"success", nil, nil, 200, []string{"verify", "run"}},
		{"verification fails", fmt.Errorf("Resource already exists"), nil, 422, []string{"verify"}},
		{"execution fails", nil, fmt.Errorf("Installation failed"), 422, []string{"verify", "run"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			operations := []string{}
			run := func(_ context.Context, operation string, actual mailtoParams) ([]byte, error) {
				require.Equal(t, params, actual)
				operations = append(operations, operation)
				if operation == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure mail"}`), tt.verifyErr
				}
				return []byte(`{"rc":true,"shortdesc":"Configure mail"}`), tt.runErr
			}
			req := httptest.NewRequest("POST", "/cib/live/wizards/mailto/submit", strings.NewReader(url.Values{"pa": {string(data)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			submitMailtoWizard(response, req, run)
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, tt.operations, operations)
		})
	}
}

func TestMailtoRejectsInvalidRequests(t *testing.T) {
	run := func(context.Context, string, mailtoParams) ([]byte, error) {
		t.Fatal("must not execute")
		return nil, nil
	}
	for _, pa := range []string{`{}`, `null`, `{"id":"mail1"}`, `{"id":"bad id","email":"a@b"}`, `{"id":"mail1","email":"a@b","dry_run":"yes"}`} {
		req := httptest.NewRequest("POST", "/cib/live/wizards/mailto/submit", strings.NewReader(url.Values{"pa": {pa}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		submitMailtoWizard(response, req, run)
		require.Equal(t, 400, response.Code)
	}
	for _, tt := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/cib/live/wizards/mailto/submit", 405},
		{"DELETE", "/cib/live/wizards/mailto", 405},
		{"GET", "/cib/live/wizards/mailto/unknown", 404},
	} {
		response := httptest.NewRecorder()
		serveMailtoWizard(response, httptest.NewRequest(tt.method, tt.path, nil), run)
		require.Equal(t, tt.status, response.Code)
	}
}

func TestMailtoVerifyFormHandler(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	operations := []string{}
	run := func(_ context.Context, operation string, params mailtoParams) ([]byte, error) {
		operations = append(operations, operation)
		if operation == "show" {
			return []byte(`{"shortdesc":"E-Mail","longdesc":"Mail wizard","steps":[{"parameters":[{"name":"id","type":"resource","required":true},{"name":"email","type":"email","required":true},{"name":"subject","type":"string","example":"Resource Group"}]}]}`), nil
		}
		require.Equal(t, "verify", operation)
		require.Equal(t, mailtoParams{"id": "mail1", "email": "admin@example.test"}, params)
		return []byte(`{"name":"cib","shortdesc":"Configure mail","text":"primitive mail1"}`), nil
	}
	req := httptest.NewRequest("POST", "/cib/live/wizards/mailto", strings.NewReader(url.Values{"mailto.id": {"mail1"}, "mailto.email": {"admin@example.test"}, "mailto.subject": {""}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	serveMailtoWizard(response, req, run)
	require.Equal(t, 200, response.Code)
	require.Equal(t, []string{"show", "verify"}, operations)
	require.Contains(t, response.Body.String(), `value="Apply"`)
	require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
	require.Len(t, response.Result().Cookies(), 1)
	// Back must restore the same form fields, without pre-filling the subject example.
	req = httptest.NewRequest("GET", "/cib/live/wizards/mailto", nil)
	req.AddCookie(response.Result().Cookies()[0])
	response = httptest.NewRecorder()
	serveMailtoWizard(response, req, run)
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `value="mail1"`)
	require.Contains(t, response.Body.String(), `value="admin@example.test"`)
	require.NotContains(t, response.Body.String(), `value="Resource Group"`)
}
