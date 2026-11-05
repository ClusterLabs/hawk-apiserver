package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
)

type alertRecipient struct {
	ID     string            `json:"id"`
	Value  string            `json:"value"`
	Params map[string]string `json:"params"`
	Meta   map[string]string `json:"meta"`
}

type alertCreate struct {
	ID         string            `json:"id"`
	Path       string            `json:"path"`
	Recipients []alertRecipient  `json:"recipients"`
	Params     map[string]string `json:"params"`
	Meta       map[string]string `json:"meta"`
}

func AlertNewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/alerts/new" && r.URL.Path != "/cib/live/alerts/new/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	renderTemplate(w, "alert_new", map[string]any{"Title": "Create Alert"})
}

func alertMetaOptions() []SelectOption {
	return []SelectOption{
		{Name: "timeout", Type: "string", DefaultValue: "30s", Longdesc: "If the alert agent does not complete within this amount of time, it will be terminated."},
		{Name: "timestamp-format", Type: "string", DefaultValue: "%H:%M:%S.%06N", Longdesc: "Format the cluster will use when sending the event’s timestamp to the agent. This is a string as used with the date(1) command."},
	}
}

func FetchAlertMetaAttributes(w http.ResponseWriter, r *http.Request) {
	if !groupFetchMethod(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(SelectContent{Options: alertMetaOptions()})
}

func alertCreateArgs(alert alertCreate) ([]string, error) {
	if alert.ID == "" {
		return nil, fmt.Errorf("ID is required")
	}
	if !groupID.MatchString(alert.ID) {
		return nil, fmt.Errorf("Invalid ID")
	}
	if alert.Path == "" {
		return nil, fmt.Errorf("Path is required")
	}
	// File.symlink? in Ruby accepts dangling symlinks as well as existing paths.
	if _, err := os.Stat(alert.Path); err != nil {
		info, linkErr := os.Lstat(alert.Path)
		if linkErr != nil || info.Mode()&os.ModeSymlink == 0 {
			return nil, fmt.Errorf("Path must be an existing file: File not found")
		}
	}
	args := []string{"configure", "alert", alert.ID, alert.Path}
	appendAttributes := func(section string, values map[string]string) error {
		if len(values) == 0 {
			return nil
		}
		keys := make([]string, 0, len(values))
		for name := range values {
			if name == "" {
				return fmt.Errorf("Attribute name is required")
			}
			if section == "meta" && name != "timeout" && name != "timestamp-format" {
				return fmt.Errorf("Unknown alert meta attribute: %s", name)
			}
			keys = append(keys, name)
		}
		sort.Strings(keys)
		args = append(args, section)
		for _, key := range keys {
			args = append(args, key+"="+values[key])
		}
		return nil
	}
	if err := appendAttributes("attributes", alert.Params); err != nil {
		return nil, err
	}
	if err := appendAttributes("meta", alert.Meta); err != nil {
		return nil, err
	}
	for _, recipient := range alert.Recipients {
		if !tagRefID.MatchString(recipient.ID) {
			return nil, fmt.Errorf("Invalid ID")
		}
		if recipient.Value == "" {
			return nil, fmt.Errorf("Recipient path is required")
		}
		args = append(args, "to", "{", recipient.Value)
		if err := appendAttributes("attributes", recipient.Params); err != nil {
			return nil, err
		}
		if err := appendAttributes("meta", recipient.Meta); err != nil {
			return nil, err
		}
		args = append(args, "}")
	}
	return args, nil
}

func AlertCreateHandler(w http.ResponseWriter, r *http.Request) { createAlert(w, r, runGroupCommand) }

func createAlert(w http.ResponseWriter, r *http.Request, run func(context.Context, ...string) error) {
	if !groupFetchMethod(w, r) {
		return
	}
	var alert alertCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&alert); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	args, err := alertCreateArgs(alert)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := run(r.Context(), args...); err != nil {
		http.Error(w, tagCreateError(alert.ID, err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "redirect": "/cib/live/alerts/" + url.PathEscape(alert.ID) + "/edit?flash=created"})
}

func AlertRubyHandler(w http.ResponseWriter, r *http.Request) { resourceRubyHandler(w, r, "Alert") }
