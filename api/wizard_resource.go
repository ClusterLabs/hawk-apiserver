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

func resourceWizardResult(ctx context.Context, operation string, params wizardParams, run resourceWizardCommand) healthResult {
	return wizardCommandResult(ctx, operation, func(ctx context.Context, operation string) ([]byte, error) { return run(ctx, operation, params) })
}
func loadResourceWizardDefinition(ctx context.Context, run resourceWizardCommand, title string) (resourceWizardDefinition, error) {
	out, err := run(ctx, "show", nil)
	if err != nil {
		return resourceWizardDefinition{}, err
	}
	var definition resourceWizardDefinition
	if err := json.Unmarshal(bytes.TrimSpace(out), &definition); err != nil {
		return definition, err
	}
	if len(definition.Steps) != 1 {
		return definition, fmt.Errorf("Unsupported %s wizard definition", title)
	}
	for i := range definition.Steps[0].Parameters {
		p := &definition.Steps[0].Parameters[i]
		if p.Type == "boolean" {
			switch strings.ToLower(p.Example) {
			case "yes", "1":
				p.Example = "true"
			case "no", "0":
				p.Example = "false"
			}
		}
		p.Label = strings.NewReplacer("_", " ", "-", " ").Replace(p.Name)
		// WizardHelper.sanitize_value suppresses unresolved template expressions.
		if strings.Contains(p.Value, "{{") {
			p.Value = ""
		}
	}
	return definition, nil
}

var wizardDigits = regexp.MustCompile(`^[0-9]+$`)

func validateResourceWizardParams(params wizardParams, definition resourceWizardDefinition, title string) error {
	allowed := map[string]wizardParameter{}
	for _, p := range definition.Steps[0].Parameters {
		allowed[p.Name] = p
	}
	for name := range params {
		if _, ok := allowed[name]; !ok {
			return fmt.Errorf("Invalid %s wizard parameter: %s", title, name)
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

type resourceWizardConfig struct{ Name, Title string }

func serveResourceWizard(w http.ResponseWriter, r *http.Request, run resourceWizardCommand, config resourceWizardConfig) {
	base := "/cib/live/wizards/" + config.Name
	if r.URL.Path == base+"/submit" || r.URL.Path == base+"/submit/" {
		submitResourceWizard(w, r, run, config)
		return
	}
	if r.URL.Path != base && r.URL.Path != base+"/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost && r.Method != http.MethodPatch {
		w.Header().Set("Allow", "GET, HEAD, POST, PATCH")
		http.Error(w, "Method not allowed", 405)
		return
	}
	definition, err := loadResourceWizardDefinition(r.Context(), run, config.Title)
	if err != nil {
		http.Error(w, "Unable to load "+config.Title+" wizard: "+err.Error(), 500)
		return
	}
	params := recalledWizardValues(r, config.Name)
	basic := []wizardParameter{}
	advanced := []wizardParameter{}
	for _, p := range definition.Steps[0].Parameters {
		if p.Advanced {
			advanced = append(advanced, p)
		} else {
			basic = append(basic, p)
		}
	}
	data := map[string]any{"Title": definition.Shortdesc, "Description": strings.Split(definition.Longdesc, "\n\n"), "StepDescription": definition.Steps[0].Longdesc, "Parameters": definition.Steps[0].Parameters, "Basic": basic, "Advanced": advanced, "Values": params, "WizardName": config.Name}
	if r.Method == http.MethodPost || r.Method == http.MethodPatch {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		params = wizardParams{}
		for _, p := range definition.Steps[0].Parameters {
			if value := r.FormValue(config.Name + "." + p.Name); value != "" {
				params[p.Name] = value
			}
		}
		rememberWizardValues(w, r, config.Name, params)
		result := healthResult{Actions: []healthAction{}, Errors: []string{}}
		if err := validateResourceWizardParams(params, definition, config.Title); err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else {
			result = resourceWizardResult(r.Context(), "verify", params, run)
		}
		data["VerifyMode"] = true
		data["Actions"] = result.Actions
		data["Errors"] = result.Errors
		pa, _ := json.Marshal(params)
		data["ParamsJSON"] = string(pa)
	}
	renderTemplate(w, "wizard_resource", data)
}
func submitResourceWizard(w http.ResponseWriter, r *http.Request, run resourceWizardCommand, config resourceWizardConfig) {
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
		http.Error(w, "Invalid "+config.Title+" wizard parameters", 400)
		return
	}
	definition, err := loadResourceWizardDefinition(r.Context(), run, config.Title)
	if err != nil {
		http.Error(w, "Unable to load "+config.Title+" wizard: "+err.Error(), 500)
		return
	}
	if err := validateResourceWizardParams(params, definition, config.Title); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	result := resourceWizardResult(r.Context(), "verify", params, run)
	if len(result.Errors) == 0 {
		expected := len(result.Actions)
		result = resourceWizardResult(r.Context(), "run", params, run)
		if len(result.Errors) == 0 && len(result.Actions) != expected {
			result.Errors = append(result.Errors, fmt.Sprintf("%s wizard returned %d of %d expected action results", config.Title, len(result.Actions), expected))
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
