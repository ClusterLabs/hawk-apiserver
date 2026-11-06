package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
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
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[4] != "edit" {
		resourceRubyHandler(w, r, "Constraint")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	renderTemplate(w, "location_new", map[string]any{"Title": "Edit Location Constraint", "EditMode": true, "LocationID": parts[3]})
}

// Read the original XML so unsupported rules are never silently discarded.
func readLocation(ctx context.Context, id string) (locationCreate, error) {
	if !groupID.MatchString(id) {
		return locationCreate{}, fmt.Errorf("Invalid ID")
	}
	data, err := exec.CommandContext(ctx, "cibadmin", "-Q", "--xpath", "//constraints/rsc_location[@id='"+id+"']").Output()
	if err != nil {
		return locationCreate{}, fmt.Errorf("Unable to read location constraint: %w", err)
	}
	return locationFromXML(data)
}

func locationFromXML(data []byte) (locationCreate, error) {
	var x struct {
		ID        string  `xml:"id,attr"`
		Resource  string  `xml:"rsc,attr"`
		Node      string  `xml:"node,attr"`
		Score     *string `xml:"score,attr"`
		Discovery string  `xml:"resource-discovery,attr"`
		Sets      []struct {
			Resources []struct {
				ID string `xml:"id,attr"`
			} `xml:"resource_ref"`
		} `xml:"resource_set"`
		Rules []struct {
			Score          string `xml:"score,attr"`
			ScoreAttribute string `xml:"score-attribute,attr"`
			Role           string `xml:"role,attr"`
			Operator       string `xml:"boolean-op,attr"`
			Children       []struct {
				XMLName   xml.Name
				Attribute string `xml:"attribute,attr"`
				Operation string `xml:"operation,attr"`
				Value     string `xml:"value,attr"`
				Type      string `xml:"type,attr"`
			} `xml:",any"`
		} `xml:"rule"`
	}
	if err := xml.Unmarshal(data, &x); err != nil {
		return locationCreate{}, err
	}
	l := locationCreate{ID: x.ID, Discovery: x.Discovery, Resources: []string{}, Rules: []locationRule{}}
	if x.Resource != "" {
		l.Resources = append(l.Resources, x.Resource)
	}
	for _, set := range x.Sets {
		for _, resource := range set.Resources {
			l.Resources = append(l.Resources, resource.ID)
		}
	}
	if x.Score != nil {
		l.Rules = append(l.Rules, locationRule{Score: *x.Score, Expressions: []locationExpression{{Attribute: "#uname", Operation: "eq", Value: x.Node, Kind: "uname"}}})
	} else {
		for _, r := range x.Rules {
			rule := locationRule{Score: r.Score, Role: r.Role, Operator: r.Operator, Expressions: []locationExpression{}}
			if rule.Score == "" {
				rule.Score = r.ScoreAttribute
			}
			if rule.Operator == "" {
				rule.Operator = "and"
			}
			for _, e := range r.Children {
				if e.XMLName.Local != "expression" {
					return locationCreate{}, fmt.Errorf("Constraint is too complex - it contains nested rules or date expressions")
				}
				kind := "attr-val"
				if strings.HasPrefix(e.Attribute, "#") {
					kind = "uname"
				}
				if e.Operation == "defined" || e.Operation == "not_defined" {
					kind = "attr-def"
				}
				rule.Expressions = append(rule.Expressions, locationExpression{Attribute: e.Attribute, Operation: e.Operation, Value: e.Value, Type: e.Type, Kind: kind})
			}
			l.Rules = append(l.Rules, rule)
		}
	}
	return l, nil
}

func FetchLocation(w http.ResponseWriter, r *http.Request) {
	if !groupFetchMethod(w, r) {
		return
	}
	l, err := readLocation(r.Context(), r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(l)
}

func LocationUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if !groupFetchMethod(w, r) {
		return
	}
	var l locationCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&l); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	args, err := locationCreateArgs(l)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if _, err = readLocation(r.Context(), l.ID); err != nil {
		http.Error(w, err.Error(), 409)
		return
	}
	cib, _, err := GetCIB()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	available := map[string]bool{}
	for _, id := range cib.LocationResources {
		available[id] = true
	}
	for _, id := range l.Resources {
		if !available[id] {
			http.Error(w, "Resource is no longer available: "+id, 409)
			return
		}
	}
	// crmsh reads a configuration document, not a shell command.
	tokens := make([]string, 0, len(args)-1)
	for _, arg := range args[1:] {
		if strings.ContainsAny(arg, "\r\n") {
			http.Error(w, "Newlines are not allowed", 400)
			return
		}
		tokens = append(tokens, "\""+strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(arg)+"\"")
	}
	cmd := exec.CommandContext(r.Context(), "/usr/sbin/crm", "-F", "configure", "load", "update", "/dev/stdin")
	cmd.Stdin = strings.NewReader(strings.Join(tokens, " ") + "\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		http.Error(w, strings.TrimSpace(string(out)), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "redirect": "/cib/live/locations/" + url.PathEscape(l.ID) + "/edit?flash=updated"})
}
