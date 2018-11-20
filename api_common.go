package main

import (
	"encoding/json"
	"net/http"
)

//go:generate bash gen.sh

func (c *Cib) MarshalJSON() ([]byte, error) {
	var struct_interface interface{}

	switch c.Configuration.URLType {
	case "nodes":
		switch c.Configuration.Nodes.URLType {
		case "all":
			struct_interface = c.Configuration.Nodes.Node
		case "node":
			index := c.Configuration.Nodes.URLIndex
			struct_interface = c.Configuration.Nodes.Node[index]
		}
	case "cluster":
		struct_interface = c.Configuration.CrmConfig
	case "resources":
		switch c.Configuration.Resources.URLType {
		case "all":
			struct_interface = c.Configuration.Resources
		case "primitive":
			index := c.Configuration.Resources.URLIndex
			struct_interface = c.Configuration.Resources.Primitive[index]
		case "group":
			index := c.Configuration.Resources.URLIndex
			struct_interface = c.Configuration.Resources.Group[index]
		case "clone":
			index := c.Configuration.Resources.URLIndex
			struct_interface = c.Configuration.Resources.Clone[index]
		case "master":
			index := c.Configuration.Resources.URLIndex
			struct_interface = c.Configuration.Resources.Master[index]
		}
	case "constraints":
		switch c.Configuration.Constraints.URLType {
		case "all":
			struct_interface = c.Configuration.Constraints
		case "location":
			index := c.Configuration.Constraints.URLIndex
			struct_interface = c.Configuration.Constraints.RscLocation[index]
		case "colocation":
			index := c.Configuration.Constraints.URLIndex
			struct_interface = c.Configuration.Constraints.RscColocation[index]
		case "order":
			index := c.Configuration.Constraints.URLIndex
			struct_interface = c.Configuration.Constraints.RscOrder[index]
		}
	case "rsc_defaults":
		struct_interface = c.Configuration.RscDefaults
	case "op_defaults":
		struct_interface = c.Configuration.OpDefaults
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
	case "fences":
		switch c.Configuration.FencingTopology.URLType {
		case "all":
			struct_interface = c.Configuration.FencingTopology.FencingLevel
		case "fence":
			index := c.Configuration.FencingTopology.URLIndex
			struct_interface = c.Configuration.FencingTopology.FencingLevel[index]
		}
	}

	jsonValue, err := json.Marshal(struct_interface)
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
