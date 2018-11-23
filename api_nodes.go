package main

func handleConfigNodes(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.Nodes.URLType = "all"
	} else {
		cib.Configuration.Nodes.URLType = "node"

		nodeIndex := urllist[4]
		index := -1
		for i, item := range cib.Configuration.Nodes.Node {
			if nodeIndex == item.Uname || nodeIndex == item.Id {
				index = i
				break
			}
		}
		if index == -1 {
			return false
		}

		cib.Configuration.Nodes.URLIndex = index
	}

	return true
}
