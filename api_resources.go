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

func handleAPIResources(w http.ResponseWriter, r *http.Request, cibData string) bool {
	// parse xml into Cib struct
	var cib Cib
	err := xml.Unmarshal([]byte(cibData), &cib)
	if err != nil {
		log.Error(err)
		return false
	}

	cib.Configuration.URLType = "resources"

	w.Header().Set("Content-Type", "application/json")

	urllist := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(urllist) == 4 {
		// for url api/v1/configuration/resources
		cib.Configuration.Resources.URLType = "all"
	} else {
		// for url api/v1/configuration/resources/{resid}
		resID := urllist[4]

		mapIDType := make(map[string]TypeIndex)
		for pi, pitem := range cib.Configuration.Resources.Primitive {
			mapIDType[pitem.Id] = TypeIndex{"primitive", pi}
		}
		for gi, gitem := range cib.Configuration.Resources.Group {
			mapIDType[gitem.Id] = TypeIndex{"group", gi}
		}
		for ci, citem := range cib.Configuration.Resources.Clone {
			mapIDType[citem.Id] = TypeIndex{"clone", ci}
		}
		for mi, mitem := range cib.Configuration.Resources.Master {
			mapIDType[mitem.Id] = TypeIndex{"master", mi}
		}

		val, ok := mapIDType[resID]
		if ok {
			cib.Configuration.Resources.URLType = val.Type
			cib.Configuration.Resources.URLIndex = val.Index
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
