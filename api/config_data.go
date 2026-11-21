package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var configCollections = []string{"resources", "constraints", "nodes", "tags", "alerts", "fencing"}

// Read the configuration tree directly: the status model does not include all
// resource types (templates, bundles, and nested group/clone members).
type configXMLNode struct {
	XMLName    xml.Name
	Attributes []xml.Attr      `xml:",any,attr"`
	Children   []configXMLNode `xml:",any"`
}

func (n configXMLNode) attr(name string) string { return groupXMLAttr(n.Attributes, name) }
func (n configXMLNode) child(name string) configXMLNode {
	for _, child := range n.Children {
		if child.XMLName.Local == name {
			return child
		}
	}
	return configXMLNode{}
}
func configResource(name string) bool {
	switch name {
	case "primitive", "template", "group", "clone", "master", "bundle":
		return true
	}
	return false
}
func configObjects(configuration configXMLNode, kind string) []configXMLNode {
	objects := []configXMLNode{}
	var visit func(configXMLNode)
	visit = func(node configXMLNode) {
		if !configResource(node.XMLName.Local) {
			return
		}
		objects = append(objects, node)
		// Ruby lists group and clone members as separate rows, but not bundle internals.
		if node.XMLName.Local == "group" || node.XMLName.Local == "clone" || node.XMLName.Local == "master" {
			for _, child := range node.Children {
				visit(child)
			}
		}
	}
	for _, node := range configuration.child(kind).Children {
		if kind == "resources" {
			visit(node)
		} else {
			objects = append(objects, node)
		}
	}
	return objects
}
func configRefs(node configXMLNode, element string) []string {
	refs := []string{}
	for _, child := range node.Children {
		if child.XMLName.Local == element {
			refs = append(refs, child.attr("id"))
		}
	}
	return refs
}
func configPairs(node configXMLNode, section string) map[string]string {
	pairs := map[string]string{}
	for _, set := range node.Children {
		if set.XMLName.Local == section {
			for _, pair := range set.Children {
				if pair.XMLName.Local == "nvpair" {
					pairs[pair.attr("name")] = pair.attr("value")
				}
			}
		}
	}
	return pairs
}
func configAlert(node configXMLNode) map[string]any {
	return map[string]any{"id": node.attr("id"), "value": node.attr("value"), "path": node.attr("path"), "meta": configPairs(node, "meta_attributes"), "attributes": configPairs(node, "instance_attributes")}
}
func readConfigTree(ctx context.Context, run clusterConfigCommand) (configXMLNode, []byte, error) {
	raw, err := run(ctx, "cibadmin", []string{"-Ql"}, "")
	if err != nil {
		return configXMLNode{}, nil, err
	}
	var document configXMLNode
	if err := xml.Unmarshal(raw, &document); err != nil {
		return document, nil, err
	}
	if document.XMLName.Local != "cib" || document.child("configuration").XMLName.Local != "configuration" {
		return document, nil, fmt.Errorf("Invalid CIB configuration")
	}
	return document, raw, nil
}
func configRows(ctx context.Context, kind string, includeRemote bool, run clusterConfigCommand) (any, error) {
	document, raw, err := readConfigTree(ctx, run)
	if err != nil {
		return nil, err
	}
	if kind == "fencing" {
		state, err := fencingEditFromXML(raw)
		return state.Fencing, err
	}
	rows := []map[string]any{}
	for _, node := range configObjects(document.child("configuration"), kind) {
		row := map[string]any{"id": node.attr("id")}
		switch kind {
		case "resources":
			objectType := node.XMLName.Local
			row["object_type"], row["sort_type"], row["editable"] = objectType, objectType, objectType != "bundle"
			row["clazz"], row["provider"], row["type"], row["template"] = node.attr("class"), node.attr("provider"), node.attr("type"), node.attr("template")
			if objectType == "primitive" {
				agent := []string{}
				for _, attr := range []string{"class", "provider", "type"} {
					if value := node.attr(attr); value != "" {
						agent = append(agent, value)
					}
				}
				row["sort_type"] = strings.Join(agent, ":")
			}
			children := []string{}
			for _, child := range node.Children {
				if configResource(child.XMLName.Local) {
					children = append(children, child.attr("id"))
				}
			}
			row["children"] = children
			if (objectType == "clone" || objectType == "master") && len(children) > 0 {
				row["child"] = children[0]
			}
		case "constraints":
			row["object_type"] = strings.TrimPrefix(node.XMLName.Local, "rsc_")
			resources := []string{}
			for _, attr := range []string{"rsc", "with-rsc", "first", "then"} {
				if value := node.attr(attr); value != "" {
					resources = append(resources, value)
				}
			}
			for _, set := range node.Children {
				if set.XMLName.Local == "resource_set" {
					resources = append(resources, configRefs(set, "resource_ref")...)
				}
			}
			row["resources"], row["children"] = resources, resources
			for _, attr := range []string{"score", "node", "resource-discovery", "ticket"} {
				if value := node.attr(attr); value != "" {
					row[strings.ReplaceAll(attr, "-", "_")] = value
				}
			}
		case "nodes":
			if !includeRemote && node.attr("type") == "remote" {
				continue
			}
			row["name"], row["remote"] = node.attr("uname"), node.attr("type") == "remote"
		case "tags":
			row["object_type"], row["refs"] = "tag", configRefs(node, "obj_ref")
		case "alerts":
			row = configAlert(node)
			recipients := []map[string]any{}
			for _, recipient := range node.Children {
				if recipient.XMLName.Local == "recipient" {
					recipients = append(recipients, configAlert(recipient))
				}
			}
			row["recipients"] = recipients
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// ConfigListHandler serves the same JSON URLs as Rails. Shadow requests go
// through the existing batch lock and session checks, never through live reads.
func ConfigListHandler(w http.ResponseWriter, r *http.Request) {
	configListHandler(w, r, runClusterConfigCommand, batch)
}
func configListHandler(w http.ResponseWriter, r *http.Request, run clusterConfigCommand, backend *batchBackend) {
	for _, kind := range configCollections {
		if r.URL.Path == batchBase+"/"+kind+".json" {
			request := r.Clone(r.Context())
			request.URL.Path = batchAPI + "/config/list/" + kind
			backend.api(w, request)
			return
		}
		if r.URL.Path == "/cib/live/"+kind+".json" {
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", "GET")
				batchError(w, 405, fmt.Errorf("Method not allowed"))
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
			defer cancel()
			serveConfigRows(w, r, ctx, kind, run)
			return
		}
	}
	http.NotFound(w, r)
}
func serveConfigRows(w http.ResponseWriter, r *http.Request, ctx context.Context, kind string, run clusterConfigCommand) {
	w.Header().Set("Cache-Control", "no-store")
	rows, err := configRows(ctx, kind, r.URL.Query().Get("remote") != "false", run)
	if err != nil {
		batchError(w, 500, err)
		return
	}
	batchJSON(w, rows)
}

// Shared mutations validate the selected collection before invoking crmsh;
// this also covers groups, clones, and templates without treating them as primitives.
func ConfigActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/cib/config/rename" && r.URL.Path != "/api/cib/config/delete" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		batchError(w, 405, fmt.Errorf("Method not allowed"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	configAction(w, r, ctx, strings.HasSuffix(r.URL.Path, "/rename"), runClusterConfigCommand)
}
func configAction(w http.ResponseWriter, r *http.Request, ctx context.Context, rename bool, run clusterConfigCommand) {
	kind := r.URL.Query().Get("kind")
	switch kind {
	case "resources", "constraints", "tags", "alerts":
	default:
		batchError(w, 400, fmt.Errorf("Invalid configuration collection"))
		return
	}
	var input struct {
		ID    string `json:"id"`
		OldID string `json:"oldID"`
		NewID string `json:"newID"`
	}
	if !readBatchJSON(w, r, &input) {
		return
	}
	id := input.ID
	if rename {
		id = input.OldID
	}
	if !groupID.MatchString(id) || (rename && (!groupID.MatchString(input.NewID) || input.NewID == id)) {
		batchError(w, 400, fmt.Errorf("Invalid configuration ID"))
		return
	}
	document, _, err := readConfigTree(ctx, run)
	if err != nil {
		batchError(w, 500, err)
		return
	}
	found := false
	for _, node := range configObjects(document.child("configuration"), kind) {
		if node.attr("id") == id {
			found = true
			break
		}
	}
	if !found {
		batchError(w, 404, fmt.Errorf("Configuration object %s does not exist in %s", id, kind))
		return
	}
	args := []string{"--force", "configure", "delete", id}
	message := "Successfully deleted " + id
	if rename {
		var containsID func(configXMLNode) bool
		containsID = func(node configXMLNode) bool {
			if node.attr("id") == input.NewID {
				return true
			}
			for _, child := range node.Children {
				if containsID(child) {
					return true
				}
			}
			return false
		}
		if containsID(document.child("configuration")) {
			batchError(w, 409, fmt.Errorf("ID %s already exists", input.NewID))
			return
		}
		args = []string{"-D", "plain", "configure", "rename", id, input.NewID}
		message = "Successfully renamed " + id + " to " + input.NewID
	}
	if _, err := run(ctx, "crm", args, ""); err != nil {
		batchError(w, 500, err)
		return
	}
	batchJSON(w, map[string]any{"success": true, "message": message})
}
