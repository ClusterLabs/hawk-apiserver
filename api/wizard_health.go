package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type healthAction struct {
	Number    int             `json:"-"`
	RC        json.RawMessage `json:"rc,omitempty"`
	Name      string          `json:"name"`
	Shortdesc string          `json:"shortdesc"`
	Longdesc  string          `json:"longdesc"`
	Text      string          `json:"text"`
	Output    string          `json:"output,omitempty"`
	Error     string          `json:"error,omitempty"`
}
type healthResult struct {
	Actions []healthAction `json:"actions"`
	Output  string         `json:"output"`
	Errors  []string       `json:"errors"`
}

type healthCommand func(context.Context, string) ([]byte, error)

func healthScriptCommand(ctx context.Context, operation string) (*exec.Cmd, error) {
	if operation != "show" && operation != "verify" && operation != "run" {
		return nil, fmt.Errorf("Invalid health wizard operation")
	}
	args := []any{operation, "health"}
	if operation != "show" {
		args = append(args, map[string]any{})
	}
	data, _ := json.Marshal(args)
	quoted := "script json \"" + strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(string(data)) + "\"\n"
	cmd := exec.CommandContext(ctx, "/usr/sbin/crm", "-f", "/dev/stdin")
	cmd.Stdin = strings.NewReader(quoted)
	return cmd, nil
}

func runHealthCommand(ctx context.Context, operation string) (out []byte, err error) {
	started := time.Now()
	log.Printf("Health wizard %s started", operation)
	defer func() {
		log.Printf("Health wizard %s finished: duration=%s success=%t output_bytes=%d", operation, time.Since(started).Round(time.Millisecond), err == nil, len(out))
	}()
	cmd, err := healthScriptCommand(ctx, operation)
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err = cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	// CrmScript.run treats stderr as a verification/execution error.
	if strings.TrimSpace(stderr.String()) != "" {
		return out, fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func healthCommandResult(ctx context.Context, operation string, run healthCommand) healthResult {
	return wizardCommandResult(ctx, operation, run)
}

// Both wizard interfaces use crmsh's newline-delimited JSON action results.
func wizardCommandResult(ctx context.Context, operation string, run healthCommand) healthResult {
	out, err := run(ctx, operation)
	result := healthResult{Actions: []healthAction{}, Errors: []string{}}
	if err != nil {
		result.Errors = append(result.Errors, strings.TrimSpace(stripANSI(err.Error())))
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 4096), 8<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "Password:") {
			continue
		}
		var action healthAction
		if err := json.Unmarshal([]byte(line), &action); err != nil {
			result.Errors = append(result.Errors, line)
			continue
		}
		if operation == "run" && len(action.RC) > 0 {
			// crmsh's JsonPrinter emits boolean success, while some versions use
			// process exit codes. Do not mistake a failed action for report output.
			var success bool
			var code int
			failed := false
			if json.Unmarshal(action.RC, &success) == nil {
				failed = !success
			} else if json.Unmarshal(action.RC, &code) == nil {
				failed = code != 0
			} else {
				result.Errors = append(result.Errors, "Invalid health action result")
				continue
			}
			if failed && action.Error == "" {
				action.Error = action.Shortdesc + ": wizard action failed"
				if action.Output != "" {
					action.Error += "\n" + action.Output
				}
			}
		}
		if action.Error != "" {
			result.Errors = append(result.Errors, action.Error)
			continue
		}
		action.Text = strings.ReplaceAll(action.Text, "\t", "    ")
		action.Number = len(result.Actions) + 1
		result.Actions = append(result.Actions, action)
		// Ruby presents the last output in the report dialog.
		if operation == "run" {
			result.Output = action.Output
		}
	}
	if err := scanner.Err(); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}
	if operation == "run" && len(result.Actions) == 0 && len(result.Errors) == 0 {
		result.Errors = append(result.Errors, "Wizard returned no execution results")
	}
	return result
}

func HealthWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveHealthWizard(w, r, runHealthCommand)
}
func serveHealthWizard(w http.ResponseWriter, r *http.Request, run healthCommand) {
	if r.URL.Path == "/cib/live/wizards/health/submit" || r.URL.Path == "/cib/live/wizards/health/submit/" {
		submitHealthWizard(w, r, run)
		return
	}
	if r.URL.Path != "/cib/live/wizards/health" && r.URL.Path != "/cib/live/wizards/health/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost && r.Method != http.MethodPatch {
		w.Header().Set("Allow", "GET, HEAD, POST, PATCH")
		http.Error(w, "Method not allowed", 405)
		return
	}
	out, err := run(r.Context(), "show")
	if err != nil {
		http.Error(w, "Unable to load health wizard: "+err.Error(), 500)
		return
	}
	var definition struct {
		Shortdesc string            `json:"shortdesc"`
		Longdesc  string            `json:"longdesc"`
		Steps     []json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &definition); err != nil || len(definition.Steps) > 0 {
		http.Error(w, "Unable to load health wizard definition", 500)
		return
	}
	data := map[string]any{"Title": definition.Shortdesc, "Description": strings.Split(strings.ReplaceAll(definition.Longdesc, "\r\n", "\n"), "\n\n")}
	if r.Method == http.MethodPost || r.Method == http.MethodPatch {
		result := healthCommandResult(r.Context(), "verify", run)
		data["VerifyMode"] = true
		data["Actions"] = result.Actions
		data["Errors"] = result.Errors
	}
	renderTemplate(w, "wizard_health", data)
}

func submitHealthWizard(w http.ResponseWriter, r *http.Request, run healthCommand) {
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
	// Health has no fields: never accept arbitrary script parameters from clients.
	var params map[string]json.RawMessage
	if err := json.Unmarshal([]byte(r.FormValue("pa")), &params); err != nil || params == nil || len(params) != 0 {
		http.Error(w, "Invalid health wizard parameters", 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	result := healthCommandResult(r.Context(), "verify", run)
	if len(result.Errors) == 0 {
		expectedActions := len(result.Actions)
		result = healthCommandResult(r.Context(), "run", run)
		if len(result.Errors) == 0 && len(result.Actions) != expectedActions {
			result.Errors = append(result.Errors, fmt.Sprintf("Health check returned %d of %d expected action results", len(result.Actions), expectedActions))
		}
	}
	log.Printf("Health wizard Apply: completed_actions=%d errors=%d", len(result.Actions), len(result.Errors))
	if len(result.Errors) > 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(result.Errors)
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}
