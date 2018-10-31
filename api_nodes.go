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

func handleAPINodes(w http.ResponseWriter, r *http.Request, cibData string) bool {
	// parse xml into Cib struct
	var cib Cib
	err := xml.Unmarshal([]byte(cibData), &cib)
	if err != nil {
		log.Error(err)
		return false
	}

	cib.Configuration.URLType = "nodes"

	w.Header().Set("Content-Type", "application/json")
	urllist := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(urllist) == 4 {
		// for url api/v1/configuration/nodes
		cib.Configuration.Nodes.URLType = "all"
	} else {
		// for url api/v1/configuration/nodes/{nodeid}
		cib.Configuration.Nodes.URLType = "node"

		nodeIndex := urllist[4]
		index := -1
		for i, item := range cib.Configuration.Nodes.Node {
			if nodeIndex == item.Uname || nodeIndex == item.Id {
				index = i
				break
			}
		}
		if index == -1 {
			http.Error(w, fmt.Sprintf("No route for %v.", r.URL.Path), 500)
			return false
		}

		cib.Configuration.Nodes.URLIndex = index
	}

	jsonData, jsonError := json.Marshal(&cib)
	if jsonError != nil {
		log.Error(jsonError)
		return false
	}

	io.WriteString(w, string(jsonData)+"\n")
	return true
}
