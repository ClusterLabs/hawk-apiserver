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


type Cib struct {
        XMLName xml.Name `xml:"cib" json:"-"`
        Config Configuration `xml:"configuration" json:"configuration"`
	Status Status `xml:"status"`
}

type Configuration struct {
        XMLName xml.Name `xml:"configuration" json:"-"`
        Nodes Nodes `xml:"nodes" json:"nodes"`
}

type Nodes struct {
        XMLName xml.Name `xml:"nodes" json:"-"`
	Node []Node `xml:"node" json:"node"`
}

type Node struct {
        XMLName xml.Name `xml:"node" json:"-"`
        Id string `xml:"id,attr" json:"id"`
        Uname string `xml:"uname,attr" json:"uname"`
        Utilization Utilization `xml:"utilization" json:"utilization"`
        Attributes  Attributes  `xml:"instance_attributes" json:"attributes"`
	Status string `json:"status"`
}

type Utilization struct {
        XMLName xml.Name `xml:"utilization" json:"-"`
        Nvpairs []Nvpair `xml:"nvpair" json:"nvpair"`
}

type Attributes struct {
        XMLName xml.Name `xml:"instance_attributes" json:"-"`
        Nvpairs []Nvpair `xml:"nvpair" json:"nvpair"`
}

type Nvpair struct {
        Name string `xml:"name,attr" json:"name"`
        Value string `xml:"value,attr" json:"value"`
}

type Status struct {
	XMLName xml.Name `xml:"status"`
	NodeState []NodeState `xml:"node_state"`
}

type NodeState struct {
	XMLName xml.Name `xml:"node_state"`
	Id string `xml:"id,attr"`
	Uname string `xml:"uname,attr"`
        Crmd string `xml:"crmd,attr"` // online or offline
}


func handleApiNodes(version string, w http.ResponseWriter, r *http.Request, cib_data string) bool {
	m := map[string]func(http.ResponseWriter, *http.Request, string)bool{
		"v1": handleApiNodesV1,
		"v2": handleApiNodesV2,
	}

	return m[version](w, r, cib_data)
}

func handleApiNodesV1(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	// parse xml into Cib struct
	xmlData := []byte(cib_data)
	var cib Cib
	err := xml.Unmarshal(xmlData, &cib)
	if err != nil {
		log.Error(err)
		return false
	}

	// assign node status
	state := make(map[string]string)
	for _, stateitem := range cib.Status.NodeState {
		state[stateitem.Uname] = stateitem.Crmd
	}

	for i, nodeitem := range cib.Config.Nodes.Node {
		cib.Config.Nodes.Node[i].Status = state[nodeitem.Uname]
	}

	w.Header().Set("Content-Type", "application/json")

	urllist := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
        if len(urllist) == 3 {
		// for url api/v[1-9]/nodes
		jsonData, jsonError := json.Marshal(cib.Config.Nodes)
		if jsonError != nil {
			log.Error(jsonError)
			return false
		}
		io.WriteString(w, string(jsonData))
	} else {
		// for url api/v[1-9]/nodes/{nodeid}
		nodeIndex := urllist[3]
		var index int = -1
		for i, item := range cib.Config.Nodes.Node {
			if nodeIndex == item.Uname || nodeIndex == item.Id {
				index = i
				break
			}
		}
		if index == -1 {
			http.Error(w, fmt.Sprintf("No route for %v.", r.URL.Path), 500)
			return false
		}

		jsonData, jsonError := json.Marshal(cib.Config.Nodes.Node[index])
		if jsonError != nil {
			log.Error(jsonError)
			return false
		}
		io.WriteString(w, string(jsonData))
	}
	return true
}

func handleApiNodesV2(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	fmt.Printf("handleApiNodesV2")
	return true
}
