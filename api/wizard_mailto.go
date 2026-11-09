package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

type wizardParams map[string]string
type mailtoParams = wizardParams

type wizardParameter struct {
	Advanced  bool   `json:"advanced"`
	Label     string `json:"-"`
	Name      string `json:"name"`
	Shortdesc string `json:"shortdesc"`
	Longdesc  string `json:"longdesc"`
	Type      string `json:"type"`
	Required  bool   `json:"required"`
	Example   string `json:"example"`
	Value     string `json:"value"`
}

// crmsh returns actual booleans for values such as database.install.
// Keep the form-facing representation textual, as Rails' select_tag does.
func (p *wizardParameter) UnmarshalJSON(data []byte) error {
	type plain wizardParameter
	var value struct {
		*plain
		Value json.RawMessage `json:"value"`
	}
	value.plain = (*plain)(p)
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.Value) == 0 || string(value.Value) == "null" {
		p.Value = ""
		return nil
	}
	var text string
	if json.Unmarshal(value.Value, &text) == nil {
		p.Value = text
		return nil
	}
	var boolean bool
	if json.Unmarshal(value.Value, &boolean) == nil {
		p.Value = fmt.Sprint(boolean)
		return nil
	}
	var number json.Number
	if json.Unmarshal(value.Value, &number) == nil {
		p.Value = number.String()
		return nil
	}
	return fmt.Errorf("Unsupported wizard parameter value")
}

type mailtoParameter = wizardParameter

type wizardStep struct {
	Name       string            `json:"name"`
	Shortdesc  string            `json:"shortdesc"`
	Longdesc   string            `json:"longdesc"`
	Required   bool              `json:"required"`
	Parameters []wizardParameter `json:"parameters"`
	Steps      []wizardStep      `json:"steps"`
}
type resourceWizardDefinition struct {
	Shortdesc string       `json:"shortdesc"`
	Longdesc  string       `json:"longdesc"`
	Steps     []wizardStep `json:"steps"`
}

type mailtoDefinition = resourceWizardDefinition

func mailtoScriptCommand(ctx context.Context, operation string, params mailtoParams) (*exec.Cmd, error) {
	return resourceWizardScriptCommand(ctx, "mailto", operation, params)
}

func resourceWizardScriptCommand(ctx context.Context, name, operation string, params wizardParams) (*exec.Cmd, error) {
	return nestedWizardScriptCommand(ctx, name, operation, params)
}

func nestedWizardScriptCommand(ctx context.Context, name, operation string, params any) (*exec.Cmd, error) {
	if name != "mailto" && name != "virtual-ip" && name != "database" && name != "db2" && name != "db2-hadr" && name != "oracle" && name != "drbd" && name != "filesystem" {
		return nil, fmt.Errorf("Invalid resource wizard")
	}
	if operation != "show" && operation != "verify" && operation != "run" {
		return nil, fmt.Errorf("Invalid resource wizard operation")
	}
	args := []any{operation, name}
	if operation != "show" {
		args = append(args, params)
	}
	data, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	quoted := "script json \"" + strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(string(data)) + "\"\n"
	cmd := exec.CommandContext(ctx, "/usr/sbin/crm", "-f", "/dev/stdin")
	cmd.Stdin = strings.NewReader(quoted)
	return cmd, nil
}

type mailtoCommand func(context.Context, string, mailtoParams) ([]byte, error)

func runMailtoCommand(ctx context.Context, operation string, params mailtoParams) ([]byte, error) {
	return runResourceWizardCommand(ctx, "mailto", operation, params)
}

func runResourceWizardCommand(ctx context.Context, name, operation string, params wizardParams) ([]byte, error) {
	return runNestedWizardCommand(ctx, name, operation, params)
}
func runNestedWizardCommand(ctx context.Context, name, operation string, params any) ([]byte, error) {
	cmd, err := nestedWizardScriptCommand(ctx, name, operation, params)
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	if strings.TrimSpace(stderr.String()) != "" {
		return out, fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func mailtoResult(ctx context.Context, operation string, params mailtoParams, run mailtoCommand) healthResult {
	return wizardCommandResult(ctx, operation, func(ctx context.Context, operation string) ([]byte, error) { return run(ctx, operation, params) })
}

func validateMailtoParams(params mailtoParams) error {
	for name := range params {
		if name != "id" && name != "email" && name != "subject" {
			return fmt.Errorf("Invalid mail wizard parameter")
		}
	}
	if strings.TrimSpace(params["id"]) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.Contains(params["id"], " ") {
		return fmt.Errorf("Must be a valid resource ID")
	}
	if strings.TrimSpace(params["email"]) == "" {
		return fmt.Errorf("email is required")
	}
	return nil
}

// Retain form values for Back, as Ruby's one-hour wizard cache does. The cookie
// contains only editable form values; never trust it without re-verification.
func rememberMailto(w http.ResponseWriter, r *http.Request, params mailtoParams) {
	rememberWizardValues(w, r, "mailto", params)
}
func rememberWizardValues(w http.ResponseWriter, r *http.Request, name string, params wizardParams) {
	if name == "database" {
		rememberDatabaseValues(w, r, params)
		return
	}
	data, _ := json.Marshal(params)
	if len(data) > 2500 {
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "hawk_wizard_" + strings.ReplaceAll(name, "-", "_"), Value: base64.RawURLEncoding.EncodeToString(data), Path: "/cib/live/wizards/" + name, MaxAge: 3600, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})
}
func recalledMailto(r *http.Request) mailtoParams { return recalledWizardValues(r, "mailto") }
func recalledWizardValues(r *http.Request, name string) wizardParams {
	if name == "database" {
		return recalledDatabaseValues(r)
	}
	params := wizardParams{}
	if cookie, err := r.Cookie("hawk_wizard_" + strings.ReplaceAll(name, "-", "_")); err == nil {
		if data, err := base64.RawURLEncoding.DecodeString(cookie.Value); err == nil {
			_ = json.Unmarshal(data, &params)
		}
	}
	if params == nil {
		params = wizardParams{}
	}
	return params
}

func MailtoWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveMailtoWizard(w, r, runMailtoCommand)
}
func serveMailtoWizard(w http.ResponseWriter, r *http.Request, run mailtoCommand) {
	if r.URL.Path == "/cib/live/wizards/mailto/submit" || r.URL.Path == "/cib/live/wizards/mailto/submit/" {
		submitMailtoWizard(w, r, run)
		return
	}
	if r.URL.Path != "/cib/live/wizards/mailto" && r.URL.Path != "/cib/live/wizards/mailto/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost && r.Method != http.MethodPatch {
		w.Header().Set("Allow", "GET, HEAD, POST, PATCH")
		http.Error(w, "Method not allowed", 405)
		return
	}
	out, err := run(r.Context(), "show", nil)
	if err != nil {
		http.Error(w, "Unable to load mail wizard: "+err.Error(), 500)
		return
	}
	var definition mailtoDefinition
	if err := json.Unmarshal(bytes.TrimSpace(out), &definition); err != nil || len(definition.Steps) != 1 {
		http.Error(w, "Unable to load mail wizard definition", 500)
		return
	}
	params := recalledMailto(r)
	data := map[string]any{"Title": definition.Shortdesc, "Description": strings.Split(definition.Longdesc, "\n\n"), "StepDescription": definition.Steps[0].Longdesc, "Parameters": definition.Steps[0].Parameters, "Values": params}
	if r.Method == http.MethodPost || r.Method == http.MethodPatch {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		params = wizardParams{}
		for _, name := range []string{"id", "email", "subject"} {
			if value := r.FormValue("mailto." + name); value != "" {
				params[name] = value
			}
		}
		rememberMailto(w, r, params)
		result := healthResult{Actions: []healthAction{}, Errors: []string{}}
		if err := validateMailtoParams(params); err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else {
			result = mailtoResult(r.Context(), "verify", params, run)
		}
		data["VerifyMode"] = true
		data["Actions"] = result.Actions
		data["Errors"] = result.Errors
		pa, _ := json.Marshal(params)
		data["ParamsJSON"] = string(pa)
	}
	renderTemplate(w, "wizard_mailto", data)
}

func submitMailtoWizard(w http.ResponseWriter, r *http.Request, run mailtoCommand) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	var params mailtoParams
	if err := json.Unmarshal([]byte(r.FormValue("pa")), &params); err != nil || params == nil {
		http.Error(w, "Invalid mail wizard parameters", 400)
		return
	}
	if err := validateMailtoParams(params); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	result := mailtoResult(r.Context(), "verify", params, run)
	if len(result.Errors) == 0 {
		expected := len(result.Actions)
		result = mailtoResult(r.Context(), "run", params, run)
		if len(result.Errors) == 0 && len(result.Actions) != expected {
			result.Errors = append(result.Errors, fmt.Sprintf("Mail wizard returned %d of %d expected action results", len(result.Actions), expected))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if len(result.Errors) > 0 {
		w.WriteHeader(422)
		_ = json.NewEncoder(w).Encode(result.Errors)
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}
