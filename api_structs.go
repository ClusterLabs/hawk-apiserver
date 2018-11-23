package main

import (
	"encoding/xml"
)

type Cib struct {
	XMLNAME         xml.Name       `xml:"cib" json:"-"`
	ValidateWith    string         `xml:"validate-with,attr" json:"validate-with,omitempty"`
	AdminEpoch      string         `xml:"admin_epoch,attr" json:"admin_epoch"`
	Epoch           string         `xml:"epoch,attr" json:"epoch"`
	NumUpdates      string         `xml:"num_updates,attr" json:"num_updates"`
	CrmFeatureSet   string         `xml:"crm_feature_set,attr" json:"crm_feature_set,omitempty"`
	RemoteTlsPort   string         `xml:"remote-tls-port,attr" json:"remote-tls-port,omitempty"`
	RemoteClearPort string         `xml:"remote-clear-port,attr" json:"remote-clear-port,omitempty"`
	HaveQuorum      string         `xml:"have-quorum,attr" json:"have-quorum,omitempty"`
	DcUuid          string         `xml:"dc-uuid,attr" json:"dc-uuid,omitempty"`
	CibLastWritten  string         `xml:"cib-last-written,attr" json:"cib-last-written,omitempty"`
	NoQuorumPanic   string         `xml:"no-quorum-panic,attr" json:"no-quorum-panic,omitempty"`
	UpdateOrigin    string         `xml:"update-origin,attr" json:"update-origin,omitempty"`
	UpdateClient    string         `xml:"update-client,attr" json:"update-client,omitempty"`
	UpdateUser      string         `xml:"update-user,attr" json:"update-user,omitempty"`
	ExecutionDate   string         `xml:"execution-date,attr" json:"execution-date,omitempty"`
	Configuration   *Configuration `xml:"configuration" json:"configuration"`
	Status          *Status        `xml:"status" json:"status,omitempty"`
}

type Configuration struct {
	XMLNAME         xml.Name         `xml:"configuration" json:"-"`
	CrmConfig       *CrmConfig       `xml:"crm_config" json:"crm_config"`
	RscDefaults     *RscDefaults     `xml:"rsc_defaults" json:"rsc_defaults,omitempty"`
	OpDefaults      *OpDefaults      `xml:"op_defaults" json:"op_defaults,omitempty"`
	Nodes           *Nodes           `xml:"nodes" json:"nodes"`
	Resources       *Resources       `xml:"resources" json:"resources"`
	Constraints     *Constraints     `xml:"constraints" json:"constraints"`
	FencingTopology *FencingTopology `xml:"fencing-topology" json:"fencing-topology,omitempty"`
	Acls            *Acls            `xml:"acls" json:"acls,omitempty"`
	Tags            *Tags            `xml:"tags" json:"tags,omitempty"`
	Alerts          *Alerts          `xml:"alerts" json:"alerts,omitempty"`
	URLType         string           `json:"-"`
}

type CrmConfig struct {
	XMLNAME            xml.Name              `xml:"crm_config" json:"-"`
	ClusterPropertySet []*ClusterPropertySet `xml:"cluster_property_set" json:"cluster_property_set,omitempty"`
	URLType            string                `json:"-"`
	URLIndex           int                   `json:"-"`
}

type ClusterPropertySet struct {
	XMLNAME  xml.Name  `xml:"cluster_property_set" json:"-"`
	IdRef    string    `xml:"id-ref,attr" json:"id-ref,omitempty"`
	Id       string    `xml:"id,attr" json:"id,omitempty"`
	Rule     *Rule     `xml:"rule" json:"rule,omitempty"`
	Nvpair   []*Nvpair `xml:"nvpair" json:"nvpair,omitempty"`
	Score    string    `xml:"score,attr" json:"score,omitempty"`
	URLType  string    `json:"-"`
	URLIndex int       `json:"-"`
}

type Rule struct {
	XMLNAME        xml.Name          `xml:"rule" json:"-"`
	IdRef          string            `xml:"id-ref,attr" json:"id-ref,omitempty"`
	Id             string            `xml:"id,attr" json:"id,omitempty"`
	Score          string            `xml:"score,attr" json:"score,omitempty"`
	ScoreAttribute string            `xml:"score-attribute,attr" json:"score-attribute,omitempty"`
	BooleanOp      string            `xml:"boolean-op,attr" json:"boolean-op,omitempty"`
	Role           string            `xml:"role,attr" json:"role,omitempty"`
	Expression     []*Expression     `xml:"expression" json:"expression,omitempty"`
	DateExpression []*DateExpression `xml:"date_expression" json:"date_expression,omitempty"`
	Rule           []*Rule           `xml:"rule" json:"rule,omitempty"`
}

type Expression struct {
	XMLNAME     xml.Name `xml:"expression" json:"-"`
	Id          string   `xml:"id,attr" json:"id"`
	Attribute   string   `xml:"attribute,attr" json:"attribute"`
	Operation   string   `xml:"operation,attr" json:"operation"`
	Value       string   `xml:"value,attr" json:"value,omitempty"`
	Type        string   `xml:"type,attr" json:"type,omitempty"`
	ValueSource string   `xml:"value-source,attr" json:"value-source,omitempty"`
}

type DateExpression struct {
	XMLNAME   xml.Name  `xml:"date_expression" json:"-"`
	Id        string    `xml:"id,attr" json:"id"`
	Operation string    `xml:"operation,attr" json:"operation,omitempty"`
	Start     string    `xml:"start,attr" json:"start,omitempty"`
	End       string    `xml:"end,attr" json:"end,omitempty"`
	Duration  *Duration `xml:"duration" json:"duration,omitempty"`
	DateSpec  *DateSpec `xml:"date_spec" json:"date_spec,omitempty"`
}

type Duration struct {
	XMLNAME   xml.Name `xml:"duration" json:"-"`
	Id        string   `xml:"id,attr" json:"id"`
	Hours     string   `xml:"hours,attr" json:"hours,omitempty"`
	Monthdays string   `xml:"monthdays,attr" json:"monthdays,omitempty"`
	Weekdays  string   `xml:"weekdays,attr" json:"weekdays,omitempty"`
	Yearsdays string   `xml:"yearsdays,attr" json:"yearsdays,omitempty"`
	Months    string   `xml:"months,attr" json:"months,omitempty"`
	Weeks     string   `xml:"weeks,attr" json:"weeks,omitempty"`
	Years     string   `xml:"years,attr" json:"years,omitempty"`
	Weekyears string   `xml:"weekyears,attr" json:"weekyears,omitempty"`
	Moon      string   `xml:"moon,attr" json:"moon,omitempty"`
}

type DateSpec struct {
	XMLNAME   xml.Name `xml:"date_spec" json:"-"`
	Id        string   `xml:"id,attr" json:"id"`
	Hours     string   `xml:"hours,attr" json:"hours,omitempty"`
	Monthdays string   `xml:"monthdays,attr" json:"monthdays,omitempty"`
	Weekdays  string   `xml:"weekdays,attr" json:"weekdays,omitempty"`
	Yearsdays string   `xml:"yearsdays,attr" json:"yearsdays,omitempty"`
	Months    string   `xml:"months,attr" json:"months,omitempty"`
	Weeks     string   `xml:"weeks,attr" json:"weeks,omitempty"`
	Years     string   `xml:"years,attr" json:"years,omitempty"`
	Weekyears string   `xml:"weekyears,attr" json:"weekyears,omitempty"`
	Moon      string   `xml:"moon,attr" json:"moon,omitempty"`
}

type Nvpair struct {
	XMLNAME xml.Name `xml:"nvpair" json:"-"`
	IdRef   string   `xml:"id-ref,attr" json:"id-ref,omitempty"`
	Name    string   `xml:"name,attr" json:"name,omitempty"`
	Id      string   `xml:"id,attr" json:"id,omitempty"`
	Value   string   `xml:"value,attr" json:"value,omitempty"`
}

type RscDefaults struct {
	XMLNAME        xml.Name          `xml:"rsc_defaults" json:"-"`
	MetaAttributes []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	URLType        string            `json:"-"`
	URLIndex       int               `json:"-"`
}

type MetaAttributes struct {
	XMLNAME  xml.Name  `xml:"meta_attributes" json:"-"`
	IdRef    string    `xml:"id-ref,attr" json:"id-ref,omitempty"`
	Id       string    `xml:"id,attr" json:"id,omitempty"`
	Rule     *Rule     `xml:"rule" json:"rule,omitempty"`
	Nvpair   []*Nvpair `xml:"nvpair" json:"nvpair,omitempty"`
	Score    string    `xml:"score,attr" json:"score,omitempty"`
	URLType  string    `json:"-"`
	URLIndex int       `json:"-"`
}

type OpDefaults struct {
	XMLNAME        xml.Name          `xml:"op_defaults" json:"-"`
	MetaAttributes []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	URLType        string            `json:"-"`
	URLIndex       int               `json:"-"`
}

type Nodes struct {
	XMLNAME  xml.Name `xml:"nodes" json:"-"`
	Node     []*Node  `xml:"node" json:"node,omitempty"`
	URLType  string   `json:"-"`
	URLIndex int      `json:"-"`
}

type Node struct {
	XMLNAME            xml.Name              `xml:"node" json:"-"`
	Id                 string                `xml:"id,attr" json:"id"`
	Uname              string                `xml:"uname,attr" json:"uname"`
	Type               string                `xml:"type,attr" json:"type,omitempty"`
	Description        string                `xml:"description,attr" json:"description,omitempty"`
	Score              string                `xml:"score,attr" json:"score,omitempty"`
	InstanceAttributes []*InstanceAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Utilization        []*Utilization        `xml:"utilization" json:"utilization,omitempty"`
}

type InstanceAttributes struct {
	XMLNAME xml.Name  `xml:"instance_attributes" json:"-"`
	IdRef   string    `xml:"id-ref,attr" json:"id-ref,omitempty"`
	Id      string    `xml:"id,attr" json:"id,omitempty"`
	Rule    *Rule     `xml:"rule" json:"rule,omitempty"`
	Nvpair  []*Nvpair `xml:"nvpair" json:"nvpair,omitempty"`
	Score   string    `xml:"score,attr" json:"score,omitempty"`
}

type Utilization struct {
	XMLNAME xml.Name  `xml:"utilization" json:"-"`
	IdRef   string    `xml:"id-ref,attr" json:"id-ref,omitempty"`
	Id      string    `xml:"id,attr" json:"id,omitempty"`
	Rule    *Rule     `xml:"rule" json:"rule,omitempty"`
	Nvpair  []*Nvpair `xml:"nvpair" json:"nvpair,omitempty"`
	Score   string    `xml:"score,attr" json:"score,omitempty"`
}

type Resources struct {
	XMLNAME   xml.Name     `xml:"resources" json:"-"`
	Primitive []*Primitive `xml:"primitive" json:"primitive,omitempty"`
	Template  []*Template  `xml:"template" json:"template,omitempty"`
	Group     []*Group     `xml:"group" json:"group,omitempty"`
	Clone     []*Clone     `xml:"clone" json:"clone,omitempty"`
	Master    []*Master    `xml:"master" json:"master,omitempty"`
	Bundle    []*Bundle    `xml:"bundle" json:"bundle,omitempty"`
	URLType   string       `json:"-"`
	URLIndex  int          `json:"-"`
}

type Primitive struct {
	XMLNAME            xml.Name              `xml:"primitive" json:"-"`
	Id                 string                `xml:"id,attr" json:"id"`
	Class              string                `xml:"class,attr" json:"class,omitempty"`
	Provider           string                `xml:"provider,attr" json:"provider,omitempty"`
	Type               string                `xml:"type,attr" json:"type,omitempty"`
	Template           string                `xml:"template,attr" json:"template,omitempty"`
	Description        string                `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes     `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*InstanceAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Operations         *Operations           `xml:"operations" json:"operations,omitempty"`
	Utilization        []*Utilization        `xml:"utilization" json:"utilization,omitempty"`
}

type Operations struct {
	XMLNAME xml.Name `xml:"operations" json:"-"`
	Id      string   `xml:"id,attr" json:"id,omitempty"`
	IdRef   string   `xml:"id-ref,attr" json:"id-ref,omitempty"`
	Op      []*Op    `xml:"op" json:"op,omitempty"`
}

type Op struct {
	XMLNAME            xml.Name              `xml:"op" json:"-"`
	Id                 string                `xml:"id,attr" json:"id"`
	Name               string                `xml:"name,attr" json:"name"`
	Interval           string                `xml:"interval,attr" json:"interval"`
	Description        string                `xml:"description,attr" json:"description,omitempty"`
	StartDelay         string                `xml:"start-delay,attr" json:"start-delay,omitempty"`
	IntervalOrigin     string                `xml:"interval-origin,attr" json:"interval-origin,omitempty"`
	Timeout            string                `xml:"timeout,attr" json:"timeout,omitempty"`
	Enabled            string                `xml:"enabled,attr" json:"enabled,omitempty"`
	RecordPending      string                `xml:"record-pending,attr" json:"record-pending,omitempty"`
	Role               string                `xml:"role,attr" json:"role,omitempty"`
	OnFail             string                `xml:"on-fail,attr" json:"on-fail,omitempty"`
	MetaAttributes     []*MetaAttributes     `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*InstanceAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
}

type Template struct {
	XMLNAME            xml.Name              `xml:"template" json:"-"`
	Id                 string                `xml:"id,attr" json:"id"`
	Class              string                `xml:"class,attr" json:"class,omitempty"`
	Provider           string                `xml:"provider,attr" json:"provider,omitempty"`
	Type               string                `xml:"type,attr" json:"type"`
	Description        string                `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes     `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*InstanceAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Operations         *Operations           `xml:"operations" json:"operations,omitempty"`
	Utilization        []*Utilization        `xml:"utilization" json:"utilization,omitempty"`
}

type Group struct {
	XMLNAME            xml.Name              `xml:"group" json:"-"`
	Id                 string                `xml:"id,attr" json:"id"`
	Description        string                `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes     `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*InstanceAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Primitive          []*Primitive          `xml:"primitive" json:"primitive"`
}

type Clone struct {
	XMLNAME            xml.Name              `xml:"clone" json:"-"`
	Id                 string                `xml:"id,attr" json:"id"`
	Description        string                `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes     `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*InstanceAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Primitive          *Primitive            `xml:"primitive" json:"primitive,omitempty"`
	Group              *Group                `xml:"group" json:"group,omitempty"`
}

type Master struct {
	XMLNAME            xml.Name              `xml:"master" json:"-"`
	Id                 string                `xml:"id,attr" json:"id"`
	Description        string                `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes     `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*InstanceAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Primitive          *Primitive            `xml:"primitive" json:"primitive,omitempty"`
	Group              *Group                `xml:"group" json:"group,omitempty"`
}

type Bundle struct {
	XMLNAME            xml.Name              `xml:"bundle" json:"-"`
	Id                 string                `xml:"id,attr" json:"id"`
	Description        string                `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes     `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*InstanceAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Docker             *Docker               `xml:"docker" json:"docker,omitempty"`
	Rkt                *Rkt                  `xml:"rkt" json:"rkt,omitempty"`
	Podman             *Podman               `xml:"podman" json:"podman,omitempty"`
	Network            *Network              `xml:"network" json:"network,omitempty"`
	Storage            *Storage              `xml:"storage" json:"storage,omitempty"`
	Primitive          *Primitive            `xml:"primitive" json:"primitive,omitempty"`
}

type Docker struct {
	XMLNAME         xml.Name `xml:"docker" json:"-"`
	Image           string   `xml:"image,attr" json:"image"`
	Replicas        string   `xml:"replicas,attr" json:"replicas,omitempty"`
	ReplicasPerHost string   `xml:"replicas-per-host,attr" json:"replicas-per-host,omitempty"`
	Masters         string   `xml:"masters,attr" json:"masters,omitempty"`
	PromotedMax     string   `xml:"promoted-max,attr" json:"promoted-max,omitempty"`
	RunCommand      string   `xml:"run-command,attr" json:"run-command,omitempty"`
	Network         string   `xml:"network,attr" json:"network,omitempty"`
	Options         string   `xml:"options,attr" json:"options,omitempty"`
}

type Rkt struct {
	XMLNAME         xml.Name `xml:"rkt" json:"-"`
	Image           string   `xml:"image,attr" json:"image"`
	Replicas        string   `xml:"replicas,attr" json:"replicas,omitempty"`
	ReplicasPerHost string   `xml:"replicas-per-host,attr" json:"replicas-per-host,omitempty"`
	Masters         string   `xml:"masters,attr" json:"masters,omitempty"`
	PromotedMax     string   `xml:"promoted-max,attr" json:"promoted-max,omitempty"`
	RunCommand      string   `xml:"run-command,attr" json:"run-command,omitempty"`
	Network         string   `xml:"network,attr" json:"network,omitempty"`
	Options         string   `xml:"options,attr" json:"options,omitempty"`
}

type Podman struct {
	XMLNAME         xml.Name `xml:"podman" json:"-"`
	Image           string   `xml:"image,attr" json:"image"`
	Replicas        string   `xml:"replicas,attr" json:"replicas,omitempty"`
	ReplicasPerHost string   `xml:"replicas-per-host,attr" json:"replicas-per-host,omitempty"`
	Masters         string   `xml:"masters,attr" json:"masters,omitempty"`
	PromotedMax     string   `xml:"promoted-max,attr" json:"promoted-max,omitempty"`
	RunCommand      string   `xml:"run-command,attr" json:"run-command,omitempty"`
	Network         string   `xml:"network,attr" json:"network,omitempty"`
	Options         string   `xml:"options,attr" json:"options,omitempty"`
}

type Network struct {
	XMLNAME       xml.Name       `xml:"network" json:"-"`
	IpRangeStart  string         `xml:"ip-range-start,attr" json:"ip-range-start,omitempty"`
	ControlPort   string         `xml:"control-port,attr" json:"control-port,omitempty"`
	HostInterface string         `xml:"host-interface,attr" json:"host-interface,omitempty"`
	HostNetmask   string         `xml:"host-netmask,attr" json:"host-netmask,omitempty"`
	AddHost       string         `xml:"add-host,attr" json:"add-host,omitempty"`
	PortMapping   []*PortMapping `xml:"port-mapping" json:"port-mapping,omitempty"`
}

type PortMapping struct {
	XMLNAME      xml.Name `xml:"port-mapping" json:"-"`
	Id           string   `xml:"id,attr" json:"id"`
	Port         string   `xml:"port,attr" json:"port,omitempty"`
	InternalPort string   `xml:"internal-port,attr" json:"internal-port,omitempty"`
	Range        string   `xml:"range,attr" json:"range,omitempty"`
}

type Storage struct {
	XMLNAME        xml.Name          `xml:"storage" json:"-"`
	StorageMapping []*StorageMapping `xml:"storage-mapping" json:"storage-mapping,omitempty"`
}

type StorageMapping struct {
	XMLNAME       xml.Name `xml:"storage-mapping" json:"-"`
	Id            string   `xml:"id,attr" json:"id"`
	SourceDir     string   `xml:"source-dir,attr" json:"source-dir,omitempty"`
	SourceDirRoot string   `xml:"source-dir-root,attr" json:"source-dir-root,omitempty"`
	TargetDir     string   `xml:"target-dir,attr" json:"target-dir"`
	Options       string   `xml:"options,attr" json:"options,omitempty"`
}

type Constraints struct {
	XMLNAME       xml.Name         `xml:"constraints" json:"-"`
	RscLocation   []*RscLocation   `xml:"rsc_location" json:"rsc_location,omitempty"`
	RscColocation []*RscColocation `xml:"rsc_colocation" json:"rsc_colocation,omitempty"`
	RscOrder      []*RscOrder      `xml:"rsc_order" json:"rsc_order,omitempty"`
	RscTicket     []*RscTicket     `xml:"rsc_ticket" json:"rsc_ticket,omitempty"`
	URLType       string           `json:"-"`
	URLIndex      int              `json:"-"`
}

type RscLocation struct {
	XMLNAME           xml.Name       `xml:"rsc_location" json:"-"`
	Id                string         `xml:"id,attr" json:"id"`
	Rsc               string         `xml:"rsc,attr" json:"rsc,omitempty"`
	RscPattern        string         `xml:"rsc-pattern,attr" json:"rsc-pattern,omitempty"`
	Role              string         `xml:"role,attr" json:"role,omitempty"`
	ResourceSet       []*ResourceSet `xml:"resource_set" json:"resource_set,omitempty"`
	Score             string         `xml:"score,attr" json:"score,omitempty"`
	Node              string         `xml:"node,attr" json:"node,omitempty"`
	Rule              []*Rule        `xml:"rule" json:"rule,omitempty"`
	Lifetime          *Lifetime      `xml:"lifetime" json:"lifetime,omitempty"`
	ResourceDiscovery string         `xml:"resource-discovery,attr" json:"resource-discovery,omitempty"`
}

type ResourceSet struct {
	XMLNAME     xml.Name       `xml:"resource_set" json:"-"`
	IdRef       string         `xml:"id-ref,attr" json:"id-ref,omitempty"`
	Id          string         `xml:"id,attr" json:"id,omitempty"`
	Sequential  string         `xml:"sequential,attr" json:"sequential,omitempty"`
	RequireAll  string         `xml:"require-all,attr" json:"require-all,omitempty"`
	Ordering    string         `xml:"ordering,attr" json:"ordering,omitempty"`
	Action      string         `xml:"action,attr" json:"action,omitempty"`
	Role        string         `xml:"role,attr" json:"role,omitempty"`
	Score       string         `xml:"score,attr" json:"score,omitempty"`
	Kind        string         `xml:"kind,attr" json:"kind,omitempty"`
	ResourceRef []*ResourceRef `xml:"resource_ref" json:"resource_ref,omitempty"`
}

type ResourceRef struct {
	XMLNAME xml.Name `xml:"resource_ref" json:"-"`
	Id      string   `xml:"id,attr" json:"id"`
}

type Lifetime struct {
	XMLNAME xml.Name `xml:"lifetime" json:"-"`
	Rule    []*Rule  `xml:"rule" json:"rule"`
}

type RscColocation struct {
	XMLNAME       xml.Name       `xml:"rsc_colocation" json:"-"`
	Id            string         `xml:"id,attr" json:"id"`
	Score         string         `xml:"score,attr" json:"score,omitempty"`
	Lifetime      *Lifetime      `xml:"lifetime" json:"lifetime,omitempty"`
	ResourceSet   []*ResourceSet `xml:"resource_set" json:"resource_set,omitempty"`
	Rsc           string         `xml:"rsc,attr" json:"rsc,omitempty"`
	WithRsc       string         `xml:"with-rsc,attr" json:"with-rsc,omitempty"`
	NodeAttribute string         `xml:"node-attribute,attr" json:"node-attribute,omitempty"`
	RscRole       string         `xml:"rsc-role,attr" json:"rsc-role,omitempty"`
	WithRscRole   string         `xml:"with-rsc-role,attr" json:"with-rsc-role,omitempty"`
}

type RscOrder struct {
	XMLNAME     xml.Name       `xml:"rsc_order" json:"-"`
	Id          string         `xml:"id,attr" json:"id"`
	Lifetime    *Lifetime      `xml:"lifetime" json:"lifetime,omitempty"`
	Symmetrical string         `xml:"symmetrical,attr" json:"symmetrical,omitempty"`
	RequireAll  string         `xml:"require-all,attr" json:"require-all,omitempty"`
	Score       string         `xml:"score,attr" json:"score,omitempty"`
	Kind        string         `xml:"kind,attr" json:"kind,omitempty"`
	ResourceSet []*ResourceSet `xml:"resource_set" json:"resource_set,omitempty"`
	First       string         `xml:"first,attr" json:"first,omitempty"`
	Then        string         `xml:"then,attr" json:"then,omitempty"`
	FirstAction string         `xml:"first-action,attr" json:"first-action,omitempty"`
	ThenAction  string         `xml:"then-action,attr" json:"then-action,omitempty"`
}

type RscTicket struct {
	XMLNAME     xml.Name       `xml:"rsc_ticket" json:"-"`
	Id          string         `xml:"id,attr" json:"id"`
	ResourceSet []*ResourceSet `xml:"resource_set" json:"resource_set,omitempty"`
	Rsc         string         `xml:"rsc,attr" json:"rsc,omitempty"`
	RscRole     string         `xml:"rsc-role,attr" json:"rsc-role,omitempty"`
	Ticket      string         `xml:"ticket,attr" json:"ticket"`
	LossPolicy  string         `xml:"loss-policy,attr" json:"loss-policy,omitempty"`
}

type FencingTopology struct {
	XMLNAME      xml.Name        `xml:"fencing-topology" json:"-"`
	FencingLevel []*FencingLevel `xml:"fencing-level" json:"fencing-level,omitempty"`
	URLType      string          `json:"-"`
	URLIndex     int             `json:"-"`
}

type FencingLevel struct {
	XMLNAME         xml.Name `xml:"fencing-level" json:"-"`
	Id              string   `xml:"id,attr" json:"id"`
	Target          string   `xml:"target,attr" json:"target,omitempty"`
	TargetPattern   string   `xml:"target-pattern,attr" json:"target-pattern,omitempty"`
	TargetAttribute string   `xml:"target-attribute,attr" json:"target-attribute,omitempty"`
	TargetValue     string   `xml:"target-value,attr" json:"target-value,omitempty"`
	Index           string   `xml:"index,attr" json:"index"`
	Devices         string   `xml:"devices,attr" json:"devices"`
}

type Acls struct {
	XMLNAME   xml.Name     `xml:"acls" json:"-"`
	AclTarget []*AclTarget `xml:"acl_target" json:"acl_target,omitempty"`
	AclGroup  []*AclGroup  `xml:"acl_group" json:"acl_group,omitempty"`
	AclRole   []*AclRole   `xml:"acl_role" json:"acl_role,omitempty"`
	URLType   string       `json:"-"`
	URLIndex  int          `json:"-"`
}

type AclTarget struct {
	XMLNAME xml.Name `xml:"acl_target" json:"-"`
	Id      string   `xml:"id,attr" json:"id"`
	Role    []*Role  `xml:"role" json:"role,omitempty"`
}

type Role struct {
	XMLNAME xml.Name `xml:"role" json:"-"`
	Id      string   `xml:"id,attr" json:"id"`
}

type AclGroup struct {
	XMLNAME xml.Name `xml:"acl_group" json:"-"`
	Id      string   `xml:"id,attr" json:"id"`
	Role    []*Role  `xml:"role" json:"role,omitempty"`
}

type AclRole struct {
	XMLNAME       xml.Name         `xml:"acl_role" json:"-"`
	Id            string           `xml:"id,attr" json:"id"`
	Description   string           `xml:"description,attr" json:"description,omitempty"`
	AclPermission []*AclPermission `xml:"acl_permission" json:"acl_permission,omitempty"`
}

type AclPermission struct {
	XMLNAME     xml.Name `xml:"acl_permission" json:"-"`
	Id          string   `xml:"id,attr" json:"id"`
	Kind        string   `xml:"kind,attr" json:"kind"`
	Xpath       string   `xml:"xpath,attr" json:"xpath,omitempty"`
	Reference   string   `xml:"reference,attr" json:"reference,omitempty"`
	ObjectType  string   `xml:"object-type,attr" json:"object-type,omitempty"`
	Attribute   string   `xml:"attribute,attr" json:"attribute,omitempty"`
	Description string   `xml:"description,attr" json:"description,omitempty"`
}

type Tags struct {
	XMLNAME  xml.Name `xml:"tags" json:"-"`
	Tag      []*Tag   `xml:"tag" json:"tag,omitempty"`
	URLType  string   `json:"-"`
	URLIndex int      `json:"-"`
}

type Tag struct {
	XMLNAME xml.Name  `xml:"tag" json:"-"`
	Id      string    `xml:"id,attr" json:"id"`
	ObjRef  []*ObjRef `xml:"obj_ref" json:"obj_ref"`
}

type ObjRef struct {
	XMLNAME xml.Name `xml:"obj_ref" json:"-"`
	Id      string   `xml:"id,attr" json:"id"`
}

type Alerts struct {
	XMLNAME  xml.Name `xml:"alerts" json:"-"`
	Alert    []*Alert `xml:"alert" json:"alert,omitempty"`
	URLType  string   `json:"-"`
	URLIndex int      `json:"-"`
}

type Alert struct {
	XMLNAME            xml.Name              `xml:"alert" json:"-"`
	Id                 string                `xml:"id,attr" json:"id"`
	Description        string                `xml:"description,attr" json:"description,omitempty"`
	Path               string                `xml:"path,attr" json:"path"`
	MetaAttributes     []*MetaAttributes     `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*InstanceAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Select             *Select               `xml:"select" json:"select,omitempty"`
	Recipient          []*Recipient          `xml:"recipient" json:"recipient,omitempty"`
}

type Select struct {
	XMLNAME          xml.Name          `xml:"select" json:"-"`
	SelectAttributes *SelectAttributes `xml:"select_attributes" json:"select_attributes,omitempty"`
	SelectFencing    *SelectFencing    `xml:"select_fencing" json:"select_fencing,omitempty"`
	SelectNodes      *SelectNodes      `xml:"select_nodes" json:"select_nodes,omitempty"`
	SelectResources  *SelectResources  `xml:"select_resources" json:"select_resources,omitempty"`
}

type SelectAttributes struct {
	XMLNAME   xml.Name     `xml:"select_attributes" json:"-"`
	Attribute []*Attribute `xml:"attribute" json:"attribute,omitempty"`
}

type Attribute struct {
	XMLNAME xml.Name `xml:"attribute" json:"-"`
	Id      string   `xml:"id,attr" json:"id"`
	Name    string   `xml:"name,attr" json:"name"`
}

type SelectFencing struct {
	XMLNAME xml.Name `xml:"select_fencing" json:"-"`
}

type SelectNodes struct {
	XMLNAME xml.Name `xml:"select_nodes" json:"-"`
}

type SelectResources struct {
	XMLNAME xml.Name `xml:"select_resources" json:"-"`
}

type Recipient struct {
	XMLNAME            xml.Name              `xml:"recipient" json:"-"`
	Id                 string                `xml:"id,attr" json:"id"`
	Description        string                `xml:"description,attr" json:"description,omitempty"`
	Value              string                `xml:"value,attr" json:"value"`
	MetaAttributes     []*MetaAttributes     `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*InstanceAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
}

type Status struct {
	XMLNAME   xml.Name     `xml:"status" json:"-"`
	NodeState []*NodeState `xml:"node_state" json:"node_state,omitempty"`
	URLType   string       `json:"-"`
	URLIndex  int          `json:"-"`
}

type NodeState struct {
	XMLNAME        xml.Name `xml:"node_state" json:"-"`
	Id             string   `xml:"id,attr" json:"id"`
	Uname          string   `xml:"uname,attr" json:"uname"`
	InCcm          string   `xml:"in_ccm,attr" json:"in_ccm"`
	Crmd           string   `xml:"crmd,attr" json:"crmd"`
	CrmDebugOrigin string   `xml:"crm-debug-origin,attr" json:"crm-debug-origin"`
	Join           string   `xml:"join,attr" json:"join"`
	Expected       string   `xml:"expected,attr" json:"expected"`
	Lrm            *Lrm     `xml:"lrm" json:"lrm,omitempty"`
}

type Lrm struct {
	XMLNAME      xml.Name      `xml:"lrm" json:"-"`
	Id           string        `xml:"id,attr" json:"id"`
	LrmResources *LrmResources `xml:"lrm_resources" json:"lrm_resources,omitempty"`
}

type LrmResources struct {
	XMLNAME     xml.Name       `xml:"lrm_resources" json:"-"`
	LrmResource []*LrmResource `xml:"lrm_resource" json:"lrm_resource,omitempty"`
}

type LrmResource struct {
	XMLNAME  xml.Name  `xml:"lrm_resource" json:"-"`
	Id       string    `xml:"id,attr" json:"id"`
	Type     string    `xml:"type,attr" json:"type"`
	Class    string    `xml:"class,attr" json:"class"`
	Provider string    `xml:"provider,attr" json:"provider,omitempty"`
	LrmRscOp *LrmRscOp `xml:"lrm_rsc_op" json:"lrm_rsc_op,omitempty"`
}

type LrmRscOp struct {
	XMLNAME         xml.Name `xml:"lrm_rsc_op" json:"-"`
	Id              string   `xml:"id,attr" json:"id"`
	OperationKey    string   `xml:"operation_key,attr" json:"operation_key"`
	Operation       string   `xml:"operation,attr" json:"operation"`
	CrmDebugOrigin  string   `xml:"crm-debug-origin,attr" json:"crm-debug-origin"`
	CrmFeatureSet   string   `xml:"crm_feature_set,attr" json:"crm_feature_set"`
	TransitionKey   string   `xml:"transition-key,attr" json:"transition-key"`
	TransitionMagic string   `xml:"transition-magic,attr" json:"transition-magic"`
	ExitReason      string   `xml:"exit-reason,attr" json:"exit-reason"`
	OnNode          string   `xml:"on_node,attr" json:"on_node"`
	CallId          string   `xml:"call-id,attr" json:"call-id"`
	RcCode          string   `xml:"rc-code,attr" json:"rc-code"`
	OpStatus        string   `xml:"op-status,attr" json:"op-status"`
	Interval        string   `xml:"interval,attr" json:"interval"`
	LastRun         string   `xml:"last-run,attr" json:"last-run"`
	LastRcChange    string   `xml:"last-rc-change,attr" json:"last-rc-change"`
	ExecTime        string   `xml:"exec-time,attr" json:"exec-time"`
	QueueTime       string   `xml:"queue-time,attr" json:"queue-time"`
	OpDigest        string   `xml:"op-digest,attr" json:"op-digest"`
}

type TypeIndex struct {
	Type  string
	Index int
}
