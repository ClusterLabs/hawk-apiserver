package main

func handleConfigAcls(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.Acls.URLType = "all"
	} else {
		aclId := urllist[4]

		mapIdType := make(map[string]TypeIndex)
		for ti, titem := range cib.Configuration.Acls.AclTarget {
			mapIdType[titem.Id] = TypeIndex{"target", ti}
		}
		for gi, gitem := range cib.Configuration.Acls.AclGroup {
			mapIdType[gitem.Id] = TypeIndex{"group", gi}
		}
		for ri, ritem := range cib.Configuration.Acls.AclRole {
			mapIdType[ritem.Id] = TypeIndex{"role", ri}
		}

		val, ok := mapIdType[aclId]
		if ok {
			cib.Configuration.Acls.URLType = val.Type
			cib.Configuration.Acls.URLIndex = val.Index
		} else {
			return false
		}
	}

	return true
}
