package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type orderCreate struct {
	ID          string          `json:"id"`
	Score       string          `json:"score"`
	Symmetrical bool            `json:"symmetrical"`
	Resources   []colocationSet `json:"resources"`
}

func OrderNewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/orders/new" && r.URL.Path != "/cib/live/orders/new/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	renderTemplate(w, "order_new", map[string]any{"Title": "Create Order"})
}

func orderCreateArgs(order orderCreate) ([]string, error) {
	if order.ID == "" {
		return nil, fmt.Errorf("ID is required")
	}
	if !groupID.MatchString(order.ID) {
		return nil, fmt.Errorf("Invalid ID")
	}
	score := strings.TrimSpace(order.Score)
	if score == "" {
		return nil, fmt.Errorf("Score is required")
	}
	switch strings.ToLower(score) {
	case "mandatory", "optional", "serialize", "inf", "-inf", "infinity", "-infinity":
	default:
		if !locationScore.MatchString(score) {
			return nil, fmt.Errorf("Invalid score value")
		}
	}
	if len(order.Resources) < 2 {
		return nil, fmt.Errorf("Constraint must consist of at least two separate resources")
	}
	args := []string{"configure", "order", order.ID, score + ":"}
	// Unlike colocation, Order#shell_syntax preserves set and resource order.
	for _, set := range order.Resources {
		switch set.Action {
		case "", "start", "promote", "demote", "stop":
		default:
			return nil, fmt.Errorf("Invalid action")
		}
		if len(set.Resources) == 0 {
			return nil, fmt.Errorf("No resource specified")
		}
		if !set.Sequential {
			args = append(args, "(")
		}
		for _, id := range set.Resources {
			if !groupID.MatchString(id) {
				return nil, fmt.Errorf("Invalid resource ID")
			}
			if set.Action != "" {
				id += ":" + set.Action
			}
			args = append(args, id)
		}
		if !set.Sequential {
			args = append(args, ")")
		}
	}
	if !order.Symmetrical {
		args = append(args, "symmetrical=false")
	}
	return args, nil
}

func OrderCreateHandler(w http.ResponseWriter, r *http.Request) {
	createOrder(w, r, GetCIB, runGroupCommand)
}
func createOrder(w http.ResponseWriter, r *http.Request, read func() (CIB, int, error), run func(context.Context, ...string) error) {
	if !groupFetchMethod(w, r) {
		return
	}
	var order orderCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&order); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	args, err := orderCreateArgs(order)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	cib, _, err := read()
	if err != nil {
		http.Error(w, "Unable to read resources: "+err.Error(), 500)
		return
	}
	// OrderHelper and ColocationHelper offer the same resources and templates.
	available := map[string]bool{}
	for _, id := range cib.ColocationResources {
		available[id] = true
	}
	for _, set := range order.Resources {
		for _, id := range set.Resources {
			if !available[id] {
				http.Error(w, "Resource is no longer available: "+id, 409)
				return
			}
		}
	}
	if err := run(r.Context(), args...); err != nil {
		http.Error(w, tagCreateError(order.ID, err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "redirect": "/cib/live/orders/" + url.PathEscape(order.ID) + "/edit?flash=created"})
}
func OrderRubyHandler(w http.ResponseWriter, r *http.Request) {
	resourceRubyHandler(w, r, "Constraint")
}
