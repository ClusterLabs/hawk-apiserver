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
	Nodes   nodes   `xml:"nodes"`
}

type summary struct {
	Nodes     nodesConfigured     `xml:"nodes_configured"`
	Resources resourcesConfigured `xml:"resources_configured"`
}

type nodesConfigured struct {
	Number int `xml:"number,attr"`
}

type resourcesConfigured struct {
	Number   int `xml:"number,attr"`
	Disabled int `xml:"disabled,attr"`
	Blocked  int `xml:"blocked,attr"`
}

type nodes struct {
	Node []node `xml:"node"`
}

type node struct {
	Name             string     `xml:"name,attr"`
	ID               string     `xml:"id,attr"`
	Online           bool       `xml:"online,attr"`
	Standby          bool       `xml:"standby,attr"`
	StandbyOnFail    bool       `xml:"standby_onfail,attr"`
	Maintenance      bool       `xml:"maintenance,attr"`
	Pending          bool       `xml:"pending,attr"`
	Unclean          bool       `xml:"unclean,attr"`
	Shutdown         bool       `xml:"shutdown,attr"`
	ExpectedUp       bool       `xml:"expected_up,attr"`
	DC               bool       `xml:"is_dc,attr"`
	ResourcesRunning int        `xml:"resources_running,attr"`
	Type             string     `xml:"type,attr"`
	Resources        []resource `xml:"resource"`
}

type resource struct {
	ID             string `xml:"id,attr"`
	Agent          string `xml:"resource_agent,attr"`
	Role           string `xml:"role,attr"`
	Active         bool   `xml:"active,attr"`
	Orphaned       bool   `xml:"orphaned,attr"`
	Blocked        bool   `xml:"blocked,attr"`
	Managed        bool   `xml:"managed,attr"`
	Failed         bool   `xml:"failed,attr"`
	FailureIgnored bool   `xml:"failure_ignored,attr"`
	NodesRunningOn int    `xml:"nodes_running_on,attr"`
}

type clusterMetrics struct {
	Node     nodeMetrics
	Resource resourceMetrics
	PerNode  map[string]perNodeMetrics
}

type nodeMetrics struct {
	Total         int
	Online        int
	Standby       int
	StandbyOnFail int
	Maintenance   int
	Pending       int
	Unclean       int
	Shutdown      int
	ExpectedUp    int
	DC            int
	TypeMember    int
	TypePing      int
	TypeRemote    int
	TypeUnknown   int
}

type resourceMetrics struct {
	Total          int
	Disabled       int
	Stopped        int
	Started        int
	Slave          int
	Master         int
	Active         int
	Orphaned       int
	Blocked        int
	Managed        int
	Failed         int
	FailureIgnored int
}

type perNodeMetrics struct {
	ResourcesRunning int
}

func parseMetrics(status *crmMon) *clusterMetrics {
	ret := &clusterMetrics{}

	ret.Node.Total = status.Summary.Nodes.Number
	ret.Resource.Total = status.Summary.Resources.Number
	ret.Resource.Disabled = status.Summary.Resources.Disabled
	ret.PerNode = make(map[string]perNodeMetrics)

	for i := range status.Nodes.Node {
		nod := status.Nodes.Node[i]
		perNode := perNodeMetrics{ResourcesRunning: nod.ResourcesRunning}
		ret.PerNode[nod.Name] = perNode

		if nod.Online {
			ret.Node.Online += 1
		}
		if nod.Standby {
			ret.Node.Standby += 1
		}
		if nod.StandbyOnFail {
			ret.Node.StandbyOnFail += 1
		}
		if nod.Maintenance {
			ret.Node.Maintenance += 1
		}
		if nod.Pending {
			ret.Node.Pending += 1
		}
		if nod.Unclean {
			ret.Node.Unclean += 1
		}
		if nod.Shutdown {
			ret.Node.Shutdown += 1
		}
		if nod.ExpectedUp {
			ret.Node.ExpectedUp += 1
		}
		if nod.DC {
			ret.Node.DC += 1
		}
		if nod.Type == "member" {
			ret.Node.TypeMember += 1
		} else if nod.Type == "ping" {
			ret.Node.TypePing += 1
		} else if nod.Type == "remote" {
			ret.Node.TypeRemote += 1
		} else {
			ret.Node.TypeUnknown += 1
		}

		for j := range nod.Resources {
			rsc := nod.Resources[j]
			if rsc.Role == "Started" {
				ret.Resource.Started += 1
			} else if rsc.Role == "Stopped" {
				ret.Resource.Stopped += 1
			} else if rsc.Role == "Slave" {
				ret.Resource.Slave += 1
			} else if rsc.Role == "Master" {
				ret.Resource.Master += 1
			}
			if rsc.Active {
				ret.Resource.Active += 1
			}
			if rsc.Orphaned {
				ret.Resource.Orphaned += 1
			}
			if rsc.Blocked {
				ret.Resource.Blocked += 1
			}
			if rsc.Managed {
				ret.Resource.Managed += 1
			}
			if rsc.Failed {
				ret.Resource.Failed += 1
			}
			if rsc.FailureIgnored {
				ret.Resource.FailureIgnored += 1
			}
		}
	}

	return ret
}

func handleMetrics(w http.ResponseWriter, r *http.Request) bool {
	monxml, err := exec.Command("/usr/sbin/crm_mon", "-1", "--as-xml", "--group-by-node", "--inactive").Output()
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

	metrics := parseMetrics(&status)

	io.WriteString(w, fmt.Sprintf("cluster_nodes_total %v\n", metrics.Node.Total))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_online %v\n", metrics.Node.Online))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_standby %v\n", metrics.Node.Standby))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_standby_onfail %v\n", metrics.Node.StandbyOnFail))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_maintenance %v\n", metrics.Node.Maintenance))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_pending %v\n", metrics.Node.Pending))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_unclean %v\n", metrics.Node.Unclean))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_shutdown %v\n", metrics.Node.Shutdown))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_expected_up %v\n", metrics.Node.ExpectedUp))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_dc %v\n", metrics.Node.DC))
	io.WriteString(w, fmt.Sprintf("cluster_nodes{type=\"member\"} %v\n", metrics.Node.TypeMember))
	io.WriteString(w, fmt.Sprintf("cluster_nodes{type=\"ping\"} %v\n", metrics.Node.TypePing))
	io.WriteString(w, fmt.Sprintf("cluster_nodes{type=\"remote\"} %v\n", metrics.Node.TypeRemote))
	io.WriteString(w, fmt.Sprintf("cluster_nodes{type=\"unknown\"} %v\n", metrics.Node.TypeUnknown))
	for k := range metrics.PerNode {
		node := metrics.PerNode[k]
		io.WriteString(w, fmt.Sprintf("cluster_resources_running{node=\"%v\"} %v\n", k, node.ResourcesRunning))
	}
	io.WriteString(w, fmt.Sprintf("cluster_resources_total %v\n", metrics.Resource.Total))
	io.WriteString(w, fmt.Sprintf("cluster_resources_disabled %v\n", metrics.Resource.Disabled))
	io.WriteString(w, fmt.Sprintf("cluster_resources{role=\"stopped\"} %v\n", metrics.Resource.Stopped))
	io.WriteString(w, fmt.Sprintf("cluster_resources{role=\"started\"} %v\n", metrics.Resource.Started))
	io.WriteString(w, fmt.Sprintf("cluster_resources{role=\"slave\"} %v\n", metrics.Resource.Slave))
	io.WriteString(w, fmt.Sprintf("cluster_resources{role=\"master\"} %v\n", metrics.Resource.Master))
	io.WriteString(w, fmt.Sprintf("cluster_resources_active %v\n", metrics.Resource.Active))
	io.WriteString(w, fmt.Sprintf("cluster_resources_orphaned %v\n", metrics.Resource.Orphaned))
	io.WriteString(w, fmt.Sprintf("cluster_resources_blocked %v\n", metrics.Resource.Blocked))
	io.WriteString(w, fmt.Sprintf("cluster_resources_managed %v\n", metrics.Resource.Managed))
	io.WriteString(w, fmt.Sprintf("cluster_resources_failed %v\n", metrics.Resource.Failed))
	io.WriteString(w, fmt.Sprintf("cluster_resources_failure_ignored %v\n", metrics.Resource.FailureIgnored))

	return true
}
