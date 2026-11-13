package api

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type clusterConfigCommand func(context.Context, string, []string, string) ([]byte, error)

func runClusterConfigCommand(ctx context.Context, name string, args []string, input string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return out, nil
}

var clusterConfigSections = []struct{ Name, Set, Element string }{{"crm_config", "cib-bootstrap-options", "cluster_property_set"}, {"rsc_defaults", "rsc-options", "meta_attributes"}, {"op_defaults", "op-options", "meta_attributes"}}

type configProperty struct {
	Name     string `xml:"name,attr"`
	Advanced string `xml:"advanced,attr"`
	Short    []struct {
		Lang string `xml:"lang,attr"`
		Text string `xml:",chardata"`
	} `xml:"shortdesc"`
	Long []struct {
		Lang string `xml:"lang,attr"`
		Text string `xml:",chardata"`
	} `xml:"longdesc"`
	Content struct {
		Type    string `xml:"type,attr"`
		Default string `xml:"default,attr"`
		Options []struct {
			Value string `xml:"value,attr"`
		} `xml:"option"`
	} `xml:"content"`
}

func clusterPropertyOptions(data []byte, includeHidden ...bool) ([]SelectOption, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	options := []SelectOption{}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "parameter" {
			continue
		}
		var p configProperty
		if err := decoder.DecodeElement(&p, &start); err != nil {
			return nil, err
		}
		short, long := "", ""
		for _, s := range p.Short {
			if short == "" || s.Lang == "en" {
				short = s.Text
			}
			if s.Lang == "en" {
				break
			}
		}
		for _, s := range p.Long {
			if long == "" || s.Lang == "en" {
				long = s.Text
			}
			if s.Lang == "en" {
				break
			}
		}
		uneditable := p.Advanced == "1" || strings.Contains(strings.ToLower(short+" "+long), "advanced use only")
		switch p.Name {
		case "cluster-infrastructure", "dc-version", "expected-quorum-votes", "have-watchdog":
			uneditable = true
		}
		if uneditable && (len(includeHidden) == 0 || !includeHidden[0]) {
			continue
		}
		values := []string{}
		for _, o := range p.Content.Options {
			values = append(values, o.Value)
		}
		if len(values) == 0 && (p.Content.Type == "enum" || p.Content.Type == "select") {
			if i := strings.Index(strings.ToLower(long), "allowed values:"); i >= 0 {
				for _, v := range strings.Split(strings.Split(long[i+15:], "\n")[0], ",") {
					if v = strings.TrimSpace(v); v != "" {
						values = append(values, v)
					}
				}
			}
		}
		options = append(options, SelectOption{Uneditable: uneditable, Name: p.Name, Type: p.Content.Type, DefaultValue: p.Content.Default, Shortdesc: short, Longdesc: long, PossibleValues: values})
	}
	return options, nil
}
func loadClusterPropertyOptions(ctx context.Context, run clusterConfigCommand) ([]SelectOption, error) {
	out, err := run(ctx, "crm_attribute", []string{"--list-options=cluster", "--all", "--output-as=xml"}, "")
	if err == nil {
		options, parseErr := clusterPropertyOptions(out, true)
		if parseErr == nil && len(options) > 0 {
			return options, nil
		}
	}
	merged := map[string]SelectOption{}
	for _, dir := range []string{"/usr/libexec/pacemaker", "/usr/lib/pacemaker", "/usr/lib64/pacemaker"} {
		for _, name := range []string{"pengine", "crmd", "cib", "pacemaker-schedulerd", "pacemaker-controld", "pacemaker-based"} {
			path := dir + "/" + name
			info, e := os.Stat(path)
			if e != nil || info.Mode()&0111 == 0 {
				continue
			}
			data, e := run(ctx, path, []string{"metadata"}, "")
			if e != nil {
				continue
			}
			options, e := clusterPropertyOptions(data, true)
			if e != nil {
				continue
			}
			for _, o := range options {
				merged[o.Name] = o
			}
		}
	}
	if len(merged) == 0 {
		return nil, fmt.Errorf("Unable to load Pacemaker cluster property metadata")
	}
	options := []SelectOption{}
	for _, o := range merged {
		options = append(options, o)
	}
	return options, nil
}
func clusterDefaultOptions(section string) []SelectOption {
	result := []SelectOption{}
	if section == "rsc_defaults" {
		for _, p := range GetRscDefaults() {
			result = append(result, SelectOption{Name: p.Name, Type: p.Content.Type, DefaultValue: p.Content.Default, Longdesc: p.Longdesc, Shortdesc: p.Shortdesc, PossibleValues: p.Content.PossibleValues})
		}
		return result
	}
	defaults := []FullPrimitive_Action_MetaParameter{actionDefaults.Interval, actionDefaults.Timeout, actionDefaults.Requires, actionDefaults.Enabled, actionDefaults.Role, actionDefaults.OnFail, actionDefaults.StartDelay, actionDefaults.IntervalOrigin, actionDefaults.RecordPending, actionDefaults.Description}
	for _, p := range defaults {
		result = append(result, NewSelectOption(p.Content.Default, p))
	}
	return result
}

type clusterConfigState struct {
	Options map[string][]SelectOption
	Current map[string]map[string]Nvpair
}

func loadClusterConfig(ctx context.Context, run clusterConfigCommand) (clusterConfigState, error) {
	state := clusterConfigState{Options: map[string][]SelectOption{}, Current: map[string]map[string]Nvpair{}}
	properties, err := loadClusterPropertyOptions(ctx, run)
	if err != nil {
		return state, err
	}
	out, err := run(ctx, "cibadmin", []string{"-Ql"}, "")
	if err != nil {
		return state, err
	}
	var cib struct {
		Configuration struct {
			Properties []struct {
				ID    string   `xml:"id,attr"`
				Pairs []Nvpair `xml:"nvpair"`
			} `xml:"crm_config>cluster_property_set"`
			Rsc []struct {
				ID    string   `xml:"id,attr"`
				Pairs []Nvpair `xml:"nvpair"`
			} `xml:"rsc_defaults>meta_attributes"`
			Ops []struct {
				ID    string   `xml:"id,attr"`
				Pairs []Nvpair `xml:"nvpair"`
			} `xml:"op_defaults>meta_attributes"`
		} `xml:"configuration"`
	}
	if err := xml.Unmarshal(out, &cib); err != nil {
		return state, err
	}
	for _, section := range clusterConfigSections {
		state.Current[section.Name] = map[string]Nvpair{}
		order := map[string]int{}
		switch section.Name {
		case "crm_config":
			state.Options[section.Name] = properties
			for _, set := range cib.Configuration.Properties {
				if set.ID == section.Set {
					for _, p := range set.Pairs {
						order[p.Name] = len(order)
						state.Current[section.Name][p.Name] = p
					}
				}
			}
		case "rsc_defaults":
			state.Options[section.Name] = clusterDefaultOptions(section.Name)
			for _, set := range cib.Configuration.Rsc {
				if set.ID == section.Set {
					for _, p := range set.Pairs {
						order[p.Name] = len(order)
						state.Current[section.Name][p.Name] = p
					}
				}
			}
		case "op_defaults":
			state.Options[section.Name] = clusterDefaultOptions(section.Name)
			for _, set := range cib.Configuration.Ops {
				if set.ID == section.Set {
					for _, p := range set.Pairs {
						order[p.Name] = len(order)
						state.Current[section.Name][p.Name] = p
					}
				}
			}
		}
		options := []SelectOption{}
		for _, option := range state.Options[section.Name] {
			if option.Uneditable {
				if _, exists := state.Current[section.Name][option.Name]; !exists {
					continue
				}
				// Ruby attrlist supplies a plain text fallback for existing keys
				// absent from the editable mapping. They remain ignored on save.
				option.Type = "string"
				option.PossibleValues = nil
				option.DefaultValue = ""
			}
			options = append(options, option)
		}
		for i := range options {
			if p, ok := state.Current[section.Name][options[i].Name]; ok {
				options[i].CibID = p.ID
				if options[i].CibID == "" {
					options[i].CibID = section.Set + "-" + p.Name
				}
				options[i].CibValue = p.Value
			}
		}
		sort.SliceStable(options, func(i, j int) bool {
			left, l := order[options[i].Name]
			right, r := order[options[j].Name]
			if l && r {
				return left < right
			}
			if l != r {
				return l
			}
			return options[i].Name < options[j].Name
		})
		state.Options[section.Name] = options
	}
	return state, nil
}

type clusterConfigMutation struct {
	Name string
	Args []string
}

func clusterConfigMutations(state clusterConfigState, values map[string]map[string]string) ([]clusterConfigMutation, error) {
	mutations := []clusterConfigMutation{}
	for name := range values {
		if _, ok := state.Options[name]; !ok {
			return nil, fmt.Errorf("Invalid configuration section: %s", name)
		}
	}
	for _, section := range clusterConfigSections {
		submitted, ok := values[section.Name]
		if !ok {
			return nil, fmt.Errorf("Missing configuration section: %s", section.Name)
		}
		allowed := map[string]SelectOption{}
		for _, o := range state.Options[section.Name] {
			allowed[o.Name] = o
		}
		for name, value := range submitted {
			option, ok := allowed[name]
			if !ok {
				return nil, fmt.Errorf("Property is read-only, advanced or unknown: %s", name)
			}
			if option.Uneditable {
				continue
			}
			if strings.ContainsRune(value, 0) {
				return nil, fmt.Errorf("Invalid value for %s", name)
			}
			if value != "" && option.Type == "boolean" && value != "true" && value != "false" {
				return nil, fmt.Errorf("%s must be true or false", name)
			}
		}
		for _, option := range state.Options[section.Name] {
			if option.Uneditable {
				continue
			}
			value := submitted[option.Name]
			old, exists := state.Current[section.Name][option.Name]
			if (!exists && value == "") || (exists && old.Value == value) {
				continue
			}
			if value == "" {
				mutations = append(mutations, clusterConfigMutation{"crm_attribute", []string{"--type", section.Name, "--set-name", section.Set, "--name", option.Name, "--delete"}})
				continue
			}
			// Pass values as argv, never shell text. crm_attribute retains existing nvpair IDs.
			mutations = append(mutations, clusterConfigMutation{"crm_attribute", []string{"--type", section.Name, "--set-name", section.Set, "--name", option.Name, "--update", value}})
		}
	}
	return mutations, nil
}
func ClusterConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/crm_config/edit" && r.URL.Path != "/cib/live/crm_config/edit/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	renderTemplate(w, "cluster_config", map[string]any{"Title": "Cluster Configuration"})
}
func ClusterConfigOptionsHandler(w http.ResponseWriter, r *http.Request) {
	clusterConfigAPI(w, r, runClusterConfigCommand, false)
}
func ClusterConfigUpdateHandler(w http.ResponseWriter, r *http.Request) {
	clusterConfigAPI(w, r, runClusterConfigCommand, true)
}
func clusterConfigAPI(w http.ResponseWriter, r *http.Request, run clusterConfigCommand, update bool) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var request struct {
		Section string                       `json:"section"`
		Values  map[string]map[string]string `json:"values"`
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	if !update && request.Section != "crm_config" && request.Section != "rsc_defaults" && request.Section != "op_defaults" {
		http.Error(w, "Invalid section", 400)
		return
	}
	state, err := loadClusterConfig(r.Context(), run)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !update {
		_ = json.NewEncoder(w).Encode(SelectContent{Options: state.Options[request.Section]})
		return
	}
	mutations, err := clusterConfigMutations(state, request.Values)
	if err != nil {
		w.WriteHeader(422)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	for _, mutation := range mutations {
		if _, err := run(r.Context(), mutation.Name, mutation.Args, ""); err != nil {
			w.WriteHeader(422)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update configuration: " + err.Error()})
			return
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "message": "Configuration updated successfully"})
}
