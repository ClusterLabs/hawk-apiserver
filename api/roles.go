package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
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
type aclTarget struct {
	ID         string   `xml:"id,attr" json:"id"`
	Roles      []string `xml:"-" json:"roles"`
	References []struct {
		XMLName xml.Name
		ID      string `xml:"id,attr"`
	} `xml:",any" json:"-"`
}

type aclRolesState struct {
	IDs       map[string]bool `json:"-"`
	Targets   []aclTarget     `json:"targets"`
	Roles     []aclRole       `json:"roles"`
	Enabled   bool            `json:"enabled"`
	Supported bool            `json:"supported"`
	Legacy    bool            `json:"legacy"`
}

func loadACLRoles(ctx context.Context, run clusterConfigCommand) (aclRolesState, error) {
	state := aclRolesState{Roles: []aclRole{}, Targets: []aclTarget{}}
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
		Schema        string `xml:"validate-with,attr"`
		Configuration struct {
			Targets    []aclTarget `xml:"acls>acl_target"`
			Users      []aclTarget `xml:"acls>acl_user"`
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
	for _, target := range append(cib.Configuration.Targets, cib.Configuration.Users...) {
		target.Roles = []string{}
		for _, ref := range target.References {
			if ref.XMLName.Local == "role" || ref.XMLName.Local == "role_ref" {
				target.Roles = append(target.Roles, ref.ID)
			}
		}
		target.References = nil
		state.Targets = append(state.Targets, target)
	}
	sort.Slice(state.Targets, func(i, j int) bool { return state.Targets[i].ID < state.Targets[j].ID })
	state.IDs = make(map[string]bool)
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	depth, configurationDepth := 0, 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return state, err
		}
		switch element := token.(type) {
		case xml.StartElement:
			depth++
			if element.Name.Local == "configuration" {
				configurationDepth = depth
			}
			if configurationDepth > 0 && depth > configurationDepth {
				for _, attr := range element.Attr {
					if attr.Name.Local == "id" {
						state.IDs[attr.Value] = true
					}
				}
			}
		case xml.EndElement:
			if depth == configurationDepth {
				configurationDepth = 0
			}
			depth--
		}
	}
	state.Legacy = strings.HasPrefix(cib.Schema, "pacemaker-1.")
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
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 5 && parts[0] == "cib" && parts[1] == "live" && parts[2] == "roles" && parts[4] == "edit" {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", 405)
			return
		}
		renderTemplate(w, "role_edit", map[string]any{"Title": "Edit Role", "RoleID": parts[3]})
		return
	}

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
	deleteACLRecord(w, r, run, false)
}
func deleteACLRecord(w http.ResponseWriter, r *http.Request, run clusterConfigCommand, target bool) {
	label, missing, success := "role", "The role does not exist", "Role deleted successfully"
	if target {
		label, missing, success = "ACL target", "The ACL target does not exist", "ACL target deleted successfully"
	}
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
		http.Error(w, "Invalid "+label+" ID", 400)
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
	if target {
		exists = false
		for _, item := range state.Targets {
			if item.ID == request.ID {
				exists = true
				break
			}
		}
	}
	if !exists {
		w.WriteHeader(404)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": missing})
		return
	}
	if _, err := run(r.Context(), "crm", []string{"--force", "configure", "delete", request.ID}, ""); err != nil {
		w.WriteHeader(422)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Error deleting %s: %s", request.ID, err)})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "message": success})
}
