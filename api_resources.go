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

func handleApiResources(version string, w http.ResponseWriter, r *http.Request, cib_data string) bool {
	m := map[string]func(http.ResponseWriter, *http.Request, string) bool{
		"v1": handleApiResourcesV1,
		"v2": handleApiResourcesV2,
	}

	return m[version](w, r, cib_data)
}

func handleApiResourcesV1(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	// parse xml into Cib struct
	var cib Cib
	err := xml.Unmarshal([]byte(cib_data), &cib)
	if err != nil {
		log.Error(err)
		return false
	}

	cib.Config.Type = "resources"

	w.Header().Set("Content-Type", "application/json")

	urllist := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(urllist) == 3 {
		// for url api/v[1-9]/resources
		cib.Config.Resources.Type = "all"
	} else {
		// for url api/v[1-9]/resources/{resid}
		resId := urllist[3]

		mapIdType := make(map[string]TypeIndex)
		for pi, pitem := range cib.Config.Resources.Primitive {
			mapIdType[pitem.Id] = TypeIndex{"primitive", pi}
		}
		for gi, gitem := range cib.Config.Resources.Group {
			mapIdType[gitem.Id] = TypeIndex{"group", gi}
		}
		for ci, citem := range cib.Config.Resources.Clone {
			mapIdType[citem.Id] = TypeIndex{"clone", ci}
		}
		for mi, mitem := range cib.Config.Resources.Master {
			mapIdType[mitem.Id] = TypeIndex{"master", mi}
		}

		val, ok := mapIdType[resId]
		if ok {
			cib.Config.Resources.Type = val.Type
			cib.Config.Resources.Index = val.Index
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

func handleApiResourcesV2(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	fmt.Printf("handleApiResourcesV2")
	return true
}
