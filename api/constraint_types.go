package api

import "net/http"

type constraintType struct {
	Name       string
	Path       string
	Icon       string
	HelpFilter string
}

func constraintTypes() []constraintType {
	return []constraintType{
		{Name: "Location", Path: "locations", Icon: "globe", HelpFilter: "location"},
		{Name: "Colocation", Path: "colocations", Icon: "tasks", HelpFilter: "colocation"},
		{Name: "Order", Path: "orders", Icon: "sort-amount-down", HelpFilter: "order"},
		{Name: "Ticket", Path: "tickets", Icon: "ticket", HelpFilter: "ticket"},
	}
}

func ConstraintTypesHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/constraints/types" && r.URL.Path != "/cib/live/constraints/types/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	renderTemplate(w, "constraint_types", map[string]any{
		"Title":           "Add Constraint",
		"ConstraintTypes": constraintTypes(),
	})
}
