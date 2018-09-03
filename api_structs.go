package main

import (
	"encoding/json"
	"encoding/xml"
)

type Cib struct {
	XMLName xml.Name      `xml:"cib" json:"-"`
	Config  Configuration `xml:"configuration" json:"configuration"`
	Status  Status        `xml:"status"`
}

type Configuration struct {
	XMLName   xml.Name  `xml:"configuration" json:"-"`
	Type      string    `json:"-"`
	Cluster   CrmConfig `xml:"crm_config" json:"crm_config"`
	Nodes     Nodes     `xml:"nodes" json:"nodes"`
	Resources Resources `xml:"resources" json:"resources"`
	Cons      Constraints `xml:"constraints" json:"constraints"`
}

// Cluster Property
type CrmConfig struct {
	XMLName xml.Name `xml:"crm_config" json:"-"`
	Property Property `xml:"cluster_property_set" json:"cluster_property"`
}

// Nodes define begin
// based on https://github.com/ClusterLabs/pacemaker/blob/master/xml/nodes-3.0.rng
type Nodes struct {
	XMLName xml.Name `xml:"nodes" json:"-"`
	Type    string   `json:"-"`
	Index   int      `json:"-"`
	Node    []*Node   `xml:"node" json:"node"`
}

type Node struct {
	XMLName     xml.Name    `xml:"node" json:"-"`
	Id          string      `xml:"id,attr" json:"id"`
	Uname       string      `xml:"uname,attr" json:"uname"`
	Type	    string      `xml:"type,attr" json:"type,omitempty"`
	Description string      `xml:"description,attr" json:"description,omitempty"`
	Score       string      `xml:"score,attr" json:"score,omitempty"`
	Utilization *Utilization `xml:"utilization" json:"utilization,omitempty"`
	Attributes  *Attributes  `xml:"instance_attributes" json:"attributes,omitempty"`
	Status      string      `json:"status"` // from node_state
}
// Nodes define end

// Resources define begin
// based on https://github.com/ClusterLabs/pacemaker/blob/master/xml/resources-3.2.rng
type Resources struct {
	XMLName   xml.Name    `xml:"resources" json:"-"`
	Type      string      `json:"-"`
	Index     int         `json:"-"`
	Primitive []*Primitive `xml:"primitive" json:"primitive"`
	Group     []*Group     `xml:"group" json:"group,omitempty"`
	Clone     []*Clone     `xml:"clone" json:"clone,omitempty"`
	Master    []*Master    `xml:"master" json:"master,omitempty"`
}

type Primitive struct {
	XMLName    xml.Name   `xml:"primitive" json:"-"`
	Id         string     `xml:"id,attr" json:"id"`
	Class      string     `xml:"class,attr" json:"class"`
	Provider   string     `xml:"provider,attr" json:"provider,omitempty"`
	Type       string     `xml:"type,attr" json:"type"`
	Description string    `xml:"description,attr" json:"description,omitempty"`
	Meta       *Meta `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	Attributes *Attributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Operations *Operations `xml:"operations" json:"operations,omitempty"`
	Utilization *Utilization `xml:"utilization" json:"utilization,omitempty"`
}

type Operations struct {
	XMLName xml.Name `xml:"operations" json:"-"`
	Op      []*Op     `xml:"op" json:"op"`
}

type Op struct {
	Id       string `xml:"id,attr" json:"id"`
	Name     string `xml:"name,attr" json:"name"`
	Interval string `xml:"interval,attr" json:"interval,omitempty"`
	Description string `xml:"description,attr" json:"description,omitempty"`
	StartDelay string `xml:"start-delay,attr" json:"start-delay,omitempty"`
	IntervalOrigin string `xml:"interval-origin,attr" json:"interval-origin,omitempty"`
	Timeout  string `xml:"timeout,attr" json:"timeout,omitempty"`
	Enabled  string `xml:"enabled,attr" json:"enabled,omitempty"`
	RecordPending string `xml:"record-pending,attr" json:"record-pending,omitempty"`
	Role     string `xml:"role,attr" json:"role,omitempty"`
	OnFail   string `xml:"on-fail,attr" json:"on-fail,omitempty"`
}

type Group struct {
	XMLName   xml.Name    `xml:"group" json:"-"`
	Id        string      `xml:"id,attr" json:"id"`
	Description string    `xml:"description,attr" json:"description,omitempty"`
	Meta       *Meta `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	Primitive []*Primitive `xml:"primitive" json:"primitive"`
}

type Clone struct {
	XMLName   xml.Name    `xml:"clone" json:"-"`
	Id        string      `xml:"id,attr" json:"id"`
	Description string    `xml:"description,attr" json:"description,omitempty"`
	Meta       *Meta `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	Primitive []*Primitive `xml:"primitive" json:"primitive,omitempty"`
	Group     []*Group     `xml:"group" json:"group,omitempty"`
}

type Master struct {
	XMLName   xml.Name    `xml:"master" json:"-"`
	Id        string      `xml:"id,attr" json:"id"`
	Description string    `xml:"description,attr" json:"description,omitempty"`
	Meta       *Meta `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	Primitive []*Primitive `xml:"primitive" json:"primitive,omitempty"`
	Group     []*Group     `xml:"group" json:"group,omitempty"`
}
// Resources define end

// Constraints deine begin
// based on https://github.com/ClusterLabs/pacemaker/blob/master/xml/constraints-3.0.rng
type Constraints struct {
	XMLName xml.Name `xml:"constraints" json:"-"`
	Type    string   `json:"-"`
	Index   int      `json:"-"`
	Location []*Location `xml:"rsc_location" json:"location,omitempty"`
	//Colocation *Colocation `xml:"colocation" json:"colocation,omitempty"`
	//Order *Order `xml:"order" json:"order,omitempty"`
	//RscTicket *RscTicket `xml:"rsc_ticket" json:"rsc_ticket,omitempty"`
}

type Location struct {
	XMLName xml.Name `xml:"rsc_location" json:"-"`
	Id          string      `xml:"id,attr" json:"id"`
	Rsc         string      `xml:"rsc,attr" json:"rsc,omitempty"`
	RscPattern  string      `xml:"rsc-pattern,attr" json:"rsc-pattern,omitempty"`
	Role        string      `xml:"role,attr" json:"role,omitempty"`
	//RscSet      []*RscSet   `xml:"resource-set" json:"resource-set,omitempty"`
	Score       string      `xml:"score,attr" json:"score,omitempty"`
	Node        string      `xml:"node,attr" json:"node,omitempty"`
	//Rule
	Discovery   string      `xml:"discovery,attr" json:"discovery,omitempty"`
}
// Constraints deine end

type Property struct {
	XMLName xml.Name `xml:"cluster_property_set" json:"-"`
	Nvpairs []*Nvpair `xml:"nvpair" json:"nvpair"`
}

type Utilization struct {
	XMLName xml.Name `xml:"utilization" json:"-"`
	Nvpairs []*Nvpair `xml:"nvpair" json:"nvpair"`
}

type Attributes struct {
	XMLName xml.Name `xml:"instance_attributes" json:"-"`
	Nvpairs []*Nvpair `xml:"nvpair" json:"nvpair"`
}

type Meta struct {
	XMLName xml.Name `xml:"meta_attributes" json:"-"`
	Nvpairs []*Nvpair `xml:"nvpair" json:"nvpair"`
}

type Nvpair struct {
	Name  string `xml:"name,attr" json:"name"`
	Value string `xml:"value,attr" json:"value"`
}

type Status struct {
	XMLName   xml.Name    `xml:"status"`
	NodeState []*NodeState `xml:"node_state"`
}

type NodeState struct {
	XMLName xml.Name `xml:"node_state"`
	Id      string   `xml:"id,attr"`
	Uname   string   `xml:"uname,attr"`
	Crmd    string   `xml:"crmd,attr"` // online or offline
	LrmRs   []*LrmRs  `xml:"lrm>lrm_resources>lrm_resource"`
}

type LrmRs struct {
	XMLName xml.Name `xml:"lrm_resource"`
	LrmOp   []*LrmOp `xml:"lrm_rsc_op"`
}

type LrmOp struct {
	XMLName xml.Name `xml:"lrm_rsc_op"`
	Id      string   `xml:"id,attr"`
	Operation string `xml:"operation,attr"`
	ExitReason string `xml:"exit-reason,attr"`
	OnNode    string `xml:"on_node,attr"`
	RcCode    string `xml:"rc-code,attr"`
}

type TypeIndex struct {
	Type  string
	Index int
}

func (c *Cib) MarshalJSON() ([]byte, error) {
	var jsonValue []byte
	var err error

	switch c.Config.Type {
	case "nodes":
		switch c.Config.Nodes.Type {
		case "all":
			jsonValue, err = json.Marshal(c.Config.Nodes.Node)
		case "node":
			index := c.Config.Nodes.Index
			jsonValue, err = json.Marshal(c.Config.Nodes.Node[index])
		}
	case "resources":
		switch c.Config.Resources.Type {
		case "all":
			jsonValue, err = json.Marshal(c.Config.Resources)
		case "primitive":
			index := c.Config.Resources.Index
			jsonValue, err = json.Marshal(c.Config.Resources.Primitive[index])
		case "group":
			index := c.Config.Resources.Index
			jsonValue, err = json.Marshal(c.Config.Resources.Group[index])
		case "clone":
			index := c.Config.Resources.Index
			jsonValue, err = json.Marshal(c.Config.Resources.Clone[index])
		case "master":
			index := c.Config.Resources.Index
			jsonValue, err = json.Marshal(c.Config.Resources.Master[index])
		}
	case "cluster":
		jsonValue, err = json.Marshal(c.Config.Cluster)
	case "constraints":
		switch c.Config.Cons.Type {
		case "all":
			jsonValue, err = json.Marshal(c.Config.Cons)
		case "location":
			index := c.Config.Cons.Index
			jsonValue, err = json.Marshal(c.Config.Cons.Location[index])
		}
	}

	return jsonValue, err
}
