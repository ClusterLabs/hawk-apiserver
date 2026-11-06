package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type colocationSet struct {
	Sequential bool     `json:"sequential"`
	Action     string   `json:"action"`
	Resources  []string `json:"resources"`
}
type colocationCreate struct {
	ID        string          `json:"id"`
	Score     string          `json:"score"`
	NodeAttr  string          `json:"node_attr"`
	Resources []colocationSet `json:"resources"`
}

// ColocationHelper offers top-level resources and resource templates.
func colocationResourcesFromXML(data []byte) ([]string, error) {
	var document struct {
		Configuration struct {
			Resources struct {
				Items []struct {
					XMLName xml.Name
					ID      string `xml:"id,attr"`
				} `xml:",any"`
			} `xml:"resources"`
		} `xml:"configuration"`
	}
	if err := xml.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	ids := []string{}
	for _, item := range document.Configuration.Resources.Items {
		switch item.XMLName.Local {
		case "primitive", "group", "clone", "master", "bundle", "template":
			ids = append(ids, item.ID)
		}
	}
	sortResourceIDs(ids)
	return ids, nil
}

func ColocationNewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/colocations/new" && r.URL.Path != "/cib/live/colocations/new/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	renderTemplate(w, "colocation_new", map[string]any{"Title": "Create Colocation"})
}
func FetchColocationOptions(w http.ResponseWriter, r *http.Request) {
	if !groupFetchMethod(w, r) {
		return
	}
	cib, _, err := GetCIB()
	if err != nil {
		http.Error(w, "Unable to read cluster: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cib.ColocationResources)
}

func colocationCreateArgs(c colocationCreate) ([]string, error) {
	if c.ID == "" {
		return nil, fmt.Errorf("ID is required")
	}
	if !groupID.MatchString(c.ID) {
		return nil, fmt.Errorf("Invalid ID")
	}
	score := strings.TrimSpace(c.Score)
	if score == "" {
		return nil, fmt.Errorf("Score is required")
	}
	switch strings.ToLower(score) {
	case "mandatory", "advisory", "inf", "-inf", "infinity", "-infinity":
	default:
		if !locationScore.MatchString(score) {
			return nil, fmt.Errorf("Invalid score value")
		}
	}
	if strings.TrimSpace(c.NodeAttr) != "" && !tagRefID.MatchString(c.NodeAttr) {
		return nil, fmt.Errorf("Invalid node attribute")
	}
	if len(c.Resources) < 2 {
		return nil, fmt.Errorf("Constraint must consist of at least two separate resources")
	}
	// Match Colocation#shell_syntax: collapse adjacent sets before reversing
	// each set, except for the special two-resource form.
	collapsed := []colocationSet{}
	for _, set := range c.Resources {
		switch set.Action {
		case "", "Started", "Master", "Slave", "Stopped":
		default:
			return nil, fmt.Errorf("Invalid role")
		}
		if len(set.Resources) == 0 {
			return nil, fmt.Errorf("No resource specified")
		}
		for _, id := range set.Resources {
			if !groupID.MatchString(id) {
				return nil, fmt.Errorf("Invalid resource ID")
			}
		}
		last := len(collapsed) - 1
		if last >= 0 && collapsed[last].Sequential == set.Sequential && collapsed[last].Action == set.Action {
			collapsed[last].Resources = append(collapsed[last].Resources, set.Resources...)
		} else {
			set.Resources = append([]string(nil), set.Resources...)
			collapsed = append(collapsed, set)
		}
	}
	args := []string{"configure", "colocation", c.ID, score + ":"}
	simple := len(collapsed) == 1 && len(collapsed[0].Resources) == 2
	for _, set := range collapsed {
		resource := func(id string) string {
			if set.Action != "" {
				return id + ":" + set.Action
			}
			return id
		}
		if simple {
			for _, id := range set.Resources {
				args = append(args, resource(id))
			}
			continue
		}
		if !set.Sequential {
			args = append(args, "(")
		}
		for i := len(set.Resources) - 1; i >= 0; i-- {
			args = append(args, resource(set.Resources[i]))
		}
		if !set.Sequential {
			args = append(args, ")")
		}
	}
	if strings.TrimSpace(c.NodeAttr) != "" {
		args = append(args, "node-attribute="+c.NodeAttr)
	}
	return args, nil
}
func ColocationCreateHandler(w http.ResponseWriter, r *http.Request) {
	createColocation(w, r, GetCIB, runGroupCommand)
}
func createColocation(w http.ResponseWriter, r *http.Request, read func() (CIB, int, error), run func(context.Context, ...string) error) {
	if !groupFetchMethod(w, r) {
		return
	}
	var c colocationCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&c); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	args, err := colocationCreateArgs(c)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	cib, _, err := read()
	if err != nil {
		http.Error(w, "Unable to read resources: "+err.Error(), 500)
		return
	}
	available := map[string]bool{}
	for _, id := range cib.ColocationResources {
		available[id] = true
	}
	for _, set := range c.Resources {
		for _, id := range set.Resources {
			if !available[id] {
				http.Error(w, "Resource is no longer available: "+id, 409)
				return
			}
		}
	}
	if err := run(r.Context(), args...); err != nil {
		http.Error(w, tagCreateError(c.ID, err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "redirect": "/cib/live/colocations/" + url.PathEscape(c.ID) + "/edit?flash=created"})
}
func ColocationRubyHandler(w http.ResponseWriter, r *http.Request) {
	resourceRubyHandler(w, r, "Constraint")
}
