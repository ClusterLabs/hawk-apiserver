package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

type fencingLevel struct {
	Type    string   `json:"type"`
	Target  string   `json:"target"`
	Value   string   `json:"value"`
	Index   int      `json:"index"`
	Devices []string `json:"devices"`
}
type fencingXMLLevel struct {
	ID        string `xml:"id,attr"`
	Target    string `xml:"target,attr,omitempty"`
	Pattern   string `xml:"target-pattern,attr,omitempty"`
	Attribute string `xml:"target-attribute,attr,omitempty"`
	Value     string `xml:"target-value,attr,omitempty"`
	Index     int    `xml:"index,attr"`
	Devices   string `xml:"devices,attr"`
}
type fencingTopologyXML struct {
	XMLName xml.Name          `xml:"fencing-topology"`
	Levels  []fencingXMLLevel `xml:"fencing-level"`
}
type fencingEditState struct {
	Fencing []fencingLevel `json:"fencing"`
	Devices []string       `json:"devices"`
	Exists  bool           `json:"-"`
}

func readFencingEdit(ctx context.Context, run clusterConfigCommand) (fencingEditState, error) {
	state := fencingEditState{Fencing: []fencingLevel{}, Devices: []string{}}
	data, err := run(ctx, "cibadmin", []string{"-Ql"}, "")
	if err != nil {
		return state, err
	}
	var doc struct {
		Configuration struct {
			Topology  *fencingTopologyXML `xml:"fencing-topology"`
			Resources struct {
				Inner string `xml:",innerxml"`
			} `xml:"resources"`
		} `xml:"configuration"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		return state, err
	}
	state.Exists = doc.Configuration.Topology != nil
	if state.Exists {
		for _, item := range doc.Configuration.Topology.Levels {
			level := fencingLevel{Index: item.Index, Devices: []string{}}
			switch {
			case item.Target != "":
				level.Type, level.Target = "node", item.Target
			case item.Pattern != "":
				level.Type, level.Target = "pattern", item.Pattern
			case item.Attribute != "":
				level.Type, level.Target, level.Value = "attribute", item.Attribute, item.Value
			default:
				return state, fmt.Errorf("Unsupported fencing target")
			}
			if item.Devices != "" {
				level.Devices = strings.Split(item.Devices, ",")
			}
			state.Fencing = append(state.Fencing, level)
		}
	}
	// Include fencing primitives inside groups and clones, as well as top-level devices.
	decoder := xml.NewDecoder(strings.NewReader("<resources>" + doc.Configuration.Resources.Inner + "</resources>"))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return state, err
		}
		if node, ok := token.(xml.StartElement); ok && node.Name.Local == "primitive" && groupXMLAttr(node.Attr, "class") == "stonith" {
			state.Devices = append(state.Devices, groupXMLAttr(node.Attr, "id"))
		}
	}
	sort.Strings(state.Devices)
	return state, nil
}
func validateFencing(levels []fencingLevel, devices []string) error {
	available := map[string]bool{}
	for _, id := range devices {
		available[id] = true
	}
	for i, level := range levels {
		if strings.TrimSpace(level.Target) == "" {
			return fmt.Errorf("Level #%d: Expected target", i+1)
		}
		switch level.Type {
		case "node", "pattern":
		case "attribute":
			if strings.TrimSpace(level.Value) == "" {
				return fmt.Errorf("Level #%d: Expected value for attribute", i+1)
			}
		default:
			return fmt.Errorf("Level #%d: Invalid target type", i+1)
		}
		if level.Index < 1 || level.Index > 9 {
			return fmt.Errorf("Level #%d: Index must be between 1 and 9", i+1)
		}
		for _, id := range level.Devices {
			if !available[id] {
				return fmt.Errorf("Level #%d: fencing device %s not found or not a fencing device", i+1, id)
			}
		}
	}
	return nil
}
func fencingXML(levels []fencingLevel) ([]byte, error) {
	topology := fencingTopologyXML{}
	for i, level := range levels {
		id := "fencing"
		if i > 0 {
			id = fmt.Sprintf("fencing-%d", i-1)
		}
		item := fencingXMLLevel{ID: id, Index: level.Index, Devices: strings.Join(level.Devices, ",")}
		switch level.Type {
		case "node":
			item.Target = level.Target
		case "pattern":
			item.Pattern = level.Target
		case "attribute":
			item.Attribute, item.Value = level.Target, level.Value
		}
		topology.Levels = append(topology.Levels, item)
	}
	return xml.Marshal(topology)
}
func FencingEditHandler(w http.ResponseWriter, r *http.Request) {
	serveFencingEdit(w, r, runClusterConfigCommand)
}
func serveFencingEdit(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	base := "/cib/live"
	switch r.URL.Path {
	case "/cib/live/fencing/edit", "/cib/live/fencing/edit/":
	case batchBase + "/fencing/edit", batchBase + "/fencing/edit/":
		base = batchBase
	default:
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	if r.Method != http.MethodPost {
		data := map[string]any{"Title": "Edit Fencing Topology", "CIBBase": base}
		if base == batchBase {
			data["BatchMode"] = true
			data["BatchConfig"] = map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}
			data["FencingAPI"] = batchAPI + "/fencing"
		}
		renderTemplate(w, "fencing_edit", data)
		return
	}
	var request struct {
		Fencing *[]fencingLevel `json:"fencing"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&request); err != nil || request.Fencing == nil {
		http.Error(w, "Invalid fencing topology", 400)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "Invalid fencing topology", 400)
		return
	}
	state, err := readFencingEdit(r.Context(), run)
	if err != nil {
		fencingError(w, err)
		return
	}
	if err := validateFencing(*request.Fencing, state.Devices); err != nil {
		fencingError(w, err)
		return
	}
	if len(*request.Fencing) == 0 {
		// Remove the container too, restoring an absent topology rather than
		// leaving an empty fencing-topology element in the CIB.
		if state.Exists {
			if _, err := run(r.Context(), "cibadmin", []string{"--delete", "--xpath", "/cib/configuration/fencing-topology"}, ""); err != nil {
				fencingError(w, err)
				return
			}
		}
	} else {
		data, err := fencingXML(*request.Fencing)
		if err != nil {
			fencingError(w, err)
			return
		}
		if !state.Exists {
			_, err = run(r.Context(), "cibadmin", []string{"-p", "-c", "--modify"}, "<cib><configuration><fencing-topology/></configuration></cib>")
			if err != nil {
				fencingError(w, err)
				return
			}
		}
		if _, err := run(r.Context(), "cibadmin", []string{"-p", "--replace", "--xpath", "/cib/configuration/fencing-topology"}, string(data)); err != nil {
			fencingError(w, err)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "message": "Fencing topology updated successfully"})
}
func fencingError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(422)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "Failed to edit fencing topology: " + err.Error()})
}
func FencingFetchHandler(w http.ResponseWriter, r *http.Request) {
	fetchFencingEdit(w, r, runClusterConfigCommand)
}
func fetchFencingEdit(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	state, err := readFencingEdit(r.Context(), run)
	if err != nil {
		fencingError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}

func BatchFencingEditHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		request := r.Clone(r.Context())
		copiedURL := *r.URL
		request.URL = &copiedURL
		request.URL.Path = batchAPI + "/fencing/update"
		batch.api(w, request)
		return
	}
	serveFencingEdit(w, r, runClusterConfigCommand)
}
