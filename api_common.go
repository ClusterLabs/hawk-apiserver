package main

import (
	"encoding/json"
	"net/http"
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
		structInterface = c.Configuration.CrmConfig
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
