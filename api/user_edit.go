package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func UserEditHandler(w http.ResponseWriter, r *http.Request) {
	serveUserEdit(w, r, runClusterConfigCommand)
}
func serveUserEdit(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	w.Header().Set("Content-Type", "application/json")
	fail := func(status int, err error) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, HEAD, POST")
		fail(405, fmt.Errorf("Method not allowed"))
		return
	}
	id := r.URL.Query().Get("id")
	var request aclTarget
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			fail(400, fmt.Errorf("Invalid ACL target"))
			return
		}
		id = request.ID
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
	var target *aclTarget
	for i := range state.Targets {
		if state.Targets[i].ID == id {
			target = &state.Targets[i]
			break
		}
	}
	if target == nil {
		fail(404, fmt.Errorf("The ACL target does not exist"))
		return
	}
	if r.Method == http.MethodPost {
		command, err := aclTargetCommand(request, state)
		if err != nil {
			fail(422, err)
			return
		}
		if _, err := run(r.Context(), "crm", []string{"-F", "configure", "load", "update", "/dev/stdin"}, command); err != nil {
			fail(422, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "ACL target updated successfully"})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"target": target, "roles": state.Roles, "enabled": state.Enabled, "legacy": state.Legacy})
}
