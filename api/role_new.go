package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func RoleCreateHandler(w http.ResponseWriter, r *http.Request) {
	createACLRole(w, r, runClusterConfigCommand)
}
func createACLRole(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
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
	var role aclRole
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&role); err != nil {
		fail(400, fmt.Errorf("Invalid role"))
		return
	}
	command, err := roleUpdateCommand(role)
	if err != nil {
		fail(422, err)
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
	if state.IDs[role.ID] {
		fail(422, fmt.Errorf("The ID %q is already in use", role.ID))
		return
	}
	// Match Record#create: crm configure with the role statement on stdin.
	if _, err := run(r.Context(), "crm", []string{"configure"}, command); err != nil {
		fail(422, fmt.Errorf("Unable to create: %s", err))
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Role created successfully"})
}
