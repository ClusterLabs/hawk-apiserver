package main

import (
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
)

func handleApiAcls(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	// parse xml into Cib struct
	var cib Cib
	err := xml.Unmarshal([]byte(cib_data), &cib)
	if err != nil {
		log.Error(err)
		return false
	}

	cib.Configuration.URLType = "acls"

	w.Header().Set("Content-Type", "application/json")

	urllist := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(urllist) == 4 {
		// for url api/v1/configuration/constraints
		cib.Configuration.Acls.URLType = "all"
	} else {
		// for url api/v1/configuration/constraints/{resid}
		aclId := urllist[4]

		mapIdType := make(map[string]TypeIndex)
		for ti, titem := range cib.Configuration.Acls.AclTarget {
			mapIdType[titem.Id] = TypeIndex{"target", ti}
		}
		for gi, gitem := range cib.Configuration.Acls.AclGroup {
			mapIdType[gitem.Id] = TypeIndex{"group", gi}
		}
		for ri, ritem := range cib.Configuration.Acls.AclRole {
			mapIdType[ritem.Id] = TypeIndex{"role", ri}
		}

		val, ok := mapIdType[aclId]
		if ok {
			cib.Configuration.Acls.URLType = val.Type
			cib.Configuration.Acls.URLIndex = val.Index
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
