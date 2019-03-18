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
}

type CrmConfig struct {
	XMLNAME            xml.Name          `xml:"crm_config" json:"-"`
	ClusterPropertySet []*MetaAttributes `xml:"cluster_property_set" json:"cluster_property_set,omitempty"`
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
	Duration  *DateSpec `xml:"duration" json:"duration,omitempty"`
	DateSpec  *DateSpec `xml:"date_spec" json:"date_spec,omitempty"`
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
}

type MetaAttributes struct {
	XMLNAME xml.Name  `xml:"meta_attributes" json:"-"`
	IdRef   string    `xml:"id-ref,attr" json:"id-ref,omitempty"`
	Id      string    `xml:"id,attr" json:"id,omitempty"`
	Rule    *Rule     `xml:"rule" json:"rule,omitempty"`
	Nvpair  []*Nvpair `xml:"nvpair" json:"nvpair,omitempty"`
	Score   string    `xml:"score,attr" json:"score,omitempty"`
}

type OpDefaults struct {
	XMLNAME        xml.Name          `xml:"op_defaults" json:"-"`
	MetaAttributes []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
}

type Nodes struct {
	XMLNAME xml.Name `xml:"nodes" json:"-"`
	Node    []*Node  `xml:"node" json:"node,omitempty"`
}

type Node struct {
	XMLNAME            xml.Name          `xml:"node" json:"-"`
	Id                 string            `xml:"id,attr" json:"id"`
	Uname              string            `xml:"uname,attr" json:"uname"`
	Type               string            `xml:"type,attr" json:"type,omitempty"`
	Description        string            `xml:"description,attr" json:"description,omitempty"`
	Score              string            `xml:"score,attr" json:"score,omitempty"`
	InstanceAttributes []*MetaAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Utilization        []*MetaAttributes `xml:"utilization" json:"utilization,omitempty"`
}

type Resources struct {
	XMLNAME   xml.Name     `xml:"resources" json:"-"`
	Primitive []*Primitive `xml:"primitive" json:"primitive,omitempty"`
	Template  []*Template  `xml:"template" json:"template,omitempty"`
	Group     []*Group     `xml:"group" json:"group,omitempty"`
	Clone     []*Clone     `xml:"clone" json:"clone,omitempty"`
	Master    []*Master    `xml:"master" json:"master,omitempty"`
	Bundle    []*Bundle    `xml:"bundle" json:"bundle,omitempty"`
}

type Primitive struct {
	XMLNAME            xml.Name          `xml:"primitive" json:"-"`
	Id                 string            `xml:"id,attr" json:"id"`
	Class              string            `xml:"class,attr" json:"class,omitempty"`
	Provider           string            `xml:"provider,attr" json:"provider,omitempty"`
	Type               string            `xml:"type,attr" json:"type,omitempty"`
	Template           string            `xml:"template,attr" json:"template,omitempty"`
	Description        string            `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*MetaAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Operations         *Operations       `xml:"operations" json:"operations,omitempty"`
	Utilization        []*MetaAttributes `xml:"utilization" json:"utilization,omitempty"`
}

type Operations struct {
	XMLNAME xml.Name `xml:"operations" json:"-"`
	Id      string   `xml:"id,attr" json:"id,omitempty"`
	IdRef   string   `xml:"id-ref,attr" json:"id-ref,omitempty"`
	Op      []*Op    `xml:"op" json:"op,omitempty"`
}

type Op struct {
	XMLNAME            xml.Name          `xml:"op" json:"-"`
	Id                 string            `xml:"id,attr" json:"id"`
	Name               string            `xml:"name,attr" json:"name"`
	Interval           string            `xml:"interval,attr" json:"interval"`
	Description        string            `xml:"description,attr" json:"description,omitempty"`
	StartDelay         string            `xml:"start-delay,attr" json:"start-delay,omitempty"`
	IntervalOrigin     string            `xml:"interval-origin,attr" json:"interval-origin,omitempty"`
	Timeout            string            `xml:"timeout,attr" json:"timeout,omitempty"`
	Enabled            string            `xml:"enabled,attr" json:"enabled,omitempty"`
	RecordPending      string            `xml:"record-pending,attr" json:"record-pending,omitempty"`
	Role               string            `xml:"role,attr" json:"role,omitempty"`
	OnFail             string            `xml:"on-fail,attr" json:"on-fail,omitempty"`
	MetaAttributes     []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*MetaAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
}

type Template struct {
	XMLNAME            xml.Name          `xml:"template" json:"-"`
	Id                 string            `xml:"id,attr" json:"id"`
	Class              string            `xml:"class,attr" json:"class,omitempty"`
	Provider           string            `xml:"provider,attr" json:"provider,omitempty"`
	Type               string            `xml:"type,attr" json:"type"`
	Description        string            `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*MetaAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Operations         *Operations       `xml:"operations" json:"operations,omitempty"`
	Utilization        []*MetaAttributes `xml:"utilization" json:"utilization,omitempty"`
}

type Group struct {
	XMLNAME            xml.Name          `xml:"group" json:"-"`
	Id                 string            `xml:"id,attr" json:"id"`
	Description        string            `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*MetaAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Primitive          []*Primitive      `xml:"primitive" json:"primitive"`
}

type Clone struct {
	XMLNAME            xml.Name          `xml:"clone" json:"-"`
	Id                 string            `xml:"id,attr" json:"id"`
	Description        string            `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*MetaAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Primitive          *Primitive        `xml:"primitive" json:"primitive,omitempty"`
	Group              *Group            `xml:"group" json:"group,omitempty"`
}

type Master struct {
	XMLNAME            xml.Name          `xml:"master" json:"-"`
	Id                 string            `xml:"id,attr" json:"id"`
	Description        string            `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*MetaAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Primitive          *Primitive        `xml:"primitive" json:"primitive,omitempty"`
	Group              *Group            `xml:"group" json:"group,omitempty"`
}

type Bundle struct {
	XMLNAME            xml.Name          `xml:"bundle" json:"-"`
	Id                 string            `xml:"id,attr" json:"id"`
	Description        string            `xml:"description,attr" json:"description,omitempty"`
	MetaAttributes     []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*MetaAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Docker             *Docker           `xml:"docker" json:"docker,omitempty"`
	Rkt                *Rkt              `xml:"rkt" json:"rkt,omitempty"`
	Podman             *Podman           `xml:"podman" json:"podman,omitempty"`
	Network            *Network          `xml:"network" json:"network,omitempty"`
	Storage            *Storage          `xml:"storage" json:"storage,omitempty"`
	Primitive          *Primitive        `xml:"primitive" json:"primitive,omitempty"`
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
	XMLNAME xml.Name `xml:"tags" json:"-"`
	Tag     []*Tag   `xml:"tag" json:"tag,omitempty"`
}

type Tag struct {
	XMLNAME xml.Name  `xml:"tag" json:"-"`
	Id      string    `xml:"id,attr" json:"id"`
	ObjRef  []*ObjRef `xml:"obj_ref" json:"obj_ref"`
}

type ObjRef struct {
	XMLNAME xml.Name `xml:"obj_ref" json:"-"`
	Id      string   `xml:"id ,attr" json:"id "`
}

type Alerts struct {
	XMLNAME xml.Name `xml:"alerts" json:"-"`
	Alert   []*Alert `xml:"alert" json:"alert,omitempty"`
}

type Alert struct {
	XMLNAME            xml.Name          `xml:"alert" json:"-"`
	Id                 string            `xml:"id,attr" json:"id"`
	Description        string            `xml:"description,attr" json:"description,omitempty"`
	Path               string            `xml:"path,attr" json:"path"`
	MetaAttributes     []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*MetaAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
	Select             *Select           `xml:"select" json:"select,omitempty"`
	Recipient          []*Recipient      `xml:"recipient" json:"recipient,omitempty"`
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
	XMLNAME            xml.Name          `xml:"recipient" json:"-"`
	Id                 string            `xml:"id,attr" json:"id"`
	Description        string            `xml:"description,attr" json:"description,omitempty"`
	Value              string            `xml:"value,attr" json:"value"`
	MetaAttributes     []*MetaAttributes `xml:"meta_attributes" json:"meta_attributes,omitempty"`
	InstanceAttributes []*MetaAttributes `xml:"instance_attributes" json:"instance_attributes,omitempty"`
}

type Status struct {
	XMLNAME xml.Name `xml:"status" json:"-"`
}

type CrmMon struct {
	XMLNAME              xml.Name              `xml:"crm_mon" json:"-"`
	CrmMonSummary        *CrmMonSummary        `xml:"summary" json:"summary"`
	CrmMonNodes          *CrmMonNodes          `xml:"nodes" json:"nodes"`
	CrmMonResources      *CrmMonResources      `xml:"resources" json:"resources"`
	CrmMonNodeAttributes *CrmMonNodeAttributes `xml:"node_attributes" json:"node_attributes"`
	CrmMonNodeHistory    *CrmMonNodeHistory    `xml:"node_history" json:"node_history"`
	CrmMonFailures       *CrmMonFailures       `xml:"failures" json:"failures"`
}

type CrmMonSummary struct {
	XMLNAME                    xml.Name                    `xml:"summary" json:"-"`
	SummaryStack               *SummaryStack               `xml:"stack" json:"stack"`
	SummaryCurrentDc           *SummaryCurrentDc           `xml:"current_dc" json:"current_dc"`
	SummaryLastUpdate          *SummaryLastUpdate          `xml:"last_update" json:"last_update"`
	SummaryLastChange          *SummaryLastChange          `xml:"last_change" json:"last_change"`
	SummaryNodesConfigured     *SummaryNodesConfigured     `xml:"nodes_configured" json:"nodes_configured"`
	SummaryResourcesConfigured *SummaryResourcesConfigured `xml:"resources_configured" json:"resources_configured"`
}

type SummaryStack struct {
	XMLNAME xml.Name `xml:"stack" json:"-"`
	Type    string   `xml:"type,attr" json:"type"`
}

type SummaryCurrentDc struct {
	XMLNAME    xml.Name `xml:"current_dc" json:"-"`
	Present    string   `xml:"present,attr" json:"present"`
	Version    string   `xml:"version,attr" json:"version"`
	Name       string   `xml:"name,attr" json:"name"`
	Id         string   `xml:"id,attr" json:"id"`
	WithQuorum string   `xml:"with_quorum,attr" json:"with_quorum"`
}

type SummaryLastUpdate struct {
	XMLNAME xml.Name `xml:"last_update" json:"-"`
	Time    string   `xml:"time,attr" json:"time"`
}

type SummaryLastChange struct {
	XMLNAME xml.Name `xml:"last_change" json:"-"`
	Time    string   `xml:"time,attr" json:"time"`
	User    string   `xml:"user,attr" json:"user"`
	Client  string   `xml:"client,attr" json:"client"`
	Origin  string   `xml:"origin,attr" json:"origin"`
}

type SummaryNodesConfigured struct {
	XMLNAME xml.Name `xml:"nodes_configured" json:"-"`
	Number  string   `xml:"number,attr" json:"number"`
}

type SummaryResourcesConfigured struct {
	XMLNAME  xml.Name `xml:"resources_configured" json:"-"`
	Number   string   `xml:"number,attr" json:"number"`
	Disabled string   `xml:"disabled,attr" json:"disabled"`
	Blocked  string   `xml:"blocked,attr" json:"blocked"`
}

type CrmMonNodes struct {
	XMLNAME   xml.Name     `xml:"nodes" json:"-"`
	NodesNode []*NodesNode `xml:"node" json:"node"`
}

type NodesNode struct {
	XMLNAME          xml.Name `xml:"node" json:"-"`
	Name             string   `xml:"name,attr" json:"name"`
	Id               string   `xml:"id,attr" json:"id"`
	Online           string   `xml:"online,attr" json:"online"`
	Standby          string   `xml:"standby,attr" json:"standby"`
	StandbyOnfail    string   `xml:"standby_onfail,attr" json:"standby_onfail"`
	Maintenance      string   `xml:"maintenance,attr" json:"maintenance"`
	Pending          string   `xml:"pending,attr" json:"pending"`
	Unclean          string   `xml:"unclean,attr" json:"unclean"`
	Shutdown         string   `xml:"shutdown,attr" json:"shutdown"`
	ExpectedUp       string   `xml:"expected_up,attr" json:"expected_up"`
	IsDc             string   `xml:"is_dc,attr" json:"is_dc"`
	ResourcesRunning string   `xml:"resources_running,attr" json:"resources_running"`
	Type             string   `xml:"type,attr" json:"type"`
}

type CrmMonResources struct {
	XMLNAME           xml.Name             `xml:"resources" json:"-"`
	ResourcesResource []*ResourcesResource `xml:"resource" json:"resource"`
	ResourcesGroup    []*ResourcesGroup    `xml:"group" json:"group"`
	ResourcesClone    []*ResourcesClone    `xml:"clone" json:"clone"`
}

type ResourcesResource struct {
	XMLNAME        xml.Name        `xml:"resource" json:"-"`
	Id             string          `xml:"id,attr" json:"id"`
	ResourceAgent  string          `xml:"resource_agent,attr" json:"resource_agent"`
	Role           string          `xml:"role,attr" json:"role"`
	Active         string          `xml:"active,attr" json:"active"`
	Orphaned       string          `xml:"orphaned,attr" json:"orphaned"`
	Blocked        string          `xml:"blocked,attr" json:"blocked"`
	Managed        string          `xml:"managed,attr" json:"managed"`
	Failed         string          `xml:"failed,attr" json:"failed"`
	FailureIgnored string          `xml:"failure_ignored,attr" json:"failure_ignored"`
	NodesRunningOn string          `xml:"nodes_running_on,attr" json:"nodes_running_on"`
	ResourceNode   []*ResourceNode `xml:"node" json:"node"`
}

type ResourceNode struct {
	XMLNAME xml.Name `xml:"node" json:"-"`
	Name    string   `xml:"name,attr" json:"name"`
	Id      string   `xml:"id,attr" json:"id"`
	Cached  string   `xml:"cached,attr" json:"cached"`
}

type ResourcesGroup struct {
	XMLNAME         xml.Name             `xml:"group" json:"-"`
	Id              string               `xml:"id,attr" json:"id"`
	NumberResources string               `xml:"number_resources,attr" json:"number_resources"`
	GroupResource   []*ResourcesResource `xml:"resource" json:"resource"`
}

type ResourcesClone struct {
	XMLNAME        xml.Name             `xml:"clone" json:"-"`
	Id             string               `xml:"id,attr" json:"id"`
	MultiState     string               `xml:"multi_state,attr" json:"multi_state"`
	Unique         string               `xml:"unique,attr" json:"unique"`
	Managed        string               `xml:"managed,attr" json:"managed"`
	Failed         string               `xml:"failed,attr" json:"failed"`
	FailureIgnored string               `xml:"failure_ignored,attr" json:"failure_ignored"`
	CloneResource  []*ResourcesResource `xml:"resource" json:"resource"`
	CloneGroup     []*ResourcesGroup    `xml:"group" json:"group"`
}

type CrmMonNodeAttributes struct {
	XMLNAME            xml.Name              `xml:"node_attributes" json:"-"`
	NodeAttributesNode []*NodeAttributesNode `xml:"node" json:"node"`
}

type NodeAttributesNode struct {
	XMLNAME xml.Name `xml:"node" json:"-"`
	Name    string   `xml:"name,attr" json:"name"`
}

type CrmMonNodeHistory struct {
	XMLNAME         xml.Name         `xml:"node_history" json:"-"`
	NodeHistoryNode *NodeHistoryNode `xml:"node" json:"node"`
}

type NodeHistoryNode struct {
	XMLNAME             xml.Name             `xml:"node" json:"-"`
	Name                string               `xml:"name,attr" json:"name"`
	NodeResourceHistory *NodeResourceHistory `xml:"resource_history" json:"resource_history"`
}

type NodeResourceHistory struct {
	XMLNAME                         xml.Name                           `xml:"resource_history" json:"-"`
	Id                              string                             `xml:"id,attr" json:"id"`
	Orphan                          string                             `xml:"orphan,attr" json:"orphan"`
	MigrationThreshold              string                             `xml:"migration-threshold,attr" json:"migration-threshold"`
	ResourceHistoryOperationHistory []*ResourceHistoryOperationHistory `xml:"operation_history" json:"operation_history"`
}

type ResourceHistoryOperationHistory struct {
	XMLNAME      xml.Name `xml:"operation_history" json:"-"`
	Call         string   `xml:"call,attr" json:"call"`
	Task         string   `xml:"task,attr" json:"task"`
	LastRcChange string   `xml:"last-rc-change,attr" json:"last-rc-change"`
	LastRun      string   `xml:"last-run,attr" json:"last-run"`
	ExecTime     string   `xml:"exec-time,attr" json:"exec-time"`
	QueueTime    string   `xml:"queue-time,attr" json:"queue-time"`
	Rc           string   `xml:"rc,attr" json:"rc"`
	RcText       string   `xml:"rc_text,attr" json:"rc_text"`
}

type CrmMonFailures struct {
	XMLNAME         xml.Name           `xml:"failures" json:"-"`
	FailuresFailure []*FailuresFailure `xml:"failure" json:"failure"`
}

type FailuresFailure struct {
	XMLNAME      xml.Name `xml:"failure" json:"-"`
	OpKey        string   `xml:"op_key,attr" json:"op_key"`
	Node         string   `xml:"node,attr" json:"node"`
	Exitstatus   string   `xml:"exitstatus,attr" json:"exitstatus"`
	Exitreason   string   `xml:"exitreason,attr" json:"exitreason"`
	Exitcode     string   `xml:"exitcode,attr" json:"exitcode"`
	Call         string   `xml:"call,attr" json:"call"`
	Status       string   `xml:"status,attr" json:"status"`
	LastRcChange string   `xml:"last-rc-change,attr" json:"last-rc-change"`
	Queued       string   `xml:"queued,attr" json:"queued"`
	Exec         string   `xml:"exec,attr" json:"exec"`
	Interval     string   `xml:"interval,attr" json:"interval"`
	Task         string   `xml:"task,attr" json:"task"`
}
