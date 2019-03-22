package api

import (
	"strconv"
	"sort"
)

// SimplePrimitive maps primitive resource to JSON
type SimplePrimitive struct {
	Id       string            `json:"id"`
	Class    string            `json:"class"`
	Type     string            `json:"type"`
	Provider string            `json:"provider,omitempty"`
	Meta     map[string]string `json:"meta,omitempty"`
	Param    map[string]string `json:"param,omitempty"`
	Op       []interface{}     `json:"op,omitempty"`
}

// Instance function for primitive resource
// Casting from Primitive struct to SimplePrimitive struct
// Primitive struct is from cib schema, contain full data about primitive resource
// SimplePrimitive struct contain mainly contents about primitive resource, which are more easy to use/understand
func (s *SimplePrimitive) Instance(item *Primitive) {
	s.Id = item.Id
	s.Class = item.Class
	s.Provider = item.Provider
	s.Type = item.Type
	s.Meta = FetchNV(item.MetaAttributes)
	s.Param = FetchNV(item.InstanceAttributes)

	if item.Operations != nil {
		for _, item := range item.Operations.Op {
			s.Op = append(s.Op, FetchNV2(item))
		}
	}
}

// handle function for url /api/v1/configuration/primitives
func handleConfigPrimitive(urllist []string, cib *Cib) (bool, interface{}) {
	primitivesData := cib.Configuration.Resources.Primitive
	if primitivesData == nil {
		return true, nil
	}

	primitiveId := ""
	if len(urllist) == 5 {
		// /api/v1/configuration/primitives/:id
		primitiveId = urllist[4]
	}

	primitives := make([]SimplePrimitive, 0)
	for _, item := range primitivesData {
		simpleItem := &SimplePrimitive{}
		simpleItem.Instance(item)
		if primitiveId == "" {
			// /api/v1/configuration/primitives
			primitives = append(primitives, *simpleItem)
		} else if item.Id == primitiveId {
			// /api/v1/configuration/primitives/:id
			return true, simpleItem
		}
	}
	return true, primitives
}

// SimpleGroup maps CIB groups to JSON
type SimpleGroup struct {
	Id              string             `json:"id"`
	Meta            map[string]string  `json:"meta,omitempty"`
	SimplePrimitive []*SimplePrimitive `json:"primitives"`
}

// Instance function for group resource
// Casting from Group struct to SimpleGroup struct
func (s *SimpleGroup) Instance(item *Group) {
	s.Id = item.Id
	s.Meta = FetchNV(item.MetaAttributes)

	for _, item := range item.Primitive {
		simpleItem := &SimplePrimitive{}
		simpleItem.Instance(item)
		s.SimplePrimitive = append(s.SimplePrimitive, simpleItem)
	}
}

// handle function for url /api/v1/configuration/groups
func handleConfigGroup(urllist []string, cib *Cib) (bool, interface{}) {
	groupsData := cib.Configuration.Resources.Group
	if groupsData == nil {
		return true, nil
	}

	groupId := ""
	primitiveId := ""
	if len(urllist) == 5 {
		// /api/v1/configuration/groups/:id
		groupId = urllist[4]
	} else if len(urllist) == 6 {
		// /api/v1/configuration/groups/:id/:primitiveId
		groupId, primitiveId = urllist[4], urllist[5]
	}

	groups := make([]SimpleGroup, 0)
	for _, item := range groupsData {
		simpleItem := &SimpleGroup{}
		simpleItem.Instance(item)
		if groupId == "" {
			// /api/v1/configuration/groups
			groups = append(groups, *simpleItem)
		} else if item.Id == groupId {
			if primitiveId == "" {
				// /api/v1/configuration/groups/:id
				return true, simpleItem
			}
			// /api/v1/configuration/groups/:id/:primitiveId
			for index, item := range simpleItem.SimplePrimitive {
				if item.Id == primitiveId {
					return true, simpleItem.SimplePrimitive[index]
				}
			}
		}
	}
	return true, groups
}

// SimpleMaster maps the CIB master tag
type SimpleMaster struct {
	Id              string            `json:"id"`
	Meta            map[string]string `json:"meta,omitempty"`
	SimplePrimitive *SimplePrimitive  `json:"primitive,omitempty"`
	SimpleGroup     *SimpleGroup      `json:"group,omitempty"`
}

// Instance function for master resource
// Casting from Master struct to SimpleMaster struct
func (s *SimpleMaster) Instance(item *Master) {
	s.Id = item.Id
	s.Meta = FetchNV(item.MetaAttributes)

	if item.Primitive != nil {
		simpleItem := &SimplePrimitive{}
		simpleItem.Instance(item.Primitive)
		s.SimplePrimitive = simpleItem
	} else if item.Group != nil {
		simpleItem := &SimpleGroup{}
		simpleItem.Instance(item.Group)
		s.SimpleGroup = simpleItem
	}
}

// handle function for url /api/v1/configuration/masters
func handleConfigMaster(urllist []string, cib *Cib) (bool, interface{}) {
	mastersData := cib.Configuration.Resources.Master
	if mastersData == nil {
		return true, nil
	}

	masterId := ""
	if len(urllist) == 5 {
		// /api/v1/configuration/masters/:id
		masterId = urllist[4]
	}

	masters := make([]SimpleMaster, 0)
	for _, item := range mastersData {
		simpleItem := &SimpleMaster{}
		simpleItem.Instance(item)
		if masterId == "" {
			// /api/v1/configuration/masters
			masters = append(masters, *simpleItem)
		} else if item.Id == masterId {
			// /api/v1/configuration/masters/:id
			return true, simpleItem
		}
	}
	return true, masters
}

// SimpleClone maps CIB clones to JSON
type SimpleClone struct {
	Id              string            `json:"id"`
	Meta            map[string]string `json:"meta,omitempty"`
	SimplePrimitive *SimplePrimitive  `json:"primitive,omitempty"`
	SimpleGroup     *SimpleGroup      `json:"group,omitempty"`
}

// Instance function for clone resource
// Casting from Clone struct to SimpleClone struct
func (s *SimpleClone) Instance(item *Clone) {
	s.Id = item.Id
	s.Meta = FetchNV(item.MetaAttributes)

	if item.Primitive != nil {
		simpleItem := &SimplePrimitive{}
		simpleItem.Instance(item.Primitive)
		s.SimplePrimitive = simpleItem
	} else if item.Group != nil {
		simpleItem := &SimpleGroup{}
		simpleItem.Instance(item.Group)
		s.SimpleGroup = simpleItem
	}
}

// handle function for url /api/v1/configuration/clones
func handleConfigClone(urllist []string, cib *Cib) (bool, interface{}) {
	clonesData := cib.Configuration.Resources.Clone
	if clonesData == nil {
		return true, nil
	}

	cloneId := ""
	if len(urllist) == 5 {
		// /api/v1/configuration/clones/:id
		cloneId = urllist[4]
	}

	clones := make([]SimpleClone, 0)
	for _, item := range clonesData {
		simpleItem := &SimpleClone{}
		simpleItem.Instance(item)
		if cloneId == "" {
			// /api/v1/configuration/clones
			clones = append(clones, *simpleItem)
		} else if item.Id == cloneId {
			// /api/v1/configuration/clones/:id
			return true, simpleItem
		}
	}
	return true, clones
}

// SimpleBundle maps CIB bundles to JSON
type SimpleBundle struct {
	Id              string                 `json:"id"`
	Meta            map[string]string      `json:"meta,omitempty"`
	Container       map[string]interface{} `json:"container"`
	SimpleNetwork   map[string]interface{} `json:"network,omitempty"`
	Storage         []interface{}          `json:"storage-mapping,omitempty"`
	SimplePrimitive *SimplePrimitive       `json:"primitive,omitempty"`
}

// Instance function for bundle resource
// Casting from Bundle struct to SimpleBundle struct
func (s *SimpleBundle) Instance(item *Bundle) {
	s.Id = item.Id
	s.Meta = FetchNV(item.MetaAttributes)

	docker := item.Docker
	rkt := item.Rkt
	podman := item.Podman
	if docker != nil {
		s.Container = FetchNV2(docker)
		s.Container["type"] = "docker"
	} else if rkt != nil {
		s.Container = FetchNV2(rkt)
		s.Container["type"] = "rkt"
	} else if podman != nil {
		s.Container = FetchNV2(podman)
		s.Container["type"] = "podman"
	}

	s.SimpleNetwork = FetchNV2(item.Network)

	if item.Storage != nil {
		for _, sItem := range item.Storage.StorageMapping {
			s.Storage = append(s.Storage, FetchNV2(sItem))
		}
	}

	if item.Primitive != nil {
		simpleItem := &SimplePrimitive{}
		simpleItem.Instance(item.Primitive)
		s.SimplePrimitive = simpleItem
	}
}

// handle function for url /api/v1/configuration/bundles
func handleConfigBundle(urllist []string, cib *Cib) (bool, interface{}) {
	bundlesData := cib.Configuration.Resources.Bundle
	if bundlesData == nil {
		return true, nil
	}

	bundleId := ""
	if len(urllist) == 5 {
		// /api/v1/configuration/bundles/:id
		bundleId = urllist[4]
	}

	bundles := make([]SimpleBundle, 0)
	for _, item := range bundlesData {
		simpleItem := &SimpleBundle{}
		simpleItem.Instance(item)
		if bundleId == "" {
			// /api/v1/configuration/bundles
			bundles = append(bundles, *simpleItem)
		} else if item.Id == bundleId {
			// /api/v1/configuration/bundles/:id
			return true, simpleItem
		}
	}

	return true, bundles
}

// SimpleResource maps a CIB resource to JSON
type SimpleResource struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

// handle function for url /api/v1/configuration/resources
func handleConfigResources(urllist []string, cib *Cib) (bool, interface{}) {
	resources := make(map[string]string)
	for _, item := range cib.Configuration.Resources.Primitive {
		resources[item.Id] = "primitive"
	}
	for _, item := range cib.Configuration.Resources.Group {
		resources[item.Id] = "group"
	}
	for _, item := range cib.Configuration.Resources.Clone {
		resources[item.Id] = "clone"
	}
	for _, item := range cib.Configuration.Resources.Master {
		resources[item.Id] = "master"
	}
	for _, item := range cib.Configuration.Resources.Bundle {
		resources[item.Id] = "bundle"
	}
	simpleResources := make([]SimpleResource, 0)
	for key, value := range resources {
		simpleResources = append(simpleResources, SimpleResource{Id: key, Type: value})
	}
	sort.Slice(simpleResources, func(i, j int) bool {
		return simpleResources[i].Id < simpleResources[j].Id
	})

	if len(urllist) == 5 {
		// for url /api/v1/configuration/resources/:id
		switch resources[urllist[4]] {
		case "primitive":
			return handleConfigPrimitive(urllist, cib)
		case "group":
			return handleConfigGroup(urllist, cib)
		case "master":
			return handleConfigMaster(urllist, cib)
		case "clone":
			return handleConfigClone(urllist, cib)
		case "bundle":
			return handleConfigBundle(urllist, cib)
		}
	}

	return true, simpleResources
}

// SimplePrimitiveState maps CIB primitive state data to JSON
type SimplePrimitiveState struct {
	Id    string `json:"id"`
	Type  string `json:"type"`
	Agent string `json:"resource_agent"`
	Role  string `json:"role"`
	Node  string `json:"on_node"`
}

// Instance function for primitive status
// Casting from ResourcesResource struct to SimplePrimitiveState struct
func (s *SimplePrimitiveState) Instance(item *ResourcesResource) {
	s.Id = item.Id
	s.Type = "primitive"
	s.Agent = item.ResourceAgent
	s.Role = item.Role

	nodesRunningOn, _ := strconv.Atoi(item.NodesRunningOn)
	if nodesRunningOn == 1 {
		s.Node = item.ResourceNode[0].Name
	}
}

// SimpleGroupState maps group state to JSON
type SimpleGroupState struct {
	Id                   string                 `json:"id"`
	Type                 string                 `json:"type"`
	SimplePrimitiveState []SimplePrimitiveState `json:"primitives"`
}

// Instance function for group status
// Casting from ResourcesGroup struct to SimpleGroupState struct
func (s *SimpleGroupState) Instance(item *ResourcesGroup) {
	s.Id = item.Id
	s.Type = "group"

	primitives := make([]SimplePrimitiveState, 0)
	for _, item := range item.GroupResource {
		simpleItem := &SimplePrimitiveState{}
		simpleItem.Instance(item)
		primitives = append(primitives, *simpleItem)
	}
	s.SimplePrimitiveState = primitives
}

// SimpleCloneState maps clone state to JSON
type SimpleCloneState struct {
	Id                   string                  `json:"id"`
	Type                 string                  `json:"type"`
	MultiState           string                  `json:"multi_state"`
	SimplePrimitiveState []*SimplePrimitiveState `json:"primitive,omitempty"`
	SimpleGroupState     []*SimpleGroupState     `json:"groups,omitempty"`
}

// Instance function for master/clone status
// Casting from ResourcesClone struct to SimpleCloneState struct
func (s *SimpleCloneState) Instance(item *ResourcesClone) {
	s.Id = item.Id
	s.MultiState = item.MultiState
	switch item.MultiState {
	case "true":
		s.Type = "master"
	case "false":
		s.Type = "clone"
	}

	for _, item := range item.CloneResource {
		simpleItem := &SimplePrimitiveState{}
		simpleItem.Instance(item)
		s.SimplePrimitiveState = append(s.SimplePrimitiveState, simpleItem)
	}

	for _, item := range item.CloneGroup {
		simpleItem := &SimpleGroupState{}
		simpleItem.Instance(item)
		s.SimpleGroupState = append(s.SimpleGroupState, simpleItem)
	}
}

// handle function for url /api/v1/status/resources
func handleStateResources(urllist []string, crmMon *CrmMon) (bool, interface{}) {
	resourcesData := crmMon.CrmMonResources
	if resourcesData == nil {
		return true, nil
	}

	var allStatus []interface{}
	resId := ""
	if len(urllist) == 5 {
		// /api/v1/status/resources/:id
		resId = urllist[4]
	}

	for _, item := range resourcesData.ResourcesResource {
		simpleItem := &SimplePrimitiveState{}
		simpleItem.Instance(item)
		if item.Id == resId {
			return true, simpleItem
		}
		allStatus = append(allStatus, *simpleItem)
	}

	for _, item := range resourcesData.ResourcesGroup {
		simpleItem := &SimpleGroupState{}
		simpleItem.Instance(item)
		if item.Id == resId {
			return true, simpleItem
		}
		allStatus = append(allStatus, *simpleItem)
	}

	for _, item := range resourcesData.ResourcesClone {
		simpleItem := &SimpleCloneState{}
		simpleItem.Instance(item)
		if item.Id == resId {
			return true, simpleItem
		}
		allStatus = append(allStatus, *simpleItem)
	}

	return true, allStatus
}

// handle function for url /api/v1/status/failures
func handleStateFailures(urllist []string, crmMon *CrmMon) (bool, interface{}) {
	failuresData := crmMon.CrmMonFailures.FailuresFailure
	if failuresData == nil {
		return true, nil
	}

	nodeId := ""
	if len(urllist) == 5 {
		// api/v1/status/failures/:node
		var resultsByNode []interface{}
		nodeId = urllist[4]
		for _, item := range failuresData {
			if item.Node == nodeId {
				resultsByNode = append(resultsByNode, item)
			}
		}
		return true, resultsByNode
	}

	return true, failuresData
}
