package main

import (
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
)

func handleApiAlerts(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	// parse xml into Cib struct
	var cib Cib
	err := xml.Unmarshal([]byte(cib_data), &cib)
	if err != nil {
		log.Error(err)
		return false
	}

	cib.Configuration.URLType = "alerts"

	w.Header().Set("Content-Type", "application/json")
	urllist := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(urllist) == 4 {
		// for url api/v1/configuration/alerts
		cib.Configuration.Alerts.URLType = "all"
	} else {
		// for url api/v1/configuration/nodes/{nodeid}
		cib.Configuration.Alerts.URLType = "alert"

		alertIndex := urllist[4]
		var index int = -1
		for i, item := range cib.Configuration.Alerts.Alert {
			if alertIndex == item.Id {
				index = i
				break
			}
		}
		if index == -1 {
			http.Error(w, fmt.Sprintf("No route for %v.", r.URL.Path), 500)
			return false
		}

		cib.Configuration.Alerts.URLIndex = index
	}

	jsonData, jsonError := MarshalOut(r, &cib)
	if jsonError != nil {
		log.Error(jsonError)
		return false
	}

	io.WriteString(w, string(jsonData)+"\n")
	return true
}
