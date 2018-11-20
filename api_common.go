package main

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
        "fmt"
        log "github.com/sirupsen/logrus"
        "io"
        "strings"
)

//go:generate bash gen.sh

// MarshalJSON returns a JSON document representing the CIB.
func (c *Cib) MarshalJSON() ([]byte, error) {
	var structInterface interface{}

	switch c.Configuration.URLType {
	case "nodes":
		switch c.Configuration.Nodes.URLType {
		case "all":
			structInterface = c.Configuration.Nodes.Node
		case "node":
			index := c.Configuration.Nodes.URLIndex
			structInterface = c.Configuration.Nodes.Node[index]
		}
	case "cluster":
		switch c.Configuration.CrmConfig.URLType {
		case "all":
			structInterface = c.Configuration.CrmConfig
		case "property":
			index_bootstrap := c.Configuration.CrmConfig.URLIndex
			index_property := c.Configuration.CrmConfig.ClusterPropertySet[index_bootstrap].URLIndex
			structInterface = c.Configuration.CrmConfig.ClusterPropertySet[index_bootstrap].Nvpair[index_property]
		}
	case "resources":
		switch c.Configuration.Resources.URLType {
		case "all":
			structInterface = c.Configuration.Resources
		case "primitive":
			index := c.Configuration.Resources.URLIndex
			structInterface = c.Configuration.Resources.Primitive[index]
		case "group":
			index := c.Configuration.Resources.URLIndex
			structInterface = c.Configuration.Resources.Group[index]
		case "clone":
			index := c.Configuration.Resources.URLIndex
			structInterface = c.Configuration.Resources.Clone[index]
		case "master":
			index := c.Configuration.Resources.URLIndex
			structInterface = c.Configuration.Resources.Master[index]
		}
	case "constraints":
		switch c.Configuration.Constraints.URLType {
		case "all":
			structInterface = c.Configuration.Constraints
		case "location":
			index := c.Configuration.Constraints.URLIndex
			structInterface = c.Configuration.Constraints.RscLocation[index]
		case "colocation":
			index := c.Configuration.Constraints.URLIndex
			structInterface = c.Configuration.Constraints.RscColocation[index]
		case "order":
			index := c.Configuration.Constraints.URLIndex
			structInterface = c.Configuration.Constraints.RscOrder[index]
		}
	case "rsc_defaults":
		switch c.Configuration.RscDefaults.URLType {
		case "all":
			struct_interface = c.Configuration.RscDefaults
		case "options":
			index_option := c.Configuration.RscDefaults.URLIndex
			index_attr := c.Configuration.RscDefaults.MetaAttributes[index_option].URLIndex
			struct_interface = c.Configuration.RscDefaults.MetaAttributes[index_option].Nvpair[index_attr]
		}
	case "op_defaults":
		switch c.Configuration.OpDefaults.URLType {
		case "all":
			struct_interface = c.Configuration.OpDefaults
		case "options":
			index_option := c.Configuration.OpDefaults.URLIndex
			index_attr := c.Configuration.OpDefaults.MetaAttributes[index_option].URLIndex
			struct_interface = c.Configuration.OpDefaults.MetaAttributes[index_option].Nvpair[index_attr]
		}
	case "alerts":
		switch c.Configuration.Alerts.URLType {
		case "all":
			struct_interface = c.Configuration.Alerts.Alert
		case "alert":
			index := c.Configuration.Alerts.URLIndex
			struct_interface = c.Configuration.Alerts.Alert[index]
		}
	case "tags":
		switch c.Configuration.Tags.URLType {
		case "all":
			struct_interface = c.Configuration.Tags.Tag
		case "tag":
			index := c.Configuration.Tags.URLIndex
			struct_interface = c.Configuration.Tags.Tag[index]
		}
	case "acls":
		switch c.Configuration.Acls.URLType {
		case "all":
			struct_interface = c.Configuration.Acls
		case "target":
			index := c.Configuration.Acls.URLIndex
			struct_interface = c.Configuration.Acls.AclTarget[index]
		case "group":
			index := c.Configuration.Acls.URLIndex
			struct_interface = c.Configuration.Acls.AclGroup[index]
		case "role":
			index := c.Configuration.Acls.URLIndex
			struct_interface = c.Configuration.Acls.AclRole[index]
		}
	case "fencing":
		switch c.Configuration.FencingTopology.URLType {
		case "all":
			struct_interface = c.Configuration.FencingTopology.FencingLevel
		case "fence":
			index := c.Configuration.FencingTopology.URLIndex
			struct_interface = c.Configuration.FencingTopology.FencingLevel[index]
		}
	}

	jsonValue, err := json.Marshal(structInterface)
	return jsonValue, err
}

// Common function for pretty print.
// Give pretty print by default;
// Give nomal print for efficiency reason,
// by setting request header "PrettyPrint" as non "1" value on client.
func MarshalOut(r *http.Request, cib_data *Cib) ([]byte, error) {
	value := r.Header.Get("PrettyPrint")
	if value == "" || value == "1" {
		return json.MarshalIndent(&cib_data, "", "  ")
	}
	return json.Marshal(&cib_data)
}


func handleConfigApi(w http.ResponseWriter, r *http.Request, cib_data string) bool{
	// parse xml into Cib struct
        var cib Cib
        err := xml.Unmarshal([]byte(cib_data), &cib)
        if err != nil {
                log.Error(err)
                return false
        }

        urllist := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
        cib.Configuration.URLType = urllist[3]

        w.Header().Set("Content-Type", "application/json")

        configHandle := map[string]func([]string, Cib) bool {
		"nodes": handleConfigNodes,
		"resources": handleConfigResources,
		"cluster": handleConfigCluster,
		"constraints": handleConfigConstraints,
		"rsc_defaults": handleConfigRscDefaults,
		"op_defaults": handleConfigOpDefaults,
		"alerts": handleConfigAlerts,
		"tags": handleConfigTags,
		"acls": handleConfigAcls,
		"fencing": handleConfigFencing,
        }

	if !configHandle[cib.Configuration.URLType](urllist, cib){
		http.Error(w, fmt.Sprintf("No route for %v.", r.URL.Path), 500)
		return false
	}

        jsonData, jsonError := MarshalOut(r, &cib)
        if jsonError != nil {
                log.Error(jsonError)
                return false
        }

        io.WriteString(w, string(jsonData)+"\n")
        return true
}
