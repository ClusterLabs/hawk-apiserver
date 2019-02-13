package main

type SimpleConstraints struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

// Handle function for api/v1/configuration/constraints
func handleConfigConstraints(urllist []string, cib *Cib) (bool, interface{}) {
	constraints := make(map[string]string)
	for _, item := range cib.Configuration.Constraints.RscLocation {
		constraints[item.Id] = "location"
	}
	for _, item := range cib.Configuration.Constraints.RscColocation {
		constraints[item.Id] = "colocation"
	}
	for _, item := range cib.Configuration.Constraints.RscOrder {
		constraints[item.Id] = "order"
	}

	simple_cons := make([]SimpleConstraints, 0)
	for key, value := range constraints {
		simple_cons = append(simple_cons, SimpleConstraints{Id: key, Type: value})
	}

	if len(urllist) == 5 {
		switch constraints[urllist[4]] {
		case "location":
			return handleConfigLocation(urllist, cib)
		case "colocation":
			return handleConfigColocation(urllist, cib)
		case "order":
			return handleConfigOrder(urllist, cib)
		}
	}

	return true, simple_cons
}

// Handle function for api/v1/configuration/locations
func handleConfigLocation(urllist []string, cib *Cib) (bool, interface{}) {
	location_data := cib.Configuration.Constraints.RscLocation
	if location_data == nil {
		return true, nil
	}

	locationId := ""
	if len(urllist) == 5 {
		// api/v1/configuration/locations/:id
		locationId = urllist[4]
		for _, item := range location_data {
			if item.Id == locationId {
				return true, item
			}
		}
	}

	return true, location_data
}

// Handle function for api/v1/configuration/colocations
func handleConfigColocation(urllist []string, cib *Cib) (bool, interface{}) {
	colocation_data := cib.Configuration.Constraints.RscColocation
	if colocation_data == nil {
		return true, nil
	}

	colocationId := ""
	if len(urllist) == 5 {
		// api/v1/configuration/colocations/:id
		colocationId = urllist[4]
		for _, item := range colocation_data {
			if item.Id == colocationId {
				return true, item
			}
		}
	}

	return true, colocation_data
}

// Handle function for api/v1/configuration/orders
func handleConfigOrder(urllist []string, cib *Cib) (bool, interface{}) {
	order_data := cib.Configuration.Constraints.RscOrder
	if order_data == nil {
		return true, nil
	}

	orderId := ""
	if len(urllist) == 5 {
		// api/v1/configuration/orders/:id
		orderId = urllist[4]
		for _, item := range order_data {
			if item.Id == orderId {
				return true, item
			}
		}
	}

	return true, order_data
}
