package main

func handleConfigConstraints(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.Constraints.URLType = "all"
	} else {
		consId := urllist[4]

		mapIdType := make(map[string]TypeIndex)
		for li, litem := range cib.Configuration.Constraints.RscLocation {
			mapIdType[litem.Id] = TypeIndex{"location", li}
		}
		for ci, citem := range cib.Configuration.Constraints.RscColocation {
			mapIdType[citem.Id] = TypeIndex{"colocation", ci}
		}
		for oi, oitem := range cib.Configuration.Constraints.RscOrder {
			mapIdType[oitem.Id] = TypeIndex{"order", oi}
		}

		val, ok := mapIdType[consId]
		if ok {
			cib.Configuration.Constraints.URLType = val.Type
			cib.Configuration.Constraints.URLIndex = val.Index
		} else {
			return false
		}
	}

	return true
}
