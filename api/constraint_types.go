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
	}
}

func ConstraintTypesHandler(w http.ResponseWriter, r *http.Request) {
	base := "/cib/live"
	switch r.URL.Path {
	case "/cib/live/constraints/types", "/cib/live/constraints/types/":
	case batchBase + "/constraints/types", batchBase + "/constraints/types/":
		base = batchBase
	default:
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data := map[string]any{"Title": "Add Constraint", "ConstraintTypes": constraintTypes(), "CIBBase": base}
	if base == batchBase {
		data["BatchMode"] = true
		data["BatchConfig"] = map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}
	}
	renderTemplate(w, "constraint_types", data)
}
