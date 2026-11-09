package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type db2Params map[string]any
type db2Command func(context.Context, string, db2Params) ([]byte, error)
type db2Field struct {
	wizardParameter
	FieldID   string
	HelpID    string
	Path      []string
	Current   string
	StepIndex int
}
type db2Group struct {
	Title       string
	Prefix      string
	Basic       []db2Field
	Advanced    []wizardParameter
	Values      wizardParams
	ContainerID string
}
type db2StepView struct {
	Index       int
	Number      int
	Title       string
	Description []string
	Groups      []db2Group
}
type db2AdvancedGroup struct {
	ContainerID string            `json:"containerID"`
	Prefix      string            `json:"prefix"`
	Parameters  []wizardParameter `json:"parameters"`
	Values      wizardParams      `json:"values"`
}
type db2View struct {
	Steps    []db2StepView
	Fields   []db2Field
	Advanced []db2AdvancedGroup
	Scopes   map[string]bool
}

func runDB2Command(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	return runNestedWizardCommand(ctx, "db2", operation, params)
}
func loadDB2Definition(ctx context.Context, run db2Command) (resourceWizardDefinition, error) {
	data, err := run(ctx, "show", nil)
	if err != nil {
		return resourceWizardDefinition{}, err
	}
	var definition resourceWizardDefinition
	if err := json.Unmarshal(data, &definition); err != nil {
		return definition, err
	}
	if len(definition.Steps) == 0 {
		return definition, fmt.Errorf("DB2 wizard has no steps")
	}
	return definition, nil
}
func db2Value(params db2Params, path []string) string {
	var node any = map[string]any(params)
	for _, part := range path {
		m, ok := node.(map[string]any)
		if !ok {
			return ""
		}
		node = m[part]
	}
	value, _ := node.(string)
	return value
}
func buildDB2View(definition resourceWizardDefinition, params db2Params) db2View {
	return buildNestedWizardView(definition, params, "db2")
}
func buildNestedWizardView(definition resourceWizardDefinition, params db2Params, name string) db2View {
	view := db2View{Steps: []db2StepView{}, Fields: []db2Field{}, Advanced: []db2AdvancedGroup{}, Scopes: map[string]bool{}}
	var groups func(wizardStep, []string, string, int, bool) []db2Group
	groups = func(step wizardStep, parent []string, parentTitle string, index int, top bool) []db2Group {
		path := append([]string{}, parent...)
		if step.Name != "" {
			path = append(path, step.Name)
			view.Scopes[strings.Join(path, ".")] = true
		}
		title := strings.TrimSpace(step.Shortdesc)
		if title == "" {
			title = parentTitle
		}
		prefix := name
		if len(path) > 0 {
			prefix += "." + strings.Join(path, ".")
		}
		group := db2Group{Prefix: prefix, Basic: []db2Field{}, Advanced: []wizardParameter{}, Values: wizardParams{}, ContainerID: fmt.Sprintf("%s-advanced-%d", name, len(view.Advanced))}
		if !top && title != parentTitle {
			group.Title = title
		}
		for _, p := range step.Parameters {
			p.Label = strings.NewReplacer("_", " ", "-", " ").Replace(p.Name)
			if strings.Contains(p.Value, "{{") {
				p.Value = ""
			}
			if p.Type == "boolean" {
				switch strings.ToLower(p.Example) {
				case "yes", "1":
					p.Example = "true"
				case "no", "0":
					p.Example = "false"
				}
			}
			fieldPath := append(append([]string{}, path...), p.Name)
			current := db2Value(params, fieldPath)
			if current == "" {
				current = p.Value
			}
			id := prefix + "." + p.Name
			field := db2Field{wizardParameter: p, FieldID: id, HelpID: strings.ReplaceAll(id, ".", "-"), Path: fieldPath, Current: current, StepIndex: index}
			view.Fields = append(view.Fields, field)
			if p.Advanced {
				group.Advanced = append(group.Advanced, p)
				if current != "" {
					group.Values[p.Name] = current
				}
			} else {
				group.Basic = append(group.Basic, field)
			}
		}
		result := []db2Group{}
		if len(group.Basic) > 0 || len(group.Advanced) > 0 || group.Title != "" {
			if len(group.Advanced) > 0 {
				view.Advanced = append(view.Advanced, db2AdvancedGroup{group.ContainerID, prefix, group.Advanced, group.Values})
			}
			result = append(result, group)
		}
		for _, sub := range step.Steps {
			result = append(result, groups(sub, path, title, index, false)...)
		}
		return result
	}
	for i, step := range definition.Steps {
		title := strings.TrimSpace(step.Shortdesc)
		if title == "" {
			title = "db2"
		}
		paragraphs := []string{}
		if strings.TrimSpace(step.Longdesc) != "" {
			paragraphs = strings.Split(step.Longdesc, "\n\n")
		}
		view.Steps = append(view.Steps, db2StepView{Index: i, Number: i + 1, Title: title, Description: paragraphs, Groups: groups(step, nil, title, i, true)})
	}
	return view
}
func setDB2Value(params db2Params, path []string, value string) {
	node := map[string]any(params)
	for _, part := range path[:len(path)-1] {
		child, ok := node[part].(map[string]any)
		if !ok {
			child = map[string]any{}
			node[part] = child
		}
		node = child
	}
	node[path[len(path)-1]] = value
}
func db2FormParams(r *http.Request, definition resourceWizardDefinition, view db2View) db2Params {
	params := db2Params{}
	// WizardsController.build_stepmap includes empty required named steps.
	for _, step := range definition.Steps {
		if step.Required && step.Name != "" {
			params[step.Name] = map[string]any{}
		}
	}
	for _, field := range view.Fields {
		if value := r.FormValue(field.FieldID); value != "" {
			setDB2Value(params, field.Path, value)
		}
	}
	return params
}
func validateDB2Params(params db2Params, view db2View) error {
	allowed := map[string]db2Field{}
	for _, field := range view.Fields {
		allowed[strings.Join(field.Path, ".")] = field
	}
	var check func(map[string]any, []string) error
	check = func(node map[string]any, path []string) error {
		for name, value := range node {
			current := append(append([]string{}, path...), name)
			key := strings.Join(current, ".")
			if child, ok := value.(map[string]any); ok {
				if !view.Scopes[key] {
					return fmt.Errorf("Invalid DB2 wizard parameter: %s", key)
				}
				if err := check(child, current); err != nil {
					return err
				}
				continue
			}
			if _, ok := allowed[key]; !ok {
				return fmt.Errorf("Invalid DB2 wizard parameter: %s", key)
			}
			if _, ok := value.(string); !ok {
				return fmt.Errorf("Invalid value for %s", key)
			}
		}
		return nil
	}
	if err := check(map[string]any(params), nil); err != nil {
		return err
	}
	for _, field := range view.Fields {
		value := db2Value(params, field.Path)
		if field.Required && strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field.Label)
		}
		if field.Type == "resource" && strings.Contains(value, " ") {
			return fmt.Errorf("Must be a valid resource ID")
		}
		if !field.Advanced && (field.Type == "integer" || field.Type == "port") && value != "" && !wizardDigits.MatchString(value) {
			return fmt.Errorf("%s must contain only digits", field.Label)
		}
	}
	return nil
}
func db2Result(ctx context.Context, operation string, params db2Params, run db2Command) healthResult {
	return wizardCommandResult(ctx, operation, func(ctx context.Context, op string) ([]byte, error) { return run(ctx, op, params) })
}
func DB2WizardHandler(w http.ResponseWriter, r *http.Request) { serveDB2Wizard(w, r, runDB2Command) }
func serveDB2Wizard(w http.ResponseWriter, r *http.Request, run db2Command) {
	serveNestedWizard(w, r, run, resourceWizardConfig{Name: "db2", Title: "DB2"})
}
func serveNestedWizard(w http.ResponseWriter, r *http.Request, run db2Command, config resourceWizardConfig) {
	base := "/cib/live/wizards/" + config.Name
	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == base+"/submit" {
		submitNestedWizard(w, r, run, config)
		return
	}
	if path != base {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost && r.Method != http.MethodPatch {
		w.Header().Set("Allow", "GET, HEAD, POST, PATCH")
		http.Error(w, "Method not allowed", 405)
		return
	}
	definition, err := loadDB2Definition(r.Context(), run)
	if err != nil {
		http.Error(w, "Unable to load "+config.Title+" wizard: "+err.Error(), 500)
		return
	}
	params := db2Params{}
	_ = json.Unmarshal(recalledEncryptedWizardValues(r, config.Name), &params)
	if params == nil {
		params = db2Params{}
	}
	view := buildNestedWizardView(definition, params, config.Name)
	data := map[string]any{"Title": definition.Shortdesc, "Description": strings.Split(definition.Longdesc, "\n\n"), "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced, "WizardName": config.Name}
	if r.Method == http.MethodPost || r.Method == http.MethodPatch {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		params = db2FormParams(r, definition, view)
		rememberEncryptedWizardValues(w, r, config.Name, params)
		result := healthResult{Actions: []healthAction{}, Errors: []string{}}
		if err := validateDB2Params(params, view); err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else {
			result = db2Result(r.Context(), "verify", params, run)
		}
		data["VerifyMode"] = true
		data["Actions"] = result.Actions
		data["Errors"] = result.Errors
		pa, _ := json.Marshal(params)
		data["ParamsJSON"] = string(pa)
	}
	renderTemplate(w, "wizard_db2", data)
}
func submitDB2Wizard(w http.ResponseWriter, r *http.Request, run db2Command) {
	submitNestedWizard(w, r, run, resourceWizardConfig{Name: "db2", Title: "DB2"})
}
func submitNestedWizard(w http.ResponseWriter, r *http.Request, run db2Command, config resourceWizardConfig) {
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
	var params db2Params
	if json.Unmarshal([]byte(r.FormValue("pa")), &params) != nil || params == nil {
		http.Error(w, "Invalid "+config.Title+" wizard parameters", 400)
		return
	}
	definition, err := loadDB2Definition(r.Context(), run)
	if err != nil {
		http.Error(w, "Unable to load "+config.Title+" wizard: "+err.Error(), 500)
		return
	}
	if err := validateDB2Params(params, buildNestedWizardView(definition, nil, config.Name)); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	result := db2Result(r.Context(), "verify", params, run)
	if len(result.Errors) == 0 {
		expected := len(result.Actions)
		result = db2Result(r.Context(), "run", params, run)
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
