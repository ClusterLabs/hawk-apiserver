package main

func handleConfigFencing(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.FencingTopology.URLType = "all"
	} else {
		cib.Configuration.FencingTopology.URLType = "fence"

		fenceIndex := urllist[4]
		var index int = -1
		for i, item := range cib.Configuration.FencingTopology.FencingLevel {
			if fenceIndex == item.Id {
				index = i
				break
			}
		}
		if index == -1 {
			return false
		}

		cib.Configuration.FencingTopology.URLIndex = index
	}

	return true
}
