package main

import (
	"strconv"
)

// Struct for primitive resource
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
	s.Meta = FetchNv(item.MetaAttributes)
	s.Param = FetchNv(item.InstanceAttributes)

	if item.Operations != nil {
		for _, item := range item.Operations.Op {
			s.Op = append(s.Op, FetchNv2(item))
		}
	}
}

// handle function for url /api/v1/configuration/primitives
func handleConfigPrimitive(urllist []string, cib *Cib) (bool, interface{}) {
	primitives_data := cib.Configuration.Resources.Primitive
	if primitives_data == nil {
		return true, nil
	}

	primitiveId := ""
	if len(urllist) == 5 {
		// /api/v1/configuration/primitives/:id
		primitiveId = urllist[4]
	}

	primitives := make([]SimplePrimitive, 0)
	for _, item := range primitives_data {
		simple_item := &SimplePrimitive{}
		simple_item.Instance(item)
		if primitiveId == "" {
			// /api/v1/configuration/primitives
			primitives = append(primitives, *simple_item)
		} else if item.Id == primitiveId {
			// /api/v1/configuration/primitives/:id
			return true, simple_item
		}
	}
	return true, primitives
}

// Struct for group resource
type SimpleGroup struct {
	Id              string             `json:"id"`
	Meta            map[string]string  `json:"meta,omitempty"`
	SimplePrimitive []*SimplePrimitive `json:"primitives"`
}

// Instance function for group resource
// Casting from Group struct to SimpleGroup struct
func (s *SimpleGroup) Instance(item *Group) {
	s.Id = item.Id
	s.Meta = FetchNv(item.MetaAttributes)

	for _, item := range item.Primitive {
		simple_item := &SimplePrimitive{}
		simple_item.Instance(item)
		s.SimplePrimitive = append(s.SimplePrimitive, simple_item)
	}
}

// handle function for url /api/v1/configuration/groups
func handleConfigGroup(urllist []string, cib *Cib) (bool, interface{}) {
	groups_data := cib.Configuration.Resources.Group
	if groups_data == nil {
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
	for _, item := range groups_data {
		simple_item := &SimpleGroup{}
		simple_item.Instance(item)
		if groupId == "" {
			// /api/v1/configuration/groups
			groups = append(groups, *simple_item)
		} else if item.Id == groupId {
			if primitiveId == "" {
				// /api/v1/configuration/groups/:id
				return true, simple_item
			} else {
				// /api/v1/configuration/groups/:id/:primitiveId
				for index, item := range simple_item.SimplePrimitive {
					if item.Id == primitiveId {
						return true, simple_item.SimplePrimitive[index]
					}
				}
			}
		}
	}
	return true, groups
}

// Struct for master resource
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
	s.Meta = FetchNv(item.MetaAttributes)

	if item.Primitive != nil {
		simple_item := &SimplePrimitive{}
		simple_item.Instance(item.Primitive)
		s.SimplePrimitive = simple_item
	} else if item.Group != nil {
		simple_item := &SimpleGroup{}
		simple_item.Instance(item.Group)
		s.SimpleGroup = simple_item
	}
}

// handle function for url /api/v1/configuration/masters
func handleConfigMaster(urllist []string, cib *Cib) (bool, interface{}) {
	masters_data := cib.Configuration.Resources.Master
	if masters_data == nil {
		return true, nil
	}

	masterId := ""
	if len(urllist) == 5 {
		// /api/v1/configuration/masters/:id
		masterId = urllist[4]
	}

	masters := make([]SimpleMaster, 0)
	for _, item := range masters_data {
		simple_item := &SimpleMaster{}
		simple_item.Instance(item)
		if masterId == "" {
			// /api/v1/configuration/masters
			masters = append(masters, *simple_item)
		} else if item.Id == masterId {
			// /api/v1/configuration/masters/:id
			return true, simple_item
		}
	}
	return true, masters
}

// Struct for clone resource
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
	s.Meta = FetchNv(item.MetaAttributes)

	if item.Primitive != nil {
		simple_item := &SimplePrimitive{}
		simple_item.Instance(item.Primitive)
		s.SimplePrimitive = simple_item
	} else if item.Group != nil {
		simple_item := &SimpleGroup{}
		simple_item.Instance(item.Group)
		s.SimpleGroup = simple_item
	}
}

// handle function for url /api/v1/configuration/clones
func handleConfigClone(urllist []string, cib *Cib) (bool, interface{}) {
	clones_data := cib.Configuration.Resources.Clone
	if clones_data == nil {
		return true, nil
	}

	cloneId := ""
	if len(urllist) == 5 {
		// /api/v1/configuration/clones/:id
		cloneId = urllist[4]
	}

	clones := make([]SimpleClone, 0)
	for _, item := range clones_data {
		simple_item := &SimpleClone{}
		simple_item.Instance(item)
		if cloneId == "" {
			// /api/v1/configuration/clones
			clones = append(clones, *simple_item)
		} else if item.Id == cloneId {
			// /api/v1/configuration/clones/:id
			return true, simple_item
		}
	}
	return true, clones
}

// Struct for bundle resource
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
	s.Meta = FetchNv(item.MetaAttributes)

	docker := item.Docker
	rkt := item.Rkt
	podman := item.Podman
	if docker != nil {
		s.Container = FetchNv2(docker)
		s.Container["type"] = "docker"
	} else if rkt != nil {
		s.Container = FetchNv2(rkt)
		s.Container["type"] = "rkt"
	} else if podman != nil {
		s.Container = FetchNv2(podman)
		s.Container["type"] = "podman"
	}

	s.SimpleNetwork = FetchNv2(item.Network)

	if item.Storage != nil {
		for _, s_item := range item.Storage.StorageMapping {
			s.Storage = append(s.Storage, FetchNv2(s_item))
		}
	}

	if item.Primitive != nil {
		simple_item := &SimplePrimitive{}
		simple_item.Instance(item.Primitive)
		s.SimplePrimitive = simple_item
	}
}

// handle function for url /api/v1/configuration/bundles
func handleConfigBundle(urllist []string, cib *Cib) (bool, interface{}) {
	bundles_data := cib.Configuration.Resources.Bundle
	if bundles_data == nil {
		return true, nil
	}

	bundleId := ""
	if len(urllist) == 5 {
		// /api/v1/configuration/bundles/:id
		bundleId = urllist[4]
	}

	bundles := make([]SimpleBundle, 0)
	for _, item := range bundles_data {
		simple_item := &SimpleBundle{}
		simple_item.Instance(item)
		if bundleId == "" {
			// /api/v1/configuration/bundles
			bundles = append(bundles, *simple_item)
		} else if item.Id == bundleId {
			// /api/v1/configuration/bundles/:id
			return true, simple_item
		}
	}

	return true, bundles
}

// Struct for all types resource
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
	simple_resources := make([]SimpleResource, 0)
	for key, value := range resources {
		simple_resources = append(simple_resources, SimpleResource{Id: key, Type: value})
	}

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

	return true, simple_resources
}

// Struct for primitive status
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

	nodes_running_on, _ := strconv.Atoi(item.NodesRunningOn)
	if nodes_running_on == 1 {
		s.Node = item.ResourceNode[0].Name
	}
}

// Struct for group status
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
		simple_item := &SimplePrimitiveState{}
		simple_item.Instance(item)
		primitives = append(primitives, *simple_item)
	}
	s.SimplePrimitiveState = primitives
}

// Struct for master/clone status
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
		simple_item := &SimplePrimitiveState{}
		simple_item.Instance(item)
		s.SimplePrimitiveState = append(s.SimplePrimitiveState, simple_item)
	}

	for _, item := range item.CloneGroup {
		simple_item := &SimpleGroupState{}
		simple_item.Instance(item)
		s.SimpleGroupState = append(s.SimpleGroupState, simple_item)
	}
}

// handle function for url /api/v1/status/resources
func handleStateResources(urllist []string, crmMon *CrmMon) (bool, interface{}) {
	resources_data := crmMon.CrmMonResources
	if resources_data == nil {
		return true, nil
	}

	var allStatus []interface{}
	resId := ""
	if len(urllist) == 5 {
		// /api/v1/status/resources/:id
		resId = urllist[4]
	}

	for _, item := range resources_data.ResourcesResource {
		simple_item := &SimplePrimitiveState{}
		simple_item.Instance(item)
		if item.Id == resId {
			return true, simple_item
		}
		allStatus = append(allStatus, *simple_item)
	}

	for _, item := range resources_data.ResourcesGroup {
		simple_item := &SimpleGroupState{}
		simple_item.Instance(item)
		if item.Id == resId {
			return true, simple_item
		}
		allStatus = append(allStatus, *simple_item)
	}

	for _, item := range resources_data.ResourcesClone {
		simple_item := &SimpleCloneState{}
		simple_item.Instance(item)
		if item.Id == resId {
			return true, simple_item
		}
		allStatus = append(allStatus, *simple_item)
	}

	return true, allStatus
}

// handle function for url /api/v1/status/failures
func handleStateFailures(urllist []string, crmMon *CrmMon) (bool, interface{}) {
	failures_data := crmMon.CrmMonFailures.FailuresFailure
	if failures_data == nil {
		return true, nil
	}

	nodeId := ""
	if len(urllist) == 5 {
		// api/v1/status/failures/:node
		var results_by_node []interface{}
		nodeId = urllist[4]
		for _, item := range failures_data {
			if item.Node == nodeId {
				results_by_node = append(results_by_node, item)
			}
		}
		return true, results_by_node
	}

	return true, failures_data
}
