package main

func handleConfigAlerts(urllist []string, cib Cib) bool {

	if len(urllist) == 4 {
		cib.Configuration.Alerts.URLType = "all"
	} else {
		cib.Configuration.Alerts.URLType = "alert"

		alertIndex := urllist[4]
		var index int = -1
		for i, item := range cib.Configuration.Alerts.Alert {
			if alertIndex == item.Id {
				index = i
				break
			}
		}
		if index == -1 {
			return false
		}

		cib.Configuration.Alerts.URLIndex = index
	}

	return true
}
