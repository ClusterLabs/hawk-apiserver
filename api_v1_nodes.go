package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type Result struct {
	NodeItem struct {
		InnerXML string `xml:",innerxml"`
	} `xml:"configuration>nodes"`
}

type Nodes struct {
	NodesList []Node `xml:"node" json:"nodes"`
}

type Node struct {
	Id    string `xml:"id,attr" json:"id"`
	Uname string `xml:"uname,attr" json:"uname"`
}

func handleApiNodesV1(w http.ResponseWriter, cib_data string) bool {
	xmlData := []byte(cib_data)
	var result Result
	err := xml.Unmarshal(xmlData, &result)
	if err != nil {
		log.Error(err)
		return false
	}

	nodesXML := []byte(fmt.Sprintf("<nodes>%s</nodes>", result.NodeItem.InnerXML))
	var nodes Nodes
	err = xml.Unmarshal(nodesXML, &nodes)
	if err != nil {
		log.Error(err)
		return false
	}

	w.Header().Set("Content-Type", "application/json")
	jsonData, jsonError := json.Marshal(nodes)
	if jsonError != nil {
		log.Error(jsonError)
		return false
	}
	io.WriteString(w, string(jsonData))
	return true
}
