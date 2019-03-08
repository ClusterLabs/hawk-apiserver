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

type nodeMetrics struct {
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

	io.WriteString(w, fmt.Sprintf("cluster_nodes_total %v\n", status.Summary.Nodes.Number))
	io.WriteString(w, fmt.Sprintf("cluster_resources_total %v\n", status.Summary.Resources.Number))
	io.WriteString(w, fmt.Sprintf("cluster_resources_disabled %v\n", status.Summary.Resources.Disabled))
	io.WriteString(w, fmt.Sprintf("cluster_resources_blocked %v\n", status.Summary.Resources.Blocked))

	nodemetrics := nodeMetrics{}
	resourcemetrics := resourceMetrics{}

	for i := range status.Nodes.Node {
		nod := status.Nodes.Node[i]
		io.WriteString(w, fmt.Sprintf("cluster_resources_running{node=\"%v\"} %v\n", nod.Name, nod.ResourcesRunning))
		if nod.Online {
			nodemetrics.Online += 1
		}
		if nod.Standby {
			nodemetrics.Standby += 1
		}
		if nod.StandbyOnFail {
			nodemetrics.StandbyOnFail += 1
		}
		if nod.Maintenance {
			nodemetrics.Maintenance += 1
		}
		if nod.Pending {
			nodemetrics.Pending += 1
		}
		if nod.Unclean {
			nodemetrics.Unclean += 1
		}
		if nod.Shutdown {
			nodemetrics.Shutdown += 1
		}
		if nod.ExpectedUp {
			nodemetrics.ExpectedUp += 1
		}
		if nod.DC {
			nodemetrics.DC += 1
		}
		if nod.Type == "member" {
			nodemetrics.TypeMember += 1
		} else if nod.Type == "ping" {
			nodemetrics.TypePing += 1
		} else if nod.Type == "remote" {
			nodemetrics.TypeRemote += 1
		} else {
			nodemetrics.TypeUnknown += 1
		}

		for j := range nod.Resources {
			rsc := nod.Resources[j]
			if rsc.Role == "Started" {
				resourcemetrics.Started += 1
			} else if rsc.Role == "Stopped" {
				resourcemetrics.Stopped += 1
			} else if rsc.Role == "Slave" {
				resourcemetrics.Slave += 1
			} else if rsc.Role == "Master" {
				resourcemetrics.Master += 1
			}
			if rsc.Active {
				resourcemetrics.Active += 1
			}
			if rsc.Orphaned {
				resourcemetrics.Orphaned += 1
			}
			if rsc.Blocked {
				resourcemetrics.Blocked += 1
			}
			if rsc.Managed {
				resourcemetrics.Managed += 1
			}
			if rsc.Failed {
				resourcemetrics.Failed += 1
			}
			if rsc.FailureIgnored {
				resourcemetrics.FailureIgnored += 1
			}
		}
	}

	io.WriteString(w, fmt.Sprintf("cluster_nodes_online %v\n", nodemetrics.Online))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_standby %v\n", nodemetrics.Standby))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_standby_onfail %v\n", nodemetrics.StandbyOnFail))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_maintenance %v\n", nodemetrics.Maintenance))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_pending %v\n", nodemetrics.Pending))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_unclean %v\n", nodemetrics.Unclean))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_shutdown %v\n", nodemetrics.Shutdown))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_expected_up %v\n", nodemetrics.ExpectedUp))
	io.WriteString(w, fmt.Sprintf("cluster_nodes_dc %v\n", nodemetrics.DC))
	io.WriteString(w, fmt.Sprintf("cluster_nodes{type=\"member\"} %v\n", nodemetrics.TypeMember))
	io.WriteString(w, fmt.Sprintf("cluster_nodes{type=\"ping\"} %v\n", nodemetrics.TypePing))
	io.WriteString(w, fmt.Sprintf("cluster_nodes{type=\"remote\"} %v\n", nodemetrics.TypeRemote))
	io.WriteString(w, fmt.Sprintf("cluster_nodes{type=\"unknown\"} %v\n", nodemetrics.TypeUnknown))
	io.WriteString(w, fmt.Sprintf("cluster_resources{role=\"stopped\"} %v\n", resourcemetrics.Stopped))
	io.WriteString(w, fmt.Sprintf("cluster_resources{role=\"started\"} %v\n", resourcemetrics.Started))
	io.WriteString(w, fmt.Sprintf("cluster_resources{role=\"slave\"} %v\n", resourcemetrics.Slave))
	io.WriteString(w, fmt.Sprintf("cluster_resources{role=\"master\"} %v\n", resourcemetrics.Master))
	io.WriteString(w, fmt.Sprintf("cluster_resources_active %v\n", resourcemetrics.Active))
	io.WriteString(w, fmt.Sprintf("cluster_resources_orphaned %v\n", resourcemetrics.Orphaned))
	io.WriteString(w, fmt.Sprintf("cluster_resources_blocked %v\n", resourcemetrics.Blocked))
	io.WriteString(w, fmt.Sprintf("cluster_resources_managed %v\n", resourcemetrics.Managed))
	io.WriteString(w, fmt.Sprintf("cluster_resources_failed %v\n", resourcemetrics.Failed))
	io.WriteString(w, fmt.Sprintf("cluster_resources_failure_ignored %v\n", resourcemetrics.FailureIgnored))

	return true
}
