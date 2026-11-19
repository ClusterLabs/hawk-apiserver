package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

type orderEditState struct {
	Order     orderCreate `json:"order"`
	Available []string    `json:"available"`
}

func orderFromXML(data []byte) (orderCreate, error) {
	var item struct {
		XMLName       xml.Name `xml:"rsc_order"`
		ID            string   `xml:"id,attr"`
		Score         string   `xml:"score,attr"`
		Kind          string   `xml:"kind,attr"`
		First         string   `xml:"first,attr"`
		Then          string   `xml:"then,attr"`
		FirstAction   string   `xml:"first-action,attr"`
		ThenAction    string   `xml:"then-action,attr"`
		FirstInstance string   `xml:"first-instance,attr"`
		ThenInstance  string   `xml:"then-instance,attr"`
		Symmetrical   string   `xml:"symmetrical,attr"`
		Sets          []struct {
			Sequential string `xml:"sequential,attr"`
			Action     string `xml:"action,attr"`
			RequireAll string `xml:"require-all,attr"`
			IDRef      string `xml:"id-ref,attr"`
			Refs       []struct {
				ID string `xml:"id,attr"`
			} `xml:"resource_ref"`
		} `xml:"resource_set"`
	}
	if err := xml.Unmarshal(data, &item); err != nil {
		return orderCreate{}, err
	}
	if item.FirstInstance != "" || item.ThenInstance != "" {
		return orderCreate{}, fmt.Errorf("Order constraints with instance selectors cannot be edited in this form")
	}
	order := orderCreate{ID: item.ID, Score: item.Score, Symmetrical: item.Symmetrical != "false" && item.Symmetrical != "0", Resources: []colocationSet{}}
	if order.Score == "" {
		order.Score = item.Kind
	}
	if item.First != "" {
		order.Resources = append(order.Resources, colocationSet{Sequential: true, Action: item.FirstAction, Resources: []string{item.First}}, colocationSet{Sequential: true, Action: item.ThenAction, Resources: []string{item.Then}})
	} else {
		for _, set := range item.Sets {
			if set.RequireAll == "false" || set.RequireAll == "0" || set.IDRef != "" {
				return orderCreate{}, fmt.Errorf("Order constraints with require-all=false or referenced sets cannot be edited in this form")
			}
			entry := colocationSet{Sequential: set.Sequential != "false" && set.Sequential != "0", Action: set.Action, Resources: []string{}}
			for _, ref := range set.Refs {
				entry.Resources = append(entry.Resources, ref.ID)
			}
			order.Resources = append(order.Resources, entry)
		}
	}
	return order, nil
}
func readOrderEdit(ctx context.Context, id string, run clusterConfigCommand) (orderEditState, bool, error) {
	state := orderEditState{}
	data, err := run(ctx, "cibadmin", []string{"-Ql"}, "")
	if err != nil {
		return state, false, err
	}
	var doc struct {
		Configuration struct {
			Orders []struct {
				ID         string     `xml:"id,attr"`
				Inner      string     `xml:",innerxml"`
				Attributes []xml.Attr `xml:",any,attr"`
			} `xml:"constraints>rsc_order"`
		} `xml:"configuration"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		return state, false, err
	}
	state.Available, err = colocationResourcesFromXML(data)
	if err != nil {
		return state, false, err
	}
	for _, item := range doc.Configuration.Orders {
		if item.ID != id {
			continue
		}
		// Re-encode the element attributes while retaining set/ref XML order.
		element := groupXMLPart{XMLName: xml.Name{Local: "rsc_order"}, Attributes: append([]xml.Attr{{Name: xml.Name{Local: "id"}, Value: item.ID}}, item.Attributes...), Inner: item.Inner}
		raw, err := xml.Marshal(element)
		if err != nil {
			return state, false, err
		}
		state.Order, err = orderFromXML(raw)
		return state, true, err
	}
	return state, false, nil
}
func OrderEditHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "cib" || (parts[1] != "live" && parts[1] != batchID) || parts[2] != "orders" || parts[4] != "edit" {
		OrderRubyHandler(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	base := "/cib/" + parts[1]
	data := map[string]any{"Title": "Edit Order", "Editing": true, "OrderID": parts[3], "CIBBase": base}
	if base == batchBase {
		data["BatchMode"] = true
		data["BatchConfig"] = map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}
		data["OrderAPI"] = batchAPI + "/order"
	}
	renderTemplate(w, "order_new", data)
}
func OrderFetchHandler(w http.ResponseWriter, r *http.Request) {
	fetchOrderEdit(w, r, runClusterConfigCommand)
}
func fetchOrderEdit(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	state, found, err := readOrderEdit(r.Context(), r.URL.Query().Get("id"), run)
	if err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
	if !found {
		http.Error(w, "The order constraint does not exist", 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}
func OrderUpdateHandler(w http.ResponseWriter, r *http.Request) {
	updateOrderEdit(w, r, runClusterConfigCommand)
}
func updateOrderEdit(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	var request orderCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	saveOrderEdit(w, r, request, run)
}
func saveOrderEdit(w http.ResponseWriter, r *http.Request, request orderCreate, run clusterConfigCommand) {
	args, err := orderCreateArgs(request)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	state, found, err := readOrderEdit(r.Context(), request.ID, run)
	if err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
	if !found {
		http.Error(w, "The order constraint does not exist", 404)
		return
	}
	available := map[string]bool{}
	for _, id := range state.Available {
		available[id] = true
	}
	for _, set := range request.Resources {
		for _, id := range set.Resources {
			if !available[id] {
				http.Error(w, "Resource is no longer available: "+id, 409)
				return
			}
		}
	}
	tokens := make([]string, 0, len(args)-1)
	for _, arg := range args[1:] {
		tokens = append(tokens, "'"+strings.ReplaceAll(arg, "'", "'\"'\"'")+"'")
	}
	if _, err := run(r.Context(), "crm", []string{"-F", "configure", "load", "update", "-"}, strings.Join(tokens, " ")+"\n"); err != nil {
		http.Error(w, tagCreateError(request.ID, err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Constraint updated successfully"})
}

func orderAction(ctx context.Context, id, newID string, run clusterConfigCommand) (int, error) {
	return constraintAction(ctx, id, newID, "order", run)
}
