package api

import (
	"sort"
)

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

	simpleCons := make([]SimpleConstraints, 0)
	for key, value := range constraints {
		simpleCons = append(simpleCons, SimpleConstraints{Id: key, Type: value})
	}
	sort.Slice(simpleCons, func(i, j int) bool {
                return simpleCons[i].Id < simpleCons[j].Id
        })

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

	return true, simpleCons
}

// Handle function for api/v1/configuration/locations
func handleConfigLocation(urllist []string, cib *Cib) (bool, interface{}) {
	locationData := cib.Configuration.Constraints.RscLocation
	if locationData == nil {
		return true, nil
	}

	locationId := ""
	if len(urllist) == 5 {
		// api/v1/configuration/locations/:id
		locationId = urllist[4]
		for _, item := range locationData {
			if item.Id == locationId {
				return true, item
			}
		}
	}

	return true, locationData
}

// Handle function for api/v1/configuration/colocations
func handleConfigColocation(urllist []string, cib *Cib) (bool, interface{}) {
	colocationData := cib.Configuration.Constraints.RscColocation
	if colocationData == nil {
		return true, nil
	}

	colocationId := ""
	if len(urllist) == 5 {
		// api/v1/configuration/colocations/:id
		colocationId = urllist[4]
		for _, item := range colocationData {
			if item.Id == colocationId {
				return true, item
			}
		}
	}

	return true, colocationData
}

// Handle function for api/v1/configuration/orders
func handleConfigOrder(urllist []string, cib *Cib) (bool, interface{}) {
	orderData := cib.Configuration.Constraints.RscOrder
	if orderData == nil {
		return true, nil
	}

	orderId := ""
	if len(urllist) == 5 {
		// api/v1/configuration/orders/:id
		orderId = urllist[4]
		for _, item := range orderData {
			if item.Id == orderId {
				return true, item
			}
		}
	}

	return true, orderData
}
