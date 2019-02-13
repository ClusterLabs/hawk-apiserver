package main

// Struct for node
type SimpleNode struct {
	Id          string            `json:"id"`
	Uname       string            `json:"uname"`
	Type        string            `json:"type,omitempty"`
	Attributes  map[string]string `json:"instance_attributes,omitempty"`
	Utilization map[string]string `json:"utilization,omitempty"`
}

// Instance function for node
// Casting from Node struct to SimpleNode struct
func (s *SimpleNode) Instance(item *Node) {
	s.Id = item.Id
	s.Uname = item.Uname
	s.Type = item.Type
	s.Attributes = FetchNv(item.InstanceAttributes)
	s.Utilization = FetchNv(item.Utilization)
}

// handle function for url /api/v1/configuration/nodes
func handleConfigNodes(urllist []string, cib *Cib) (bool, interface{}) {
	nodes_data := cib.Configuration.Nodes.Node
	if nodes_data == nil {
		return true, nil
	}

	nodeId := ""
	if len(urllist) == 5 {
		// /api/v1/configuration/nodes/:id
		nodeId = urllist[4]
	}

	nodes := make([]SimpleNode, 0)
	for _, item := range nodes_data {
		simple_item := &SimpleNode{}
		simple_item.Instance(item)
		if nodeId == "" {
			// /api/v1/configuration/nodes
			nodes = append(nodes, *simple_item)
		} else if item.Id == nodeId {
			// /api/v1/configuration/nodes/:id
			return true, simple_item
		}
	}
	return true, nodes
}

// /api/v1/status/nodes
func handleStateNodes(urllist []string, crmMon *CrmMon) (bool, interface{}) {
	nodes_data := crmMon.CrmMonNodes.NodesNode
	if nodes_data == nil {
		return true, nil
	}

	nodeId := ""
	if len(urllist) == 5 {
		// /api/v1/status/nodes/:id
		nodeId = urllist[4]
	}

	for index, item := range nodes_data {
		if item.Id == nodeId {
			return true, nodes_data[index]
		}
	}
	return true, nodes_data
}
