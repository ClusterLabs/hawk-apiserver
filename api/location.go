package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type locationExpression struct {
	Attribute string `json:"attribute"`
	Operation string `json:"operation"`
	Value     string `json:"value"`
	Type      string `json:"type"`
	Kind      string `json:"kind"`
}
type locationRule struct {
	Score       string               `json:"score"`
	Role        string               `json:"role"`
	Operator    string               `json:"operator"`
	Expressions []locationExpression `json:"expressions"`
}
type locationCreate struct {
	ID        string         `json:"id"`
	Resources []string       `json:"resources"`
	Discovery string         `json:"discovery"`
	Rules     []locationRule `json:"rules"`
}

// LocationHelper offers top-level resources, unlike TagHelper's nested objects.
func locationResourcesFromXML(data []byte) ([]string, error) {
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
		case "primitive", "group", "clone", "master", "bundle":
			ids = append(ids, item.ID)
		}
	}
	sortResourceIDs(ids)
	return ids, nil
}

func LocationNewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/locations/new" && r.URL.Path != "/cib/live/locations/new/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	renderTemplate(w, "location_new", map[string]any{"Title": "Create Location Constraint"})
}
func FetchLocationOptions(w http.ResponseWriter, r *http.Request) {
	if !groupFetchMethod(w, r) {
		return
	}
	cib, _, err := GetCIB()
	if err != nil {
		http.Error(w, "Unable to read cluster: "+err.Error(), 500)
		return
	}
	nodes := []string{}
	for _, node := range cib.Configuration.Nodes {
		nodes = append(nodes, node.Uname)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"resources": cib.LocationResources, "nodes": nodes})
}

func simpleLocation(rules []locationRule) bool {
	if len(rules) != 1 || len(rules[0].Expressions) != 1 || rules[0].Role != "" {
		return false
	}
	expression := rules[0].Expressions[0]
	return expression.Attribute == "#uname" && expression.Operation == "eq" && expression.Value != ""
}

var locationScore = regexp.MustCompile(`^-?[0-9]+$`)

func locationCreateArgs(location locationCreate) ([]string, error) {
	if location.ID == "" {
		return nil, fmt.Errorf("ID is required")
	}
	if !groupID.MatchString(location.ID) {
		return nil, fmt.Errorf("Invalid ID")
	}
	if len(location.Resources) == 0 {
		return nil, fmt.Errorf("No resource specified")
	}
	if len(location.Rules) == 0 {
		return nil, fmt.Errorf("No rules specified")
	}
	if location.Discovery != "" && location.Discovery != "always" && location.Discovery != "never" && location.Discovery != "exclusive" {
		return nil, fmt.Errorf("Invalid resource discovery type")
	}
	args := []string{"configure", "location", location.ID}
	if len(location.Resources) > 1 {
		args = append(args, "{")
	}
	args = append(args, location.Resources...)
	if len(location.Resources) > 1 {
		args = append(args, "}")
	}
	if location.Discovery != "" {
		args = append(args, "resource-discovery="+location.Discovery)
	}
	simple := simpleLocation(location.Rules)
	for _, rule := range location.Rules {
		score := strings.TrimSpace(rule.Score)
		if simple {
			switch strings.ToLower(score) {
			case "mandatory", "advisory", "inf", "-inf", "infinity", "-infinity":
			default:
				if !locationScore.MatchString(score) {
					return nil, fmt.Errorf("Invalid score %q", score)
				}
			}
		}
		if len(rule.Expressions) == 0 {
			return nil, fmt.Errorf("No expressions specified")
		}
		if !simple {
			args = append(args, "rule")
			if rule.Role != "" {
				if rule.Role != "Started" && rule.Role != "Master" && rule.Role != "Slave" {
					return nil, fmt.Errorf("Invalid role")
				}
				args = append(args, "$role="+rule.Role)
			}
		}
		args = append(args, score+":")
		operator := rule.Operator
		if operator == "" {
			operator = "and"
		}
		if operator != "and" && operator != "or" {
			return nil, fmt.Errorf("Invalid boolean operator")
		}
		for i, expression := range rule.Expressions {
			attribute, value := strings.TrimSpace(expression.Attribute), strings.TrimSpace(expression.Value)
			if strings.ContainsAny(attribute, "'") && strings.Contains(attribute, `"`) {
				return nil, fmt.Errorf("Attribute contains both single and double quotes")
			}
			if strings.Contains(value, "'") && strings.Contains(value, `"`) {
				return nil, fmt.Errorf("Value contains both single and double quotes")
			}
			if simple {
				args = append(args, value)
				continue
			}
			if i > 0 {
				args = append(args, operator)
			}
			switch expression.Operation {
			case "defined", "not_defined":
				args = append(args, expression.Operation, attribute)
			case "eq", "ne", "lt", "lte", "gte", "gt":
				operation := expression.Operation
				if expression.Type != "" {
					if expression.Type != "string" && expression.Type != "integer" && expression.Type != "version" {
						return nil, fmt.Errorf("Invalid expression type")
					}
					operation = expression.Type + ":" + operation
				}
				args = append(args, attribute, operation, value)
			default:
				return nil, fmt.Errorf("Invalid expression operation")
			}
		}
	}
	return args, nil
}
func LocationCreateHandler(w http.ResponseWriter, r *http.Request) {
	createLocation(w, r, GetCIB, runGroupCommand)
}
func createLocation(w http.ResponseWriter, r *http.Request, read func() (CIB, int, error), run func(context.Context, ...string) error) {
	if !groupFetchMethod(w, r) {
		return
	}
	var location locationCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&location); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	args, err := locationCreateArgs(location)
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
	for _, id := range cib.LocationResources {
		available[id] = true
	}
	for _, id := range location.Resources {
		if !available[id] {
			http.Error(w, "Resource is no longer available: "+id, 409)
			return
		}
	}
	if err := run(r.Context(), args...); err != nil {
		http.Error(w, tagCreateError(location.ID, err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "redirect": "/cib/live/locations/" + url.PathEscape(location.ID) + "/edit?flash=created"})
}
func LocationRubyHandler(w http.ResponseWriter, r *http.Request) {
	resourceRubyHandler(w, r, "Constraint")
}
