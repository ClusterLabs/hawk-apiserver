package main

import (
	"encoding/xml"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

func handleAPICluster(w http.ResponseWriter, r *http.Request, cibData string) bool {
	// parse xml into Cib struct
	var cib Cib
	err := xml.Unmarshal([]byte(cibData), &cib)
	if err != nil {
		log.Error(err)
		return false
	}

	cib.Configuration.URLType = "cluster"

	w.Header().Set("Content-Type", "application/json")

	jsonData, jsonError := MarshalOut(r, &cib)
	if jsonError != nil {
		log.Error(jsonError)
		return false
	}

	io.WriteString(w, string(jsonData)+"\n")
	return true
}
