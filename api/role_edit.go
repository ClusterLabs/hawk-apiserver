package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

func RoleEditHandler(w http.ResponseWriter, r *http.Request) {
	serveRoleEdit(w, r, runClusterConfigCommand)
}

func serveRoleEdit(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	w.Header().Set("Content-Type", "application/json")
	fail := func(status int, err error) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD, POST")
		fail(405, fmt.Errorf("Method not allowed"))
		return
	}
	state, err := loadACLRoles(r.Context(), run)
	if err != nil {
		fail(500, err)
		return
	}
	if !state.Supported {
		fail(422, fmt.Errorf("ACL is not supported by the cluster"))
		return
	}
	id := r.URL.Query().Get("id")
	var request aclRole
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			fail(400, fmt.Errorf("Invalid role"))
			return
		}
		id = request.ID
	}
	var role *aclRole
	for i := range state.Roles {
		if state.Roles[i].ID == id {
			role = &state.Roles[i]
			break
		}
	}
	if role == nil {
		fail(404, fmt.Errorf("The role does not exist"))
		return
	}
	if r.Method == http.MethodPost {
		command, err := roleUpdateCommand(request)
		if err != nil {
			fail(422, err)
			return
		}
		if _, err := run(r.Context(), "crm", []string{"-F", "configure", "load", "update", "/dev/stdin"}, command); err != nil {
			fail(422, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Role updated successfully"})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"role": role, "enabled": state.Enabled, "legacy": state.Legacy})
}

var aclRoleID = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.:-]*$`)

// Build crmsh input, never a shell command. Reject line breaks so a field cannot
// introduce another configure statement. crmsh accepts either quote style.
func roleUpdateCommand(role aclRole) (string, error) {
	if !aclRoleID.MatchString(role.ID) {
		return "", fmt.Errorf("Invalid role ID")
	}
	if len(role.Rules) == 0 {
		return "", fmt.Errorf("At least one rule is required")
	}
	parts := []string{"role", role.ID}
	for _, rule := range role.Rules {
		if rule.Right != "read" && rule.Right != "write" && rule.Right != "deny" {
			return "", fmt.Errorf("Right must be read, write, or deny")
		}
		if strings.TrimSpace(rule.XPath) == "" && strings.TrimSpace(rule.Tag) == "" {
			return "", fmt.Errorf("At least one of xpath or object type must be set.")
		}
		if strings.TrimSpace(rule.XPath) != "" && (strings.TrimSpace(rule.Tag) != "" || strings.TrimSpace(rule.Attribute) != "") {
			return "", fmt.Errorf("Conflicting options for rule")
		}
		parts = append(parts, rule.Right)
		for _, item := range []struct{ key, value string }{{"xpath", rule.XPath}, {"tag", rule.Tag}, {"ref", rule.Ref}, {"attr", rule.Attribute}} {
			if strings.TrimSpace(item.value) == "" {
				continue
			}
			if strings.ContainsAny(item.value, "\r\n\x00") {
				return "", fmt.Errorf("Rule fields must not contain line breaks")
			}
			quote := "'"
			if strings.Contains(item.value, quote) {
				quote = `"`
			}
			value := item.value
			if strings.Contains(value, quote) {
				// crmsh uses shlex: adjacent quoted segments preserve both
				// quote styles in XPath expressions without changing tokens.
				quote = "'"
				value = strings.ReplaceAll(value, "'", "'\"'\"'")
			}
			parts = append(parts, quote+item.key+":"+value+quote)
		}
	}
	return strings.Join(parts, " ") + "\n", nil
}
