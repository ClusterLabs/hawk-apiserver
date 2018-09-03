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

func handleApiConstraints(version string, w http.ResponseWriter, r *http.Request, cib_data string) bool {
	m := map[string]func(http.ResponseWriter, *http.Request, string) bool{
		"v1": handleApiConstraintsV1,
		"v2": handleApiConstraintsV2,
	}

	return m[version](w, r, cib_data)
}

func handleApiConstraintsV1(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	// parse xml into Cib struct
	var cib Cib
	err := xml.Unmarshal([]byte(cib_data), &cib)
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
		consId := urllist[3]

		mapIdType := make(map[string]TypeIndex)
		for li, litem := range cib.Configuration.Constraints.RscLocation {
			mapIdType[litem.Id] = TypeIndex{"location", li}
		}
		for ci, citem := range cib.Configuration.Constraints.RscColocation {
			mapIdType[citem.Id] = TypeIndex{"colocation", ci}
		}
		for oi, oitem := range cib.Configuration.Constraints.RscOrder {
			mapIdType[oitem.Id] = TypeIndex{"order", oi}
		}

		val, ok := mapIdType[consId]
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

func handleApiConstraintsV2(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	fmt.Printf("handleApiConstraintsV2")
	return true
}
