package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

type alertAttributeSet struct {
	Pairs []Nvpair `xml:"nvpair"`
}
type alertXMLValues struct {
	Params []alertAttributeSet `xml:"instance_attributes"`
	Meta   []alertAttributeSet `xml:"meta_attributes"`
}

func alertAttributeMap(sets []alertAttributeSet) map[string]string {
	values := make(map[string]string)
	if len(sets) > 0 {
		for _, pair := range sets[0].Pairs {
			values[pair.Name] = pair.Value
		}
	}
	return values
}
func loadAlert(ctx context.Context, id string, run clusterConfigCommand) (alertCreate, bool, error) {
	data, err := run(ctx, "cibadmin", []string{"-Ql"}, "")
	if err != nil {
		return alertCreate{}, false, err
	}
	var document struct {
		Configuration struct {
			Alerts []struct {
				ID   string `xml:"id,attr"`
				Path string `xml:"path,attr"`
				alertXMLValues
				Recipients []struct {
					ID    string `xml:"id,attr"`
					Value string `xml:"value,attr"`
					alertXMLValues
				} `xml:"recipient"`
			} `xml:"alerts>alert"`
		} `xml:"configuration"`
	}
	if err := xml.Unmarshal(data, &document); err != nil {
		return alertCreate{}, false, err
	}
	for _, item := range document.Configuration.Alerts {
		if item.ID != id {
			continue
		}
		alert := alertCreate{ID: item.ID, Path: item.Path, Params: alertAttributeMap(item.Params), Meta: alertAttributeMap(item.Meta), Recipients: []alertRecipient{}}
		for _, recipient := range item.Recipients {
			alert.Recipients = append(alert.Recipients, alertRecipient{ID: recipient.ID, Value: recipient.Value, Params: alertAttributeMap(recipient.Params), Meta: alertAttributeMap(recipient.Meta)})
		}
		return alert, true, nil
	}
	return alertCreate{}, false, nil
}
func AlertEditHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "cib" || parts[1] != "live" || parts[2] != "alerts" || parts[4] != "edit" {
		AlertRubyHandler(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	renderTemplate(w, "alert_new", map[string]any{"Title": "Edit Alert", "Editing": true, "AlertID": parts[3]})
}
func AlertFetchHandler(w http.ResponseWriter, r *http.Request) {
	fetchAlert(w, r, runClusterConfigCommand)
}
func fetchAlert(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	alert, found, err := loadAlert(r.Context(), r.URL.Query().Get("id"), run)
	if err != nil {
		http.Error(w, "Unable to read alert: "+err.Error(), 500)
		return
	}
	if !found {
		http.Error(w, "The alert does not exist", 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(alert)
}
func alertUpdateCommand(alert alertCreate) (string, error) {
	args, err := alertCreateArgs(alert)
	if err != nil {
		return "", err
	}
	tokens := make([]string, 0, len(args)-1)
	for _, arg := range args[1:] {
		if strings.ContainsAny(arg, "\r\n\x00") {
			return "", fmt.Errorf("Alert fields must not contain line breaks")
		}
		tokens = append(tokens, "'"+strings.ReplaceAll(arg, "'", "'\"'\"'")+"'")
	}
	return strings.Join(tokens, " ") + "\n", nil
}
func AlertUpdateHandler(w http.ResponseWriter, r *http.Request) {
	updateAlert(w, r, runClusterConfigCommand)
}
func updateAlert(w http.ResponseWriter, r *http.Request, run clusterConfigCommand) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	var request alertCreate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	_, found, err := loadAlert(r.Context(), request.ID, run)
	if err != nil {
		http.Error(w, "Unable to read alert: "+err.Error(), 500)
		return
	}
	if !found {
		http.Error(w, "The alert does not exist", 404)
		return
	}
	command, err := alertUpdateCommand(request)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if _, err := run(r.Context(), "crm", []string{"-F", "configure", "load", "update", "/dev/stdin"}, command); err != nil {
		http.Error(w, tagCreateError(request.ID, err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Alert updated successfully"})
}
