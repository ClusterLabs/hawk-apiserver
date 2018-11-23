package main

func handleConfigResources(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.Resources.URLType = "all"
	} else {
		resID := urllist[4]

		mapIDType := make(map[string]TypeIndex)
		for pi, pitem := range cib.Configuration.Resources.Primitive {
			mapIDType[pitem.Id] = TypeIndex{"primitive", pi}
		}
		for gi, gitem := range cib.Configuration.Resources.Group {
			mapIDType[gitem.Id] = TypeIndex{"group", gi}
		}
		for ci, citem := range cib.Configuration.Resources.Clone {
			mapIDType[citem.Id] = TypeIndex{"clone", ci}
		}
		for mi, mitem := range cib.Configuration.Resources.Master {
			mapIDType[mitem.Id] = TypeIndex{"master", mi}
		}

		val, ok := mapIDType[resID]
		if ok {
			cib.Configuration.Resources.URLType = val.Type
			cib.Configuration.Resources.URLIndex = val.Index
		} else {
			return false
		}
	}

	return true
}

func handleStateResources(urllist []string, cib Cib) bool {
	return true
}
