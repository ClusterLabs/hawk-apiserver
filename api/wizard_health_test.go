package api

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthWizardTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/wizard_health.html")
	require.NoError(t, err)
	for _, verify := range []bool{false, true} {
		var body bytes.Buffer
		require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": "Verify health and configuration", "VerifyMode": verify, "Actions": []healthAction{{Shortdesc: "Collect information", Text: "<unsafe>"}}}))
		require.Contains(t, body.String(), `id="edit_wizard_health"`)
		require.Contains(t, body.String(), `name="submit"`)
		if verify {
			require.Contains(t, body.String(), `action="/cib/live/wizards/health/submit"`)
			require.Contains(t, body.String(), `name="pa" id="pa" value="{}"`)
			require.Contains(t, body.String(), `value="Apply"`)
			require.Contains(t, body.String(), `&lt;unsafe&gt;`)
		} else {
			require.Contains(t, body.String(), `action="/cib/live/wizards/health"`)
			require.Contains(t, body.String(), `value="Verify"`)
			require.Contains(t, body.String(), "This wizard takes no parameters.")
		}
	}
}

func TestHealthCommandResult(t *testing.T) {
	result := healthCommandResult(context.Background(), "verify", func(context.Context, string) ([]byte, error) {
		return []byte("{\"name\":\"collect\",\"shortdesc\":\"Collect information\",\"text\":\"a\\tb\"}\n{\"error\":\"SSH unavailable\"}\n"), nil
	})
	require.Len(t, result.Actions, 1)
	require.Equal(t, "a    b", result.Actions[0].Text)
	require.Equal(t, []string{"SSH unavailable"}, result.Errors)
	result = healthCommandResult(context.Background(), "run", func(context.Context, string) ([]byte, error) {
		return []byte("{\"output\":\"first\"}\n{\"output\":\"final report\"}\n"), nil
	})
	require.Equal(t, "final report", result.Output)
}

func TestSubmitHealthWizard(t *testing.T) {
	for _, tt := range []struct {
		name              string
		verifyErr, runErr error
		status            int
		operations        []string
	}{
		{"success", nil, nil, 200, []string{"verify", "run"}},
		{"verification fails", fmt.Errorf("Invalid configuration"), nil, 422, []string{"verify"}},
		{"execution fails", nil, fmt.Errorf("SSH unavailable"), 422, []string{"verify", "run"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			operations := []string{}
			run := func(_ context.Context, operation string) ([]byte, error) {
				operations = append(operations, operation)
				if operation == "verify" {
					return []byte(`{"name":"collect","shortdesc":"Collect information"}`), tt.verifyErr
				}
				return []byte(`{"shortdesc":"Report cluster state","output":"Report text"}`), tt.runErr
			}
			req := httptest.NewRequest("POST", "/cib/live/wizards/health/submit", strings.NewReader(url.Values{"pa": {"{}"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			submitHealthWizard(response, req, run)
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, tt.operations, operations)
			if tt.status == 200 {
				require.Contains(t, response.Body.String(), `"output":"Report text"`)
			}
		})
	}
}

func TestHealthWizardRejectsInvalidRequests(t *testing.T) {
	run := func(context.Context, string) ([]byte, error) { t.Fatal("must not execute"); return nil, nil }
	for _, params := range []string{"", "null", `{"nodes":"remote"}`, `[]`} {
		req := httptest.NewRequest("POST", "/cib/live/wizards/health/submit", strings.NewReader(url.Values{"pa": {params}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		submitHealthWizard(response, req, run)
		require.Equal(t, 400, response.Code)
	}
	for _, tt := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/cib/live/wizards/health/submit", 405},
		{"DELETE", "/cib/live/wizards/health", 405},
		{"GET", "/cib/live/wizards/health/unknown", 404},
	} {
		response := httptest.NewRecorder()
		serveHealthWizard(response, httptest.NewRequest(tt.method, tt.path, nil), run)
		require.Equal(t, tt.status, response.Code)
	}
}

func TestHealthScriptCommand(t *testing.T) {
	for _, operation := range []string{"show", "verify", "run"} {
		cmd, err := healthScriptCommand(context.Background(), operation)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		expected := `script json "[\"` + operation + `\",\"health\"`
		if operation != "show" {
			expected += `,{}`
		}
		expected += `]"` + "\n"
		require.Equal(t, expected, string(input))
	}
	_, err := healthScriptCommand(context.Background(), "delete")
	require.Error(t, err)
}

func TestHealthExecutionResults(t *testing.T) {
	for _, tt := range []struct {
		name, output string
		failure      bool
	}{
		{"empty response", "", true},
		{"boolean failure", `{"rc":false,"shortdesc":"Collect information","output":"SSH failed"}`, true},
		{"boolean success", `{"rc":true,"shortdesc":"Report","output":"Health report"}`, false},
		{"exit code failure", `{"rc":1,"shortdesc":"Collect information"}`, true},
		{"exit code success", `{"rc":0,"shortdesc":"Report","output":"Health report"}`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := healthCommandResult(context.Background(), "run", func(context.Context, string) ([]byte, error) { return []byte(tt.output), nil })
			require.Equal(t, tt.failure, len(result.Errors) > 0)
		})
	}
}

func TestSubmitHealthRejectsIncompleteExecution(t *testing.T) {
	run := func(_ context.Context, operation string) ([]byte, error) {
		if operation == "verify" {
			return []byte("{\"shortdesc\":\"Collect\"}\n{\"shortdesc\":\"Check\"}\n{\"shortdesc\":\"Report\"}\n"), nil
		}
		return []byte(`{"rc":true,"shortdesc":"Collect","output":"collected"}`), nil
	}
	req := httptest.NewRequest("POST", "/cib/live/wizards/health/submit", strings.NewReader(url.Values{"pa": {"{}"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	submitHealthWizard(response, req, run)
	require.Equal(t, 422, response.Code)
	require.Contains(t, response.Body.String(), "1 of 3 expected action results")
}
