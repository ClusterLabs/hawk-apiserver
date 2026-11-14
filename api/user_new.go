package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

func UserCreateHandler(w http.ResponseWriter, r *http.Request) {
	createACLTarget(w, r, runClusterConfigCommand)
}

func createACLTarget(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	w.Header().Set("Content-Type", "application/json")
	fail := func(status int, err error) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		fail(405, fmt.Errorf("Method not allowed"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var target aclTarget
	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		fail(400, fmt.Errorf("Invalid ACL target"))
		return
	}
	state, err := loadACLRoles(r.Context(), run)
	if err != nil {
		fail(500, err)
		return
	}
	if !state.Supported {
		fail(422, fmt.Errorf("ACL is not supported by this cluster."))
		return
	}
	command, err := aclTargetCreateCommand(target, state)
	if err != nil {
		fail(422, err)
		return
	}
	if _, err := run(r.Context(), "crm", []string{"configure"}, command); err != nil {
		fail(422, fmt.Errorf("Unable to create: %s", err))
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ACL target created successfully"})
}

var aclTargetID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func aclTargetCreateCommand(target aclTarget, state aclRolesState) (string, error) {
	if strings.TrimSpace(target.ID) == "" {
		return "", fmt.Errorf("ACL target ID is required")
	}
	if !aclTargetID.MatchString(target.ID) {
		return "", fmt.Errorf("Invalid ACL target ID")
	}
	if len(target.Roles) == 0 {
		return "", fmt.Errorf("At least one role is required")
	}
	if state.IDs[target.ID] {
		return "", fmt.Errorf("The ID %q is already in use", target.ID)
	}
	available := make(map[string]bool)
	for _, role := range state.Roles {
		available[role.ID] = true
	}
	for _, id := range target.Roles {
		if !available[id] || !aclRoleID.MatchString(id) {
			return "", fmt.Errorf("The role %q does not exist", id)
		}
	}
	command := "acl_target"
	if state.Legacy {
		command = "acl_user"
	}
	return command + " " + target.ID + " " + strings.Join(target.Roles, " ") + "\n", nil
}
