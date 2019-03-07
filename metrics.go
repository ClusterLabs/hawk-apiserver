package main

import (
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os/exec"
)

type crmMon struct {
	Version string  `xml:"version,attr"`
	Summary summary `xml:"summary"`
}

type summary struct {
	Nodes     nodesConfigured     `xml:"nodes_configured"`
	Resources resourcesConfigured `xml:"resources_configured"`
}

type nodesConfigured struct {
	Number string `xml:"number,attr"`
}

type resourcesConfigured struct {
	Number   string `xml:"number,attr"`
	Disabled string `xml:"disabled,attr"`
	Blocked  string `xml:"blocked,attr"`
}

func handleMetrics(w http.ResponseWriter, r *http.Request, cibData string) bool {
	monxml, err := exec.Command("/usr/sbin/crm_mon", "--as-xml").Output()
	if err != nil {
		log.Error(err)
		return false
	}

	var status crmMon
	err = xml.Unmarshal(monxml, &status)
	if err != nil {
		log.Error(err)
		return false
	}

	io.WriteString(w, fmt.Sprintf("cluster_node_count %v\n", status.Summary.Nodes.Number))
	io.WriteString(w, fmt.Sprintf("cluster_resource_count %v\n", status.Summary.Resources.Number))
	io.WriteString(w, fmt.Sprintf("cluster_resource_count{status=\"disabled\"} %v\n", status.Summary.Resources.Disabled))
	io.WriteString(w, fmt.Sprintf("cluster_resource_count{status=\"blocked\"} %v\n", status.Summary.Resources.Blocked))

	return true
}
