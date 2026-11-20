package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

// Keep primitive subtrees verbatim when only group metadata changes.
type groupXMLPart struct {
	XMLName    xml.Name
	Attributes []xml.Attr `xml:",any,attr"`
	Inner      string     `xml:",innerxml"`
}
type groupXMLElement struct {
	XMLName    xml.Name
	Attributes []xml.Attr     `xml:",any,attr"`
	Parts      []groupXMLPart `xml:",any"`
}

func groupXMLAttr(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}
func setGroupXMLAttr(attrs []xml.Attr, name, value string) []xml.Attr {
	for i := range attrs {
		if attrs[i].Name.Local == name {
			attrs[i].Value = value
			return attrs
		}
	}
	return append(attrs, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

type groupEditState struct {
	Group     groupCreate     `json:"group"`
	Available []string        `json:"available"`
	XML       groupXMLElement `json:"-"`
}

func readGroupEdit(ctx context.Context, id string, run clusterConfigCommand) (groupEditState, bool, error) {
	state := groupEditState{Group: groupCreate{ID: id, Children: []string{}, Meta: map[string]string{}}, Available: []string{}}
	data, err := run(ctx, "cibadmin", []string{"-Ql"}, "")
	if err != nil {
		return state, false, err
	}
	var doc struct {
		Configuration struct {
			Resources struct {
				Inner string `xml:",innerxml"`
			} `xml:"resources"`
		} `xml:"configuration"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		return state, false, err
	}
	decoder := xml.NewDecoder(strings.NewReader("<resources>" + doc.Configuration.Resources.Inner + "</resources>"))
	free := []string{}
	depth := 0
	found := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return state, false, err
		}
		switch node := token.(type) {
		case xml.StartElement:
			depth++
			if depth == 2 && node.Name.Local == "primitive" {
				free = append(free, groupXMLAttr(node.Attr, "id"))
			}
			if node.Name.Local == "group" && groupXMLAttr(node.Attr, "id") == id {
				if err := decoder.DecodeElement(&state.XML, &node); err != nil {
					return state, false, err
				}
				depth--
				found = true
			}
		case xml.EndElement:
			depth--
		}
	}
	if !found {
		return state, false, nil
	}
	metaRead := false
	for _, part := range state.XML.Parts {
		if part.XMLName.Local == "primitive" {
			state.Group.Children = append(state.Group.Children, groupXMLAttr(part.Attributes, "id"))
		}
		if part.XMLName.Local == "meta_attributes" && !metaRead {
			var set struct {
				Pairs []Nvpair `xml:"nvpair"`
			}
			if err := xml.Unmarshal([]byte("<meta_attributes>"+part.Inner+"</meta_attributes>"), &set); err != nil {
				return state, false, err
			}
			for _, pair := range set.Pairs {
				state.Group.Meta[pair.Name] = pair.Value
			}
			metaRead = true
		}
	}
	state.Available = append(state.Available, state.Group.Children...)
	sortResourceIDs(free)
	seen := map[string]bool{}
	for _, child := range state.Available {
		seen[child] = true
	}
	for _, child := range free {
		if !seen[child] {
			state.Available = append(state.Available, child)
			seen[child] = true
		}
	}
	return state, true, nil
}
func GroupEditHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "cib" || parts[1] != "live" || parts[2] != "groups" || parts[4] != "edit" {
		resourceFallback(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	renderTemplate(w, "group_new", map[string]any{"Title": "Edit Group", "Editing": true, "GroupID": parts[3]})
}
func GroupFetchHandler(w http.ResponseWriter, r *http.Request) {
	fetchGroupEdit(w, r, runClusterConfigCommand)
}
func fetchGroupEdit(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	state, found, err := readGroupEdit(r.Context(), r.URL.Query().Get("id"), run)
	if err != nil {
		http.Error(w, "Unable to read group: "+err.Error(), 500)
		return
	}
	if !found {
		http.Error(w, "The group does not exist", 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}
func knownGroupMeta(name string) bool {
	for _, option := range groupMetaOptions() {
		if option.Name == name {
			return true
		}
	}
	return false
}
func mergedGroupMeta(request groupCreate, state groupEditState) map[string]string {
	values := map[string]string{}
	for name, value := range state.Group.Meta {
		if !knownGroupMeta(name) {
			values[name] = value
		}
	}
	for name, value := range request.Meta {
		values[name] = value
	}
	return values
}
func replaceGroupMeta(group groupXMLElement, values map[string]string) ([]byte, error) {
	index := -1
	for i, part := range group.Parts {
		if part.XMLName.Local == "meta_attributes" {
			index = i
			break
		}
	}
	if index < 0 && len(values) > 0 {
		group.Parts = append(group.Parts, groupXMLPart{XMLName: xml.Name{Local: "meta_attributes"}, Attributes: []xml.Attr{{Name: xml.Name{Local: "id"}, Value: groupXMLAttr(group.Attributes, "id") + "-meta_attributes"}}})
		index = len(group.Parts) - 1
	}
	if index >= 0 {
		part := group.Parts[index]
		var content struct {
			Parts []groupXMLPart `xml:",any"`
		}
		if err := xml.Unmarshal([]byte("<meta_attributes>"+part.Inner+"</meta_attributes>"), &content); err != nil {
			return nil, err
		}
		result := []groupXMLPart{}
		seen := map[string]bool{}
		for _, pair := range content.Parts {
			if pair.XMLName.Local != "nvpair" {
				result = append(result, pair)
				continue
			}
			name := groupXMLAttr(pair.Attributes, "name")
			value, ok := values[name]
			if !ok {
				continue
			}
			pair.Attributes = setGroupXMLAttr(pair.Attributes, "value", value)
			result = append(result, pair)
			seen[name] = true
		}
		names := []string{}
		for name := range values {
			if !seen[name] {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			result = append(result, groupXMLPart{XMLName: xml.Name{Local: "nvpair"}, Attributes: []xml.Attr{{Name: xml.Name{Local: "id"}, Value: groupXMLAttr(part.Attributes, "id") + "-" + name}, {Name: xml.Name{Local: "name"}, Value: name}, {Name: xml.Name{Local: "value"}, Value: values[name]}}})
		}
		if len(result) == 0 {
			group.Parts = append(group.Parts[:index], group.Parts[index+1:]...)
		} else {
			var inner strings.Builder
			for _, pair := range result {
				data, err := xml.Marshal(pair)
				if err != nil {
					return nil, err
				}
				inner.Write(data)
			}
			group.Parts[index].Inner = inner.String()
		}
	}
	return xml.Marshal(group)
}
func persistGroupEdit(ctx context.Context, request groupCreate, state groupEditState, run clusterConfigCommand) error {
	meta := mergedGroupMeta(request, state)
	if reflect.DeepEqual(request.Children, state.Group.Children) {
		data, err := replaceGroupMeta(state.XML, meta)
		if err != nil {
			return err
		}
		_, err = run(ctx, "cibadmin", []string{"-p", "--replace"}, string(data))
		return err
	}
	tokens := append([]string{"group", request.ID}, request.Children...)
	if len(meta) > 0 {
		tokens = append(tokens, "meta")
		names := []string{}
		for name := range meta {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			tokens = append(tokens, name+"="+meta[name])
		}
	}
	for i, token := range tokens {
		if strings.ContainsAny(token, "\r\n\x00") {
			return fmt.Errorf("Group fields must not contain line breaks")
		}
		tokens[i] = "'" + strings.ReplaceAll(token, "'", "'\"'\"'") + "'"
	}
	_, err := run(ctx, "crm", []string{"-F", "configure", "load", "update", "-"}, strings.Join(tokens, " ")+"\n")
	return err
}
func GroupUpdateHandler(w http.ResponseWriter, r *http.Request) {
	updateGroupEdit(w, r, runClusterConfigCommand)
}
func updateGroupEdit(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	var request groupCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	if err := validateGroup(request); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	state, found, err := readGroupEdit(r.Context(), request.ID, run)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if !found {
		http.Error(w, "The group does not exist", 404)
		return
	}
	available := map[string]bool{}
	for _, id := range state.Available {
		available[id] = true
	}
	for _, id := range request.Children {
		if !available[id] {
			http.Error(w, "Child is no longer available: "+id, 409)
			return
		}
	}
	if err := persistGroupEdit(r.Context(), request, state, run); err != nil {
		http.Error(w, tagCreateError(request.ID, err), 500)
		return
	}
	action := map[string]string{"Started": "start", "Stopped": "stop", "Master": "promote"}[request.Meta["target-role"]]
	if action != "" {
		if _, err := run(r.Context(), "crm", []string{"-F", "resource", action, request.ID}, ""); err != nil {
			http.Error(w, "Group updated, but setting its target role failed: "+err.Error(), 500)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Group updated successfully"})
}

// Match PrimitivesController#create with a parent: create the primitive, then
// append it to the group's current members. Do not apply group role actions here.
func createPrimitiveInGroup(w http.ResponseWriter, r *http.Request, primitive Primitive, run clusterConfigCommand, cibBase ...string) {
	base := "/cib/live"
	if len(cibBase) > 0 {
		base = cibBase[0]
	}
	if !groupID.MatchString(primitive.Parent) {
		http.Error(w, "Invalid parent group ID", 400)
		return
	}
	_, found, err := readGroupEdit(r.Context(), primitive.Parent, run)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if !found {
		http.Error(w, "The parent group does not exist", 404)
		return
	}
	if _, err := run(r.Context(), "crm", primitiveCreateArgs(primitive), ""); err != nil {
		http.Error(w, tagCreateError(primitive.ID, err), 500)
		return
	}
	state, found, err := readGroupEdit(r.Context(), primitive.Parent, run)
	if err != nil || !found {
		http.Error(w, "Primitive created, but the parent group could not be reloaded", 500)
		return
	}
	request := state.Group
	request.Children = append(append([]string{}, state.Group.Children...), primitive.ID)
	if err := persistGroupEdit(r.Context(), request, state, run); err != nil {
		http.Error(w, "Primitive created, but adding it to the group failed: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "redirect": base + "/groups/" + primitive.Parent + "/edit?flash=child-created"})
}
