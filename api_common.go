package main

import "encoding/json"

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
	}

	jsonValue, err := json.Marshal(structInterface)
	return jsonValue, err
}
