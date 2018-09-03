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

func handleApiNodes(version string, w http.ResponseWriter, r *http.Request, cib_data string) bool {
	m := map[string]func(http.ResponseWriter, *http.Request, string) bool{
		"v1": handleApiNodesV1,
		"v2": handleApiNodesV2,
	}

	return m[version](w, r, cib_data)
}

func handleApiNodesV1(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	// parse xml into Cib struct
	var cib Cib
	err := xml.Unmarshal([]byte(cib_data), &cib)
	if err != nil {
		log.Error(err)
		return false
	}

	cib.Configuration.URLType = "nodes"

	w.Header().Set("Content-Type", "application/json")
	urllist := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(urllist) == 3 {
		// for url api/v[1-9]/nodes
		cib.Configuration.Nodes.URLType = "all"
	} else {
		// for url api/v[1-9]/nodes/{nodeid}
		cib.Configuration.Nodes.URLType = "node"

		nodeIndex := urllist[3]
		var index int = -1
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

func handleApiNodesV2(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	fmt.Printf("handleApiNodesV2")
	return true
}
