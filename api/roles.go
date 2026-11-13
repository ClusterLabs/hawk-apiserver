package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type aclRoleRule struct {
	Right     string `json:"right"`
	Tag       string `json:"tag,omitempty"`
	Ref       string `json:"ref,omitempty"`
	XPath     string `json:"xpath,omitempty"`
	Attribute string `json:"attribute,omitempty"`
}
type aclRole struct {
	ID    string        `json:"id"`
	Rules []aclRoleRule `json:"rules"`
}
type aclRolesState struct {
	Roles     []aclRole `json:"roles"`
	Enabled   bool      `json:"enabled"`
	Supported bool      `json:"supported"`
}

func loadACLRoles(ctx context.Context, run clusterConfigCommand) (aclRolesState, error) {
	state := aclRolesState{Roles: []aclRole{}}
	features, err := run(ctx, "pacemakerd", []string{"-F"}, "")
	if err != nil {
		return state, err
	}
	for _, feature := range strings.Fields(string(features)) {
		if feature == "acls" {
			state.Supported = true
		}
	}
	// Since Pacemaker 2.1.0 ACL support is unconditional and the feature
	// flag is no longer reported. Match Ruby's version fallback.
	fields := strings.Fields(string(features))
	if !state.Supported && len(fields) > 1 {
		version := strings.Split(strings.Split(fields[1], "+")[0], ".")
		if len(version) >= 2 {
			major, majorErr := strconv.Atoi(version[0])
			minor, minorErr := strconv.Atoi(version[1])
			state.Supported = majorErr == nil && minorErr == nil && (major > 2 || major == 2 && minor >= 1)
		}
	}
	if !state.Supported {
		return state, nil
	}
	data, err := run(ctx, "cibadmin", []string{"-Ql"}, "")
	if err != nil {
		return state, err
	}
	var cib struct {
		Configuration struct {
			Properties []struct {
				Pairs []Nvpair `xml:"nvpair"`
			} `xml:"crm_config>cluster_property_set"`
			Roles []struct {
				ID    string `xml:"id,attr"`
				Rules []struct {
					XMLName   xml.Name
					Kind      string `xml:"kind,attr"`
					Tag       string `xml:"object-type,attr"`
					OldTag    string `xml:"tag,attr"`
					Ref       string `xml:"reference,attr"`
					OldRef    string `xml:"ref,attr"`
					XPath     string `xml:"xpath,attr"`
					Attribute string `xml:"attribute,attr"`
				} `xml:",any"`
			} `xml:"acls>acl_role"`
		} `xml:"configuration"`
	}
	if err := xml.Unmarshal(data, &cib); err != nil {
		return state, err
	}
	for _, set := range cib.Configuration.Properties {
		for _, p := range set.Pairs {
			if p.Name == "enable-acl" && p.Value == "true" {
				state.Enabled = true
			}
		}
	}
	for _, role := range cib.Configuration.Roles {
		item := aclRole{ID: role.ID, Rules: []aclRoleRule{}}
		for _, rule := range role.Rules {
			right, tag, ref := rule.Kind, rule.Tag, rule.Ref
			if right == "" {
				right = rule.XMLName.Local
			}
			if tag == "" {
				tag = rule.OldTag
			}
			if ref == "" {
				ref = rule.OldRef
			}
			item.Rules = append(item.Rules, aclRoleRule{Right: right, Tag: tag, Ref: ref, XPath: rule.XPath, Attribute: rule.Attribute})
		}
		state.Roles = append(state.Roles, item)
	}
	sort.Slice(state.Roles, func(i, j int) bool { return state.Roles[i].ID < state.Roles[j].ID })
	return state, nil
}
func RolesHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/roles" && r.URL.Path != "/cib/live/roles/" {
		if Routehandler != nil {
			Routehandler.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if r.Method == http.MethodPost && Routehandler != nil {
			Routehandler.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	renderTemplate(w, "roles", map[string]any{"Title": "Roles"})
}
func RolesListHandler(w http.ResponseWriter, r *http.Request) {
	serveRolesList(w, r, runClusterConfigCommand)
}
func serveRolesList(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	state, err := loadACLRoles(r.Context(), run)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Unable to load ACL roles: " + err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(state)
}
func RoleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	deleteACLRole(w, r, runClusterConfigCommand)
}
func deleteACLRole(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var request struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.ID == "" {
		http.Error(w, "Invalid role ID", 400)
		return
	}
	state, err := loadACLRoles(r.Context(), run)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if !state.Supported {
		w.WriteHeader(422)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "ACL is not supported by the cluster"})
		return
	}
	exists := false
	for _, role := range state.Roles {
		if role.ID == request.ID {
			exists = true
			break
		}
	}
	if !exists {
		w.WriteHeader(404)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "The role does not exist"})
		return
	}
	if _, err := run(r.Context(), "crm", []string{"--force", "configure", "delete", request.ID}, ""); err != nil {
		w.WriteHeader(422)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Error deleting %s: %s", request.ID, err)})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "message": "Role deleted successfully"})
}
