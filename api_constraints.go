package main

func handleConfigConstraints(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.Constraints.URLType = "all"
	} else {
		consID := urllist[4]

		mapIDType := make(map[string]TypeIndex)
		for li, litem := range cib.Configuration.Constraints.RscLocation {
			mapIDType[litem.Id] = TypeIndex{"location", li}
		}
		for ci, citem := range cib.Configuration.Constraints.RscColocation {
			mapIDType[citem.Id] = TypeIndex{"colocation", ci}
		}
		for oi, oitem := range cib.Configuration.Constraints.RscOrder {
			mapIDType[oitem.Id] = TypeIndex{"order", oi}
		}

		val, ok := mapIDType[consID]
		if ok {
			cib.Configuration.Constraints.URLType = val.Type
			cib.Configuration.Constraints.URLIndex = val.Index
		} else {
			return false
		}
	}

	return true
}
