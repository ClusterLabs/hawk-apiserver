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
	base := "/cib/live"
	switch r.URL.Path {
	case "/cib/live/colocations/new", "/cib/live/colocations/new/":
	case batchBase + "/colocations/new", batchBase + "/colocations/new/":
		base = batchBase
	default:
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	data := map[string]any{"Title": "Create Colocation", "CIBBase": base}
	if base == batchBase {
		data["BatchMode"] = true
		data["BatchConfig"] = map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}
		data["ColocationAPI"] = batchAPI + "/colocation"
	}
	renderTemplate(w, "colocation_new", data)
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
	saveColocation(w, r, c, read, run, "/cib/live")
}

func saveColocation(w http.ResponseWriter, r *http.Request, c colocationCreate, read func() (CIB, int, error), run func(context.Context, ...string) error, base string) {
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
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "redirect": base + "/colocations/" + url.PathEscape(c.ID) + "/edit?flash=created"})
}
func ColocationRubyHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "cib" || (parts[1] != "live" && parts[1] != batchID) || parts[2] != "colocations" || parts[4] != "edit" {
		resourceRubyHandler(w, r, "Constraint")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	base := "/cib/" + parts[1]
	data := map[string]any{"Title": "Edit Colocation", "EditMode": true, "ColocationID": parts[3], "CIBBase": base}
	if base == batchBase {
		data["BatchMode"] = true
		data["BatchConfig"] = map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}
		data["ColocationAPI"] = batchAPI + "/colocation"
	}
	renderTemplate(w, "colocation_new", data)
}

func colocationFromXML(data []byte) (colocationCreate, error) {
	var x struct {
		XMLName      xml.Name `xml:"rsc_colocation"`
		ID           string   `xml:"id,attr"`
		Score        string   `xml:"score,attr"`
		NodeAttr     string   `xml:"node-attribute,attr"`
		Resource     string   `xml:"rsc,attr"`
		WithResource string   `xml:"with-rsc,attr"`
		Role         string   `xml:"rsc-role,attr"`
		WithRole     string   `xml:"with-rsc-role,attr"`
		Sets         []struct {
			Sequential string `xml:"sequential,attr"`
			Role       string `xml:"role,attr"`
			Resources  []struct {
				ID string `xml:"id,attr"`
			} `xml:"resource_ref"`
		} `xml:"resource_set"`
	}
	if err := xml.Unmarshal(data, &x); err != nil {
		return colocationCreate{}, err
	}
	c := colocationCreate{ID: x.ID, Score: x.Score, NodeAttr: x.NodeAttr, Resources: []colocationSet{}}
	if x.Resource != "" {
		c.Resources = append(c.Resources, colocationSet{true, x.Role, []string{x.Resource}}, colocationSet{true, x.WithRole, []string{x.WithResource}})
	} else {
		for _, set := range x.Sets {
			resources := []string{}
			// Ruby displays resource sets in reverse XML order.
			for i := len(set.Resources) - 1; i >= 0; i-- {
				resources = append(resources, set.Resources[i].ID)
			}
			c.Resources = append(c.Resources, colocationSet{set.Sequential != "false", set.Role, resources})
		}
	}
	return c, nil
}
func readColocation(ctx context.Context, id string) (colocationCreate, error) {
	return readColocationUsing(ctx, id, runClusterConfigCommand)
}
func readColocationUsing(ctx context.Context, id string, run clusterConfigCommand) (colocationCreate, error) {
	if !groupID.MatchString(id) {
		return colocationCreate{}, fmt.Errorf("Invalid ID")
	}
	data, err := run(ctx, "cibadmin", []string{"-Q", "--xpath", "//constraints/rsc_colocation[@id='" + id + "']"}, "")
	if err != nil {
		return colocationCreate{}, fmt.Errorf("Unable to read colocation constraint: %w", err)
	}
	return colocationFromXML(data)
}
func FetchColocation(w http.ResponseWriter, r *http.Request) {
	fetchColocation(w, r, runClusterConfigCommand)
}
func fetchColocation(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if !groupFetchMethod(w, r) {
		return
	}
	c, err := readColocationUsing(r.Context(), r.URL.Query().Get("id"), run)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}
func runColocationUpdate(ctx context.Context, args ...string) error {
	return runColocationUpdateUsing(ctx, runClusterConfigCommand, args...)
}
func runColocationUpdateUsing(ctx context.Context, run clusterConfigCommand, args ...string) error {
	// Tokens are validated by colocationCreateArgs before reaching this runner.
	out, err := run(ctx, "crm", []string{"-F", "configure", "load", "update", "-"}, strings.Join(args[1:], " ")+"\n")
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}
func ColocationUpdateHandler(w http.ResponseWriter, r *http.Request) {
	updateColocation(w, r, readColocation, GetCIB, runColocationUpdate)
}
func updateColocation(w http.ResponseWriter, r *http.Request, find func(context.Context, string) (colocationCreate, error), read func() (CIB, int, error), run func(context.Context, ...string) error) {
	if !groupFetchMethod(w, r) {
		return
	}
	var c colocationCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&c); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	saveColocationUpdate(w, r, c, find, read, run, "/cib/live")
}
func saveColocationUpdate(w http.ResponseWriter, r *http.Request, c colocationCreate, find func(context.Context, string) (colocationCreate, error), read func() (CIB, int, error), run func(context.Context, ...string) error, base string) {
	args, err := colocationCreateArgs(c)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if _, err := find(r.Context(), c.ID); err != nil {
		http.Error(w, err.Error(), 409)
		return
	}
	cib, _, err := read()
	if err != nil {
		http.Error(w, err.Error(), 500)
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
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "redirect": base + "/colocations/" + url.PathEscape(c.ID) + "/edit?flash=updated"})
}

func colocationAction(ctx context.Context, id, newID string, run clusterConfigCommand) (int, error) {
	return constraintAction(ctx, id, newID, "colocation", run)
}
