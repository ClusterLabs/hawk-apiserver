package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

type tagReference struct {
	ID string `xml:"id,attr"`
}
type tagElement struct {
	XMLName    xml.Name       `xml:"tag"`
	Attributes []xml.Attr     `xml:",any,attr"`
	Refs       []tagReference `xml:"obj_ref"`
}

func (tag tagElement) id() string {
	for _, attr := range tag.Attributes {
		if attr.Name.Local == "id" {
			return attr.Value
		}
	}
	return ""
}

func TagEditHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "cib" || parts[1] != "live" || parts[2] != "tags" || parts[4] != "edit" {
		TagRubyHandler(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	renderTemplate(w, "tag_new", map[string]any{"Title": "Edit Tag", "Editing": true, "TagID": parts[3]})
}

func loadTag(ctx context.Context, id string, run clusterConfigCommand) (tagElement, []string, error) {
	data, err := run(ctx, "cibadmin", []string{"-Ql"}, "")
	if err != nil {
		return tagElement{}, nil, err
	}
	var cib struct {
		Configuration struct {
			Tags []tagElement `xml:"tags>tag"`
		} `xml:"configuration"`
	}
	if err := xml.Unmarshal(data, &cib); err != nil {
		return tagElement{}, nil, err
	}
	objects, err := tagObjectsFromXML(data)
	if err != nil {
		return tagElement{}, nil, err
	}
	for _, tag := range cib.Configuration.Tags {
		if tag.id() == id {
			return tag, objects, nil
		}
	}
	return tagElement{}, objects, nil
}

func TagFetchHandler(w http.ResponseWriter, r *http.Request) { fetchTag(w, r, runClusterConfigCommand) }
func fetchTag(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	tag, objects, err := loadTag(r.Context(), r.URL.Query().Get("id"), run)
	if err != nil {
		http.Error(w, "Unable to read tag: "+err.Error(), 500)
		return
	}
	if tag.id() == "" {
		http.Error(w, "The tag does not exist", 404)
		return
	}
	refs := []string{}
	for _, ref := range tag.Refs {
		refs = append(refs, ref.ID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": tag.id(), "refs": refs, "objects": objects})
}

func TagUpdateHandler(w http.ResponseWriter, r *http.Request) {
	updateTag(w, r, runClusterConfigCommand)
}
func updateTag(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	var request tagCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	if _, err := tagCreateArgs(request); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	tag, objects, err := loadTag(r.Context(), request.ID, run)
	if err != nil {
		http.Error(w, "Unable to read tag: "+err.Error(), 500)
		return
	}
	if tag.id() == "" {
		http.Error(w, fmt.Sprintf("The ID %q does not exist", request.ID), 404)
		return
	}
	available := make(map[string]bool)
	for _, id := range objects {
		available[id] = true
	}
	tag.Refs = nil
	for _, id := range request.Refs {
		if !available[id] {
			http.Error(w, "Object is no longer available: "+id, 409)
			return
		}
		tag.Refs = append(tag.Refs, tagReference{ID: id})
	}
	data, err := xml.Marshal(tag)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if _, err := run(r.Context(), "cibadmin", []string{"-p", "--replace"}, string(data)); err != nil {
		http.Error(w, tagCreateError(request.ID, err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Tag updated successfully"})
}
