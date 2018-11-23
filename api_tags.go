package main

func handleConfigTags(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.Tags.URLType = "all"
	} else {
		cib.Configuration.Tags.URLType = "tag"

		tagIndex := urllist[4]
		var index int = -1
		for i, item := range cib.Configuration.Tags.Tag {
			if tagIndex == item.Id {
				index = i
				break
			}
		}
		if index == -1 {
			return false
		}

		cib.Configuration.Tags.URLIndex = index
	}

	return true
}
