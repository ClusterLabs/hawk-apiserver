package main


func handleConfigResources(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.Resources.URLType = "all"
	} else {
		resId := urllist[4]

		mapIdType := make(map[string]TypeIndex)
		for pi, pitem := range cib.Configuration.Resources.Primitive {
			mapIdType[pitem.Id] = TypeIndex{"primitive", pi}
		}
		for gi, gitem := range cib.Configuration.Resources.Group {
			mapIdType[gitem.Id] = TypeIndex{"group", gi}
		}
		for ci, citem := range cib.Configuration.Resources.Clone {
			mapIdType[citem.Id] = TypeIndex{"clone", ci}
		}
		for mi, mitem := range cib.Configuration.Resources.Master {
			mapIdType[mitem.Id] = TypeIndex{"master", mi}
		}

		val, ok := mapIdType[resId]
		if ok {
			cib.Configuration.Resources.URLType = val.Type
			cib.Configuration.Resources.URLIndex = val.Index
		} else {
			return false
		}
	}

	return true
}
