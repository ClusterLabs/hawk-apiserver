package api

// SimpleNode maps a CIB node to JSON
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
	s.Attributes = FetchNV(item.InstanceAttributes)
	s.Utilization = FetchNV(item.Utilization)
}

// handle function for url /api/v1/configuration/nodes
func handleConfigNodes(urllist []string, cib *Cib) (bool, interface{}) {
	nodesData := cib.Configuration.Nodes.Node
	if nodesData == nil {
		return true, nil
	}

	nodeId := ""
	if len(urllist) == 5 {
		// /api/v1/configuration/nodes/:id
		nodeId = urllist[4]
	}

	nodes := make([]SimpleNode, 0)
	for _, item := range nodesData {
		simpleItem := &SimpleNode{}
		simpleItem.Instance(item)
		if nodeId == "" {
			// /api/v1/configuration/nodes
			nodes = append(nodes, *simpleItem)
		} else if item.Id == nodeId {
			// /api/v1/configuration/nodes/:id
			return true, simpleItem
		}
	}
	return true, nodes
}

// /api/v1/status/nodes
func handleStateNodes(urllist []string, crmMon *CrmMon) (bool, interface{}) {
	nodesData := crmMon.CrmMonNodes.NodesNode
	if nodesData == nil {
		return true, nil
	}

	nodeId := ""
	if len(urllist) == 5 {
		// /api/v1/status/nodes/:id
		nodeId = urllist[4]
	}

	for index, item := range nodesData {
		if item.Id == nodeId {
			return true, nodesData[index]
		}
	}
	return true, nodesData
}
