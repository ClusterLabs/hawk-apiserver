package main

import (
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
)

func handleApiResources(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	// parse xml into Cib struct
	var cib Cib
	err := xml.Unmarshal([]byte(cib_data), &cib)
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
		resId := urllist[4]

		mapIdType := make(map[string]TypeIndex)
		for pi, pitem := range cib.Configuration.Resources.Primitive {
			mapIdType[pitem.Id] = TypeIndex{"primitive", pi}
		}
		for gi, gitem := range cib.Configuration.Resources.Group {
			mapIdType[gitem.Id] = TypeIndex{"group", gi}
		}
		for ci, citem := range cib.Configuration.Resources.Clone {
			mapIdType[citem.Id] = TypeIndex{"clone", ci}
		}
		for mi, mitem := range cib.Configuration.Resources.Master {
			mapIdType[mitem.Id] = TypeIndex{"master", mi}
		}

		val, ok := mapIdType[resId]
		if ok {
			cib.Configuration.Resources.URLType = val.Type
			cib.Configuration.Resources.URLIndex = val.Index
		} else {
			http.Error(w, fmt.Sprintf("No route for %v.", r.URL.Path), 500)
			return false
		}
	}

	jsonData, jsonError := MarshalOut(r, &cib)
	if jsonError != nil {
		log.Error(jsonError)
		return false
	}

	io.WriteString(w, string(jsonData)+"\n")
	return true
}
