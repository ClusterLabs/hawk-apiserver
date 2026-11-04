package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Match Group.mapping and Record's ID validation in the Ruby implementation.
var groupID = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
var groupNumber = regexp.MustCompile(`\d+|\D+`)

type groupCreate struct {
	ID       string            `json:"id"`
	Children []string          `json:"children"`
	Meta     map[string]string `json:"meta"`
}

func groupMetaOptions() []SelectOption {
	return []SelectOption{
		{Name: "is-managed", Type: "boolean", DefaultValue: "true",
			Longdesc: "Is the cluster allowed to start and stop the resource?"},
		{Name: "maintenance", Type: "boolean", DefaultValue: "false",
			Longdesc: "Resources in maintenance mode are not monitored by the cluster."},
		{Name: "priority", Type: "integer", DefaultValue: "0",
			Longdesc: "If not all resources can be active, the cluster will stop lower priority resources " +
				"in order to keep higher priority ones active."},
		{Name: "target-role", Type: "enum", DefaultValue: "Stopped",
			PossibleValues: []string{"Started", "Stopped", "Master"},
			Longdesc:       "What state should the cluster attempt to keep this resource in?"},
	}
}

func GroupNewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/groups/new" && r.URL.Path != "/cib/live/groups/new/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	renderTemplate(w, "group_new", map[string]any{"Title": "Create Group"})
}

func groupFetchMethod(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodPost {
		return true
	}
	w.Header().Set("Allow", "POST")
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	return false
}

func FetchGroupMetaAttributes(w http.ResponseWriter, r *http.Request) {
	if !groupFetchMethod(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(SelectContent{Options: groupMetaOptions()})
}

// Like GroupHelper, offer only top-level primitives, not children of containers.
func groupChildren(cib CIB) []string {
	children := make([]string, 0, len(cib.Configuration.Primitives))
	for _, primitive := range cib.Configuration.Primitives {
		children = append(children, primitive.ID)
	}
	sort.SliceStable(children, func(i, j int) bool {
		a := groupNumber.FindAllString(strings.ToLower(children[i]), -1)
		b := groupNumber.FindAllString(strings.ToLower(children[j]), -1)
		for k := 0; k < len(a) && k < len(b); k++ {
			x, y := a[k], b[k]
			if x[0] >= '0' && x[0] <= '9' && y[0] >= '0' && y[0] <= '9' {
				x, y = strings.TrimLeft(x, "0"), strings.TrimLeft(y, "0")
				if len(x) != len(y) {
					return len(x) < len(y)
				}
			}
			if x != y {
				return x < y
			}
		}
		return len(a) < len(b)
	})
	return children
}

func FetchGroupChildren(w http.ResponseWriter, r *http.Request) {
	if !groupFetchMethod(w, r) {
		return
	}
	cib, _, err := GetCIB()
	if err != nil {
		http.Error(w, "Unable to read available children: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(groupChildren(cib))
}

func validateGroup(group groupCreate) error {
	if group.ID == "" {
		return fmt.Errorf("ID is required")
	}
	if !groupID.MatchString(group.ID) {
		return fmt.Errorf("Invalid ID")
	}
	if len(group.Children) == 0 {
		return fmt.Errorf("Group cannot be empty, expected at least one child")
	}
	seen := map[string]bool{}
	for _, child := range group.Children {
		if child == "" || seen[child] || child == group.ID {
			return fmt.Errorf("Invalid group child: %s", child)
		}
		seen[child] = true
	}
	for name, value := range group.Meta {
		switch name {
		case "is-managed", "maintenance":
			if value != "true" && value != "false" {
				return fmt.Errorf("Invalid value for %s", name)
			}
		case "priority":
			if _, err := strconv.ParseInt(value, 10, 64); err != nil {
				return fmt.Errorf("Invalid priority")
			}
		case "target-role":
			if value != "Started" && value != "Stopped" && value != "Master" {
				return fmt.Errorf("Invalid target-role")
			}
		default:
			return fmt.Errorf("Unknown group meta attribute: %s", name)
		}
	}
	return nil
}

func groupCreateArgs(group groupCreate) []string {
	args := append([]string{"configure", "group", group.ID}, group.Children...)
	if len(group.Meta) > 0 {
		args = append(args, "meta")
		// Stable ordering while preserving the selected child order.
		for _, option := range groupMetaOptions() {
			if value, ok := group.Meta[option.Name]; ok {
				args = append(args, option.Name+"="+value)
			}
		}
	}
	return args
}

func runGroupCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "/usr/sbin/crm", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}

func GroupCreateHandler(w http.ResponseWriter, r *http.Request) {
	createGroup(w, r, GetCIB, runGroupCommand)
}

func createGroup(w http.ResponseWriter, r *http.Request,
	readCIB func() (CIB, int, error), run func(context.Context, ...string) error) {
	if !groupFetchMethod(w, r) {
		return
	}
	var group groupCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&group); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := validateGroup(group); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cib, _, err := readCIB()
	if err != nil {
		http.Error(w, "Unable to read available children: "+err.Error(), http.StatusInternalServerError)
		return
	}
	available := map[string]bool{}
	for _, child := range groupChildren(cib) {
		available[child] = true
	}
	for _, child := range group.Children {
		if !available[child] {
			http.Error(w, "Child is no longer available: "+child, http.StatusConflict)
			return
		}
	}
	ctx := r.Context()
	if err := run(ctx, groupCreateArgs(group)...); err != nil {
		http.Error(w, "Unable to create: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// GroupsController#post_process_for! applies the requested role after saving.
	action := map[string]string{"Started": "start", "Stopped": "stop", "Master": "promote"}[group.Meta["target-role"]]
	if action != "" {
		if err := run(ctx, "-F", "resource", action, group.ID); err != nil {
			http.Error(w, "Group created, but setting its target role failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok", "message": "Group created successfully",
		"redirect": "/cib/live/groups/" + url.PathEscape(group.ID) + "/edit?flash=created",
	})
}
