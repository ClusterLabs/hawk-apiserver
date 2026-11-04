package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type tagCreate struct {
	ID   string   `json:"id"`
	Refs []string `json:"refs"`
}

// TagHelper uses resources_by_id (including nested resources) and constraints.
func tagObjectsFromXML(data []byte) ([]string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	objects := []string{}
	path := []string{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch node := token.(type) {
		case xml.StartElement:
			path = append(path, node.Name.Local)
			resource := len(path) >= 4 && path[0] == "cib" && path[1] == "configuration" && path[2] == "resources"
			constraint := len(path) == 4 && path[0] == "cib" && path[1] == "configuration" && path[2] == "constraints"
			if resource {
				switch node.Name.Local {
				case "primitive", "group", "clone", "master", "bundle":
				default:
					resource = false
				}
			}
			if resource || constraint {
				for _, attr := range node.Attr {
					if attr.Name.Local == "id" {
						objects = append(objects, attr.Value)
						break
					}
				}
			}
		case xml.EndElement:
			path = path[:len(path)-1]
		}
	}
	sortResourceIDs(objects)
	return objects, nil
}

func TagNewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/tags/new" && r.URL.Path != "/cib/live/tags/new/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	renderTemplate(w, "tag_new", map[string]any{"Title": "Create Tag"})
}

func FetchTagObjects(w http.ResponseWriter, r *http.Request) {
	if !groupFetchMethod(w, r) {
		return
	}
	cib, _, err := GetCIB()
	if err != nil {
		http.Error(w, "Unable to read objects: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cib.TagObjects)
}

var tagRefID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func tagCreateArgs(tag tagCreate) ([]string, error) {
	if tag.ID == "" {
		return nil, fmt.Errorf("ID is required")
	}
	if !groupID.MatchString(tag.ID) {
		return nil, fmt.Errorf("Invalid ID")
	}
	if len(tag.Refs) == 0 {
		return nil, fmt.Errorf("No Tag resources specified")
	}
	for _, ref := range tag.Refs {
		if !tagRefID.MatchString(ref) {
			return nil, fmt.Errorf("Invalid ID")
		}
	}
	return append([]string{"configure", "tag", tag.ID}, tag.Refs...), nil
}

func TagCreateHandler(w http.ResponseWriter, r *http.Request) {
	createTag(w, r, GetCIB, runGroupCommand)
}

func createTag(w http.ResponseWriter, r *http.Request, read func() (CIB, int, error), run func(context.Context, ...string) error) {
	if !groupFetchMethod(w, r) {
		return
	}
	var tag tagCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&tag); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	args, err := tagCreateArgs(tag)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	cib, _, err := read()
	if err != nil {
		http.Error(w, "Unable to read objects: "+err.Error(), 500)
		return
	}
	available := map[string]bool{}
	for _, id := range cib.TagObjects {
		available[id] = true
	}
	for _, ref := range tag.Refs {
		if !available[ref] {
			http.Error(w, "Object is no longer available: "+ref, 409)
			return
		}
	}
	if err := run(r.Context(), args...); err != nil {
		http.Error(w, tagCreateError(tag.ID, err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "redirect": "/cib/live/tags/" + url.PathEscape(tag.ID) + "/edit?flash=created"})
}

// Present crmsh failures as form errors, without terminal formatting or process status.
func tagCreateError(id string, err error) string {
	message := strings.TrimSpace(stripANSI(err.Error()))
	if strings.Contains(message, "Found existing") && strings.Contains(message, "same ID") {
		return fmt.Sprintf("The ID %q is already in use", id)
	}
	message = regexp.MustCompile(`: exit status [0-9]+$`).ReplaceAllString(message, "")
	message = strings.TrimSpace(strings.TrimPrefix(message, "ERROR:"))
	return message
}
