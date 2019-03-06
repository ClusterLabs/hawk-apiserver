package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
)

func handleAPIConstraints(w http.ResponseWriter, r *http.Request, cibData string) bool {
	// parse xml into Cib struct
	var cib Cib
	err := xml.Unmarshal([]byte(cibData), &cib)
	if err != nil {
		log.Error(err)
		return false
	}

	cib.Configuration.URLType = "constraints"

	w.Header().Set("Content-Type", "application/json")

	urllist := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(urllist) == 3 {
		// for url api/v[1-9]/constraints
		cib.Configuration.Constraints.URLType = "all"
	} else {
		// for url api/v[1-9]/constraints/{resid}
		consID := urllist[3]

		mapIDType := make(map[string]TypeIndex)
		for li, litem := range cib.Configuration.Constraints.RscLocation {
			mapIDType[litem.Id] = TypeIndex{"location", li}
		}
		for ci, citem := range cib.Configuration.Constraints.RscColocation {
			mapIDType[citem.Id] = TypeIndex{"colocation", ci}
		}
		for oi, oitem := range cib.Configuration.Constraints.RscOrder {
			mapIDType[oitem.Id] = TypeIndex{"order", oi}
		}

		val, ok := mapIDType[consID]
		if ok {
			cib.Configuration.Constraints.URLType = val.Type
			cib.Configuration.Constraints.URLIndex = val.Index
		} else {
			http.Error(w, fmt.Sprintf("No route for %v.", r.URL.Path), 500)
			return false
		}
	}

	jsonData, jsonError := json.Marshal(&cib)
	if jsonError != nil {
		log.Error(jsonError)
		return false
	}

	io.WriteString(w, string(jsonData)+"\n")
	return true
}
