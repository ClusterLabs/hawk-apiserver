package main

import "encoding/json"

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
	}

	jsonValue, err := json.Marshal(struct_interface)
	return jsonValue, err
}
