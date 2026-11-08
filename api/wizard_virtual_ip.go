package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type resourceWizardCommand func(context.Context, string, wizardParams) ([]byte, error)

func runVirtualIPCommand(ctx context.Context, operation string, params wizardParams) ([]byte, error) {
	return runResourceWizardCommand(ctx, "virtual-ip", operation, params)
}
func virtualIPResult(ctx context.Context, operation string, params wizardParams, run resourceWizardCommand) healthResult {
	return wizardCommandResult(ctx, operation, func(ctx context.Context, operation string) ([]byte, error) { return run(ctx, operation, params) })
}
func loadVirtualIPDefinition(ctx context.Context, run resourceWizardCommand) (resourceWizardDefinition, error) {
	out, err := run(ctx, "show", nil)
	if err != nil {
		return resourceWizardDefinition{}, err
	}
	var definition resourceWizardDefinition
	if err := json.Unmarshal(bytes.TrimSpace(out), &definition); err != nil {
		return definition, err
	}
	if len(definition.Steps) != 1 {
		return definition, fmt.Errorf("Unsupported Virtual IP wizard definition")
	}
	for i := range definition.Steps[0].Parameters {
		p := &definition.Steps[0].Parameters[i]
		p.Label = strings.NewReplacer("_", " ", "-", " ").Replace(p.Name)
		// WizardHelper.sanitize_value suppresses unresolved template expressions.
		if strings.Contains(p.Value, "{{") {
			p.Value = ""
		}
	}
	return definition, nil
}

var wizardDigits = regexp.MustCompile(`^[0-9]+$`)

func validateVirtualIPParams(params wizardParams, definition resourceWizardDefinition) error {
	allowed := map[string]wizardParameter{}
	for _, p := range definition.Steps[0].Parameters {
		allowed[p.Name] = p
	}
	for name := range params {
		if _, ok := allowed[name]; !ok {
			return fmt.Errorf("Invalid Virtual IP wizard parameter: %s", name)
		}
	}
	for _, p := range definition.Steps[0].Parameters {
		value := params[p.Name]
		if p.Required && strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", p.Label)
		}
		if p.Type == "resource" && strings.Contains(value, " ") {
			return fmt.Errorf("Must be a valid resource ID")
		}
		if !p.Advanced && (p.Type == "integer" || p.Type == "port") && value != "" && !wizardDigits.MatchString(value) {
			return fmt.Errorf("%s must contain only digits", p.Label)
		}
	}
	// crmsh remains responsible for IP address, RA, and cluster validation.
	return nil
}
func VirtualIPWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveVirtualIPWizard(w, r, runVirtualIPCommand)
}
func serveVirtualIPWizard(w http.ResponseWriter, r *http.Request, run resourceWizardCommand) {
	if r.URL.Path == "/cib/live/wizards/virtual-ip/submit" || r.URL.Path == "/cib/live/wizards/virtual-ip/submit/" {
		submitVirtualIPWizard(w, r, run)
		return
	}
	if r.URL.Path != "/cib/live/wizards/virtual-ip" && r.URL.Path != "/cib/live/wizards/virtual-ip/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost && r.Method != http.MethodPatch {
		w.Header().Set("Allow", "GET, HEAD, POST, PATCH")
		http.Error(w, "Method not allowed", 405)
		return
	}
	definition, err := loadVirtualIPDefinition(r.Context(), run)
	if err != nil {
		http.Error(w, "Unable to load Virtual IP wizard: "+err.Error(), 500)
		return
	}
	params := recalledWizardValues(r, "virtual-ip")
	basic := []wizardParameter{}
	advanced := []wizardParameter{}
	for _, p := range definition.Steps[0].Parameters {
		if p.Advanced {
			advanced = append(advanced, p)
		} else {
			basic = append(basic, p)
		}
	}
	data := map[string]any{"Title": definition.Shortdesc, "Description": strings.Split(definition.Longdesc, "\n\n"), "StepDescription": definition.Steps[0].Longdesc, "Parameters": definition.Steps[0].Parameters, "Basic": basic, "Advanced": advanced, "Values": params}
	if r.Method == http.MethodPost || r.Method == http.MethodPatch {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		params = wizardParams{}
		for _, p := range definition.Steps[0].Parameters {
			if value := r.FormValue("virtual-ip." + p.Name); value != "" {
				params[p.Name] = value
			}
		}
		rememberWizardValues(w, r, "virtual-ip", params)
		result := healthResult{Actions: []healthAction{}, Errors: []string{}}
		if err := validateVirtualIPParams(params, definition); err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else {
			result = virtualIPResult(r.Context(), "verify", params, run)
		}
		data["VerifyMode"] = true
		data["Actions"] = result.Actions
		data["Errors"] = result.Errors
		pa, _ := json.Marshal(params)
		data["ParamsJSON"] = string(pa)
	}
	renderTemplate(w, "wizard_virtual_ip", data)
}
func submitVirtualIPWizard(w http.ResponseWriter, r *http.Request, run resourceWizardCommand) {
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
	var params wizardParams
	if err := json.Unmarshal([]byte(r.FormValue("pa")), &params); err != nil || params == nil {
		http.Error(w, "Invalid Virtual IP wizard parameters", 400)
		return
	}
	definition, err := loadVirtualIPDefinition(r.Context(), run)
	if err != nil {
		http.Error(w, "Unable to load Virtual IP wizard: "+err.Error(), 500)
		return
	}
	if err := validateVirtualIPParams(params, definition); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	result := virtualIPResult(r.Context(), "verify", params, run)
	if len(result.Errors) == 0 {
		expected := len(result.Actions)
		result = virtualIPResult(r.Context(), "run", params, run)
		if len(result.Errors) == 0 && len(result.Actions) != expected {
			result.Errors = append(result.Errors, fmt.Sprintf("Virtual IP wizard returned %d of %d expected action results", len(result.Actions), expected))
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
