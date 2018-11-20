package main


func handleConfigCluster(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.CrmConfig.URLType = "all"
	} else {
		cib.Configuration.CrmConfig.URLType = "property"

		attrIndex := urllist[4]
		var index int = -1
		var boot_index int

		// find cib-bootstrap-options firstly
		// then match the specific property
		for i, item := range cib.Configuration.CrmConfig.ClusterPropertySet {
			if item.Id == "cib-bootstrap-options"{
				boot_index = i
				for nv_i, nv_item := range item.Nvpair {
					if attrIndex == nv_item.Id || attrIndex == nv_item.Name {
						index = nv_i
						break
					}
				}
				break
			}
		}

		if index == -1 {
			return false
		}

		cib.Configuration.CrmConfig.URLIndex = boot_index
		cib.Configuration.CrmConfig.ClusterPropertySet[boot_index].URLIndex = index
	}

	return true
}


func handleConfigRscDefaults(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.RscDefaults.URLType = "all"
	} else {
		cib.Configuration.RscDefaults.URLType = "options"

		attrIndex := urllist[4]
		var index int = -1
		var option_index int

		// find rsc-options firstly
		// then match the specific options
		for i, item := range cib.Configuration.RscDefaults.MetaAttributes {
			if item.Id == "rsc-options"{
				option_index = i
				for nv_i, nv_item := range item.Nvpair {
					if attrIndex == nv_item.Id || attrIndex == nv_item.Name {
						index = nv_i
						break
					}
				}
				break
			}
		}

		if index == -1 {
			return false
		}

		cib.Configuration.RscDefaults.URLIndex = option_index
		cib.Configuration.RscDefaults.MetaAttributes[option_index].URLIndex = index
	}

	return true
}


func handleConfigOpDefaults(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.OpDefaults.URLType = "all"
	} else {
		cib.Configuration.OpDefaults.URLType = "options"

		attrIndex := urllist[4]
		var index int = -1
		var option_index int

		// find op-options firstly
		// then match the specific options
		for i, item := range cib.Configuration.OpDefaults.MetaAttributes {
			if item.Id == "op-options"{
				option_index = i
				for nv_i, nv_item := range item.Nvpair {
					if attrIndex == nv_item.Id || attrIndex == nv_item.Name {
						index = nv_i
						break
					}
				}
				break
			}
		}

		if index == -1 {
			return false
		}

		cib.Configuration.OpDefaults.URLIndex = option_index
		cib.Configuration.OpDefaults.MetaAttributes[option_index].URLIndex = index
	}

	return true
}
