package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

func handleApiCluster(version string, w http.ResponseWriter, r *http.Request, cib_data string) bool {
	m := map[string]func(http.ResponseWriter, *http.Request, string) bool{
		"v1": handleApiClusterV1,
		"v2": handleApiClusterV2,
	}

	return m[version](w, r, cib_data)
}

func handleApiClusterV1(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	// parse xml into Cib struct
	var cib Cib
	err := xml.Unmarshal([]byte(cib_data), &cib)
	if err != nil {
		log.Error(err)
		return false
	}

	cib.Config.Type = "cluster"

	w.Header().Set("Content-Type", "application/json")

	jsonData, jsonError := json.Marshal(&cib)
	if jsonError != nil {
		log.Error(jsonError)
		return false
	}

	io.WriteString(w, string(jsonData)+"\n")
	return true
}

func handleApiClusterV2(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	fmt.Printf("handleApiClusterV2")
	return true
}
