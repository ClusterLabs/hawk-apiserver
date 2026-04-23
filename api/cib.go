package api

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"path"
	"strings"
)

var Routehandler http.Handler // set from main

func renderTemplate(w http.ResponseWriter, name string, data map[string]any) {
	tmpl, err := template.ParseFiles(
		"templates/layout.html",
		fmt.Sprintf("templates/%s.html", name),
	)
	if err != nil {
		http.Error(w, "Template parsing error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
	}
}

/***************************
 * crm status --as-xml
 ***************************/

type CrmStatus struct {
	XMLName   xml.Name   `xml:"crm_mon"`
	Nodes     []CrmNode  `xml:"nodes>node"`
	Resources []CrmRsc   `xml:"resources>resource"`
	Clones    []CrmClone `xml:"resources>clone"`
}

type CrmNode struct {
	Name        string `xml:"name,attr"`
	ID          string `xml:"id,attr"`
	Online      bool   `xml:"online,attr"`
	Maintenance bool   `xml:"maintenance,attr"`
}

type CrmRsc struct {
	ID             string       `xml:"id,attr"`
	ResourceAgent  string       `xml:"resource_agent,attr"`
	Role           string       `xml:"role,attr"`
	TargetRole     string       `xml:"target_role,attr"`
	Active         bool         `xml:"active,attr"`
	Orphaned       bool         `xml:"orphaned,attr"`
	Blocked        bool         `xml:"blocked,attr"`
	Maintenance    bool         `xml:"maintenance,attr"`
	Managed        bool         `xml:"managed,attr"`
	Faield         bool         `xml:"failed,attr"`
	FailureIgnored bool         `xml:"failure_ignored,attr"`
	NodesRunningOn bool         `xml:"nodes_running_on,attr"`
	Nodes          []CrmRscNode `xml:"node"`
}

type CrmClone struct {
	ID             string   `xml:"id,attr"`
	MultiState     bool     `xml:"multi_state,attr"`
	Unique         bool     `xml:"unique,attr"`
	Maintenance    bool     `xml:"maintenance,attr"`
	Managed        bool     `xml:"managed,attr"`
	Disabled       bool     `xml:"disabled,attr"`
	Failed         bool     `xml:"failed,attr"`
	FailureIgnored bool     `xml:"failure_ignored,attr"`
	Resources      []CrmRsc `xml:"resource"`
}

type CrmRscNode struct {
	Name   string `xml:"name,attr"`
	ID     string `xml:"id,attr"`
	Cached bool   `xml:"cached,attr"`
}

func GetCrmStatus() (CrmStatus, error) {
	cmd := exec.Command("crm", "status", "--as-xml")
	output, err := cmd.Output()
	if err != nil {
		return CrmStatus{}, err
	}

	var crm CrmStatus
	if err := xml.Unmarshal(output, &crm); err != nil {
		return CrmStatus{}, err
	}

	return crm, nil
}

type CrmResourceRow struct {
	ID          string
	Type        string
	Node        string
	Status      string
	Maintenance bool
}

// flatten the CrmStatus for the easier UI
func ToCrmResourceRows(crm CrmStatus) []CrmResourceRow {
	var rows []CrmResourceRow
	for _, rsc := range crm.Resources {
		nodeName := ""
		if len(rsc.Nodes) > 0 {
			nodeName = rsc.Nodes[0].Name
		}
		rows = append(rows, CrmResourceRow{
			ID:          rsc.ID,
			Type:        rsc.ResourceAgent,
			Node:        nodeName,
			Status:      rsc.Role,
			Maintenance: rsc.Maintenance,
		})
	}
	return rows
}

/***************************
 * cibadmin -Ql
 ***************************/

type CIB struct {
	XMLName        xml.Name      `xml:"cib"`
	ValidateWith   string        `xml:"validate-with,attr"`
	Epoch          string        `xml:"epoch,attr"`
	NumUpdates     string        `xml:"num_updates,attr"`
	AdminEpoch     string        `xml:"admin_epoch,attr"`
	CibLastWritten string        `xml:"cib-last-written,attr"`
	UpdateOrigin   string        `xml:"update-origin,attr"`
	UpdateClient   string        `xml:"update-client,attr"`
	UpdateUser     string        `xml:"update-user,attr"`
	HaveQuorum     string        `xml:"have-quorum,attr"`
	DcUuid         string        `xml:"dc-uuid,attr"`
	Configuration  Configuration `xml:"configuration"`
	Status         Status        `xml:"status"`
}

type Configuration struct {
	CrmConfig   CrmConfig   `xml:"crm_config"`
	Nodes       []Node      `xml:"nodes>node"`
	Constraints Constraints `xml:"constraints"`
	Primitives  []Primitive `xml:"resources>primitive"`
}

type CrmConfig struct {
	ClusterPropertySet ClusterPropertySet `xml:"cluster_property_set"`
}

type ClusterPropertySet struct {
	ID      string   `xml:"id,attr"`
	NVPairs []Nvpair `xml:"nvpair"`
}

type Constraints struct {
	Colocations []RscColocation `xml:"rsc_colocation"`
	Locations   []RscLocation   `xml:"rsc_location"`
	Orders      []RscOrder      `xml:"rsc_order"`
}

// To add colocation constraint: crm configure colocation location_constration 5000: dummy1 dummy2
type RscColocation struct {
	ID          string `xml:"id,attr"`
	Score       string `xml:"score,attr"`
	Rsc         string `xml:"rsc,attr"`
	RscRole     string `xml:"rsc-role,attr"`
	WithRsc     string `xml:"with-rsc,attr"`
	WithRscRole string `xml:"with-rsc-role,attr"`
}

type RscLocation struct {
	ID    string `xml:"id,attr"`
	Score string `xml:"score,attr"`
	Rsc   string `xml:"rsc,attr"`
	Node  string `xml:"node,attr"`
}

type RscOrder struct {
	ID          string `xml:"id,attr"`
	Kind        string `xml:"kind,attr"`
	First       string `xml:"first,attr"`
	FirstAction string `xml:"first-action,attr"`
	Then        string `xml:"then,attr"`
	ThenAction  string `xml:"then-action,attr"`
}

type Node struct {
	ID           string   `xml:"id,attr"`
	Uname        string   `xml:"uname,attr"`
	Utilizations []Nvpair `xml:"utilization>nvpair"`
	Attributes   []Nvpair `xml:"instance_attributes>nvpair"`
}

type Primitive struct {
	XMLName            xml.Name          `xml:"primitive"` // w/o it, marshalled xml would be 'Primitive' (not 'primitive')
	ID                 string            `xml:"id,attr" json:"id"`
	Class              string            `xml:"class,attr" json:"class"`
	Provider           string            `xml:"provider,attr" json:"provider"`
	Type               string            `xml:"type,attr" json:"type"`
	MetaAttributes     MetaAttribute     `xml:"meta_attributes" json:"meta_attributes"`
	InstanceAttributes InstanceAttribute `xml:"instance_attributes" json:"instance_attributes"`
	Operations         []Operation       `xml:"operations>op" json:"operations"`
	Utilizations       []Nvpair          `xml:"utilization>nvpair" json:"utilizations"`
}

type MetaAttribute struct {
	ID      string   `xml:"id,attr" json:"id"`
	NVPairs []Nvpair `xml:"nvpair" json:"nvpair"`
}

type InstanceAttribute struct {
	ID      string   `xml:"id,attr" json:"id"`
	NVPairs []Nvpair `xml:"nvpair" json:"nvpair"`
}

/* don't confuse it with Action.
 * Action is "crm_resource --show-metadata ocf:pacemaker:Dummy"
 * Operation is "cibamdin -Ql" */
/* However, they are so much alike; I think they can be merged (18.06.2026) */
type Operation struct {
	XMLName        xml.Name `xml:"op"`
	Depth          string   `xml:"depth,attr,omitempty" json:"depth"`
	Description    string   `xml:"description,attr,omitempty" json:"description"`
	Enabled        string   `xml:"enabled,attr,omitempty" json:"enabled"`
	ID             string   `xml:"id,attr" json:"id"`
	Interval       string   `xml:"interval,attr,omitempty" json:"interval"`
	IntervalOrigin string   `xml:"interval-origin,attr,omitempty" json:"interval-origin"`
	OnFail         string   `xml:"on-fail,attr,omitempty" json:"on-fail"`
	Name           string   `xml:"name,attr" json:"name"`
	RecordPending  string   `xml:"record-pending,attr,omitempty" json:"record-pending"`
	Requires       string   `xml:"requires,attr,omitempty" json:"requires"`
	Role           string   `xml:"role,attr,omitempty" json:"role"`
	StartDelay     string   `xml:"start-delay,attr,omitempty" json:"start-delay"`
	Timeout        string   `xml:"timeout,attr,omitempty" json:"timeout"`
}

type Nvpair struct {
	XMLName xml.Name `xml:"nvpair" json:"nvpair"`
	ID      string   `xml:"id,attr" json:"id"`
	Name    string   `xml:"name,attr" json:"name"`
	Value   string   `xml:"value,attr" json:"value"`
}

type Status struct {
	NodeStates []NodeState `xml:"node_state"`
}

type NodeState struct {
	Uname string `xml:"uname,attr"`
	LRM   LRM    `xml:"lrm"`
}

type LRM struct {
	Resources []LRMResource `xml:"lrm_resources>lrm_resource"`
}

type LRMResource struct {
	ID    string  `xml:"id,attr"`
	Class string  `xml:"class,attr"`
	Type  string  `xml:"type,attr"`
	Ops   []LRMOp `xml:"lrm_rsc_op"`
}

type LRMOp struct {
	ID           string `xml:"id,attr"`
	CallID       string `xml:"call-id,attr"`
	ExecTime     string `xml:"exec-time,attr"`
	LastRcChange string `xml:"last-rc-change,attr"`
	OnNode       string `xml:"on_node,attr"`
	Operation    string `xml:"operation,attr"`
	OperationKey string `xml:"operation_key,attr"`
	OpStatus     string `xml:"op-status,attr"`
	RCCode       string `xml:"rc-code,attr"`
}

type ResourceRow struct {
	ID             string
	Class          string
	Provider       string
	Type           string
	Node           string
	Status         string
	TargetRole     string
	Constraints    Constraints
	MetaAttributes []Nvpair
	Events         []LRMOp
	Utilizations   []Nvpair
}

type NodeRow struct {
	ID           string
	Name         string
	Utilizations []Nvpair
}

func getResourceConstraints(resourceName string, configuration Configuration) Constraints {
	var Colocations []RscColocation
	var Locations []RscLocation
	for _, colocation := range configuration.Constraints.Colocations {
		if (colocation.Rsc == resourceName) || (colocation.WithRsc == resourceName) {
			Colocations = append(Colocations, colocation)
		}
	}
	for _, location := range configuration.Constraints.Locations {
		if location.Rsc == resourceName {
			Locations = append(Locations, location)
		}
	}
	return Constraints{Colocations, Locations, configuration.Constraints.Orders}
}

// return node name where the resource is running or "" if stopped
func getResourceRunningNode(resourceName string, nodeStates []NodeState) string {
	for _, nodeState := range nodeStates {
		for _, lrmResource := range nodeState.LRM.Resources {
			if lrmResource.ID == resourceName {
				for _, lrmRscOp := range lrmResource.Ops {
					if strings.ToLower(lrmRscOp.Operation) == "start" {
						return nodeState.Uname
					}
				}
			}
		}
	}
	return ""
}

func getResourceEvents(resourceName string, nodeStates []NodeState) []LRMOp {
	var result []LRMOp
	for _, nodeState := range nodeStates {
		for _, lrmResource := range nodeState.LRM.Resources {
			if lrmResource.ID == resourceName {
				result = append(result, lrmResource.Ops...)
			}
		}
	}
	return result
}

func GetCIBResources() ([]ResourceRow, error) {
	cmd := exec.Command("cibadmin", "-Ql")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var cib CIB
	if err := xml.Unmarshal(out, &cib); err != nil {
		return nil, err
	}

	var rows []ResourceRow
	for _, resource := range cib.Configuration.Primitives {
		status := "Unknown"
		role := "Unknown"
		for _, meta_attribute := range resource.MetaAttributes.NVPairs {
			// FIXME (low-prio): status and role are excessive.
			if meta_attribute.Name == "target-role" {
				role = meta_attribute.Value
				// if the status is "Maintenance Mode" don't do anything
				if status == "Unknown" {
					if role == "Started" {
						status = "Online"
					}
					if role == "Stopped" {
						status = "Offline"
					}
				}
			}
			if meta_attribute.Name == "maintenance" {
				if meta_attribute.Value == "true" {
					status = "Maintenance Mode"
				}
			}
		}
		constraints := getResourceConstraints(resource.ID, cib.Configuration)
		node := getResourceRunningNode(resource.ID, cib.Status.NodeStates)
		events := getResourceEvents(resource.ID, cib.Status.NodeStates)
		rows = append(rows, ResourceRow{
			ID:             resource.ID,
			Class:          resource.Class,
			Provider:       resource.Provider,
			Type:           resource.Type,
			Node:           node,
			Status:         status,
			TargetRole:     role,
			Constraints:    constraints,
			MetaAttributes: resource.MetaAttributes.NVPairs,
			Events:         events,
			Utilizations:   resource.Utilizations,
		})
	}

	return rows, nil
}

func GetCIBNodes() ([]Node, error) {
	cmd := exec.Command("cibadmin", "-Ql")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var cib CIB
	if err := xml.Unmarshal(out, &cib); err != nil {
		return nil, err
	}

	return cib.Configuration.Nodes, nil
}

/*****************************
 * default meta attributes
 *****************************/

// copied from hawk -> tableless.rb --> RSC_DEFAULTS
// TODO: consider using hash-map Name -> Longdesc,Content
var rscDefaults = []MetaParameter{
	{
		Name:     "allow-migrate",
		Longdesc: "Set to true if the resource agent supports the migrate action",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	{
		Name:     "is-managed",
		Longdesc: "Is the cluster allowed to start and stop the resource?",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "true",
		},
	},
	{
		Name:     "maintenance",
		Longdesc: "Resources in maintenance mode are not monitored by the cluster.",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	{
		Name:     "migration-threshold",
		Longdesc: "How many failures may occur for this resource on a node before it's marked ineligible...",
		Content: ContentAttr{
			Type:    "integer",
			Default: "0",
		},
	},
	{
		Name:     "priority",
		Longdesc: "If not all resources can be active, lower priority ones will be stopped first.",
		Content: ContentAttr{
			Type:    "integer",
			Default: "0",
		},
	},
	{
		Name:     "multiple-active",
		Longdesc: "What should the cluster do if it finds the resource active on more than one node?",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "stop_start",
			PossibleValues: []string{"block", "stop_only", "stop_start"},
		},
	},
	{
		Name:     "failure-timeout",
		Longdesc: "Time to wait before considering the failure 'expired'.",
		Content: ContentAttr{
			Type:    "integer",
			Default: "0",
		},
	},
	{
		Name:     "resource-stickiness",
		Longdesc: "How much does the resource prefer to stay where it is?",
		Content: ContentAttr{
			Type:    "integer",
			Default: "0",
		},
	},
	{
		Name:     "target-role",
		Longdesc: "What state should the cluster try to maintain for this resource?",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "Stopped",
			PossibleValues: []string{"Started", "Stopped", "Master"},
		},
	},
	{
		Name: "restart-type",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "ignore",
			PossibleValues: []string{"ignore", "restart"},
		},
	},
	{
		Name: "description",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	{
		Name:     "requires",
		Longdesc: "Conditions required to start the resource.",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "fencing",
			PossibleValues: []string{"nothing", "quorum", "fencing"},
		},
	},
	{
		Name:     "remote-node",
		Longdesc: "The name of the remote-node this resource defines.",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	{
		Name:     "remote-port",
		Longdesc: "Port used for the guest connection.",
		Content: ContentAttr{
			Type:    "integer",
			Default: "3121",
		},
	},
	{
		Name:     "remote-addr",
		Longdesc: "The IP address or hostname for remote-node connection.",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	{
		Name:     "remote-connect-timeout",
		Longdesc: "Timeout before a pending guest connection fails.",
		Content: ContentAttr{
			Type:    "string",
			Default: "60s",
		},
	},
}

// copied from hawk -> tableless.rb --> OP_DEFAULTS
// TODO: consider using hash-map Name -> Longdesc,Content
var opDefaults = []MetaParameter{
	{
		Name:     "interval",
		Longdesc: "How frequently(in seconds) to perform the operation.",
		Content: ContentAttr{
			Type:     "string",
			Default:  "0",
			Required: "false",
		},
	},
	{
		Name:     "timeout",
		Longdesc: "How long to wait before declaring the action has failed.",
		Content: ContentAttr{
			Type:     "string",
			Default:  "20",
			Required: "true",
		},
	},
	{
		Name:     "requires",
		Longdesc: "What conditions need to be satisfied before this action occurs.",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "fencing",
			PossibleValues: []string{"nothing", "quorum", "fencing"},
		},
	},
	{
		Name:     "enabled",
		Longdesc: "If false, the operation is treated as if it does not exist.",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "true",
		},
	},
	{
		Name:     "role",
		Longdesc: "This option only makes sense for recurring operations. It restricts the operation to a specific role. The truly paranoid can even specify role=Stopped which allows the cluster to detect an admin that manually started cluster services.",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "",
			PossibleValues: []string{"Stopped", "Started", "Slave", "Master"},
		},
	},
	{
		Name:     "on-fail",
		Longdesc: "The action to take if this action ever fails.",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "stop",
			PossibleValues: []string{"ignore", "block", "stop", "restart", "standby", "fence"},
		},
	},
	{
		Name:     "start-delay",
		Longdesc: "The delay time(in seconds) before doing the operation",
		Content: ContentAttr{
			Type:    "string",
			Default: "0",
		},
	},
	{
		Name:     "interval-origin",
		Longdesc: "The start time of action interval. Follow the ISO8601 standard.",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	{
		Name:     "record-pending",
		Longdesc: "If true, the intention to perform the operation is recorded so that GUIs and CLI tools can indicate that an operation is in progress.",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	{
		Name: "description",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
}

// opDescriptions comes from hawk/app/models/template.rb
// TODO: consider using hash-map Name -> Description
var opDescriptions = []MetaParameter{{
	Name:      "template",
	Shortdesc: "Template",
	Longdesc:  "Resource template to inherit from.",
}, {
	Name:      "clazz",
	Shortdesc: "Template",
	Longdesc:  "Resource template to inherit from.",
}, {
	Name:      "provider",
	Shortdesc: "Provider",
	Longdesc:  "Vendor or project which provided the resource agent.",
}, {
	Name:      "type",
	Shortdesc: "Type",
	Longdesc:  "Resource agent name.",
}, {
	Name:      "op-start",
	Shortdesc: "Start",
	Longdesc:  "After the specified timeout period, the operation will be treated as failed.",
}, {
	Name:      "op-stop",
	Shortdesc: "Stop",
	Longdesc:  "After the specified timeout period, the operation will be treated as failed.",
}, {
	Name:      "op-monitor",
	Shortdesc: "Monitor",
	Longdesc:  "Define a monitor operation to instruct the cluster to ensure that the resource is still healthy.",
},
}

func GetRscDefaults() []MetaParameter {
	// return a copy to prevent modification
	result := make([]MetaParameter, len(rscDefaults))
	copy(result, rscDefaults)
	return result
}

func GetOpDefaults() []MetaParameter {
	// return a copy to prevent modification
	result := make([]MetaParameter, len(opDefaults))
	copy(result, opDefaults)
	return result
}

func GetOpDescriptions() []MetaParameter {
	// return a copy to prevent modification
	result := make([]MetaParameter, len(opDescriptions))
	copy(result, opDescriptions)
	return result
}

/***************************************************
 * crm_resource --show-metadata ocf:pacemaker:Dummy
 ***************************************************/

type CrmResourceMetadata struct {
	Name       string          `xml:"name,attr"`
	Version    string          `xml:"version,attr"`
	Longdesc   string          `xml:"longdesc"`
	Shortdesc  string          `xml:"shortdesc"`
	Parameters []MetaParameter `xml:"parameters>parameter"` // maps to instance_attributes
	Actions    []Action        `xml:"actions>action"`
	/* RscDefaults (#meta_attributes) is not in 'crm_resource --show-metadata'
	 * but it's copied from rscDefaults
	 * and later enriched from 'cibadmin' */
	RscDefaults []MetaParameter
}

type MetaParameter struct {
	Name      string      `xml:"name,attr"`
	Longdesc  string      `xml:"longdesc"`
	Shortdesc string      `xml:"shortdesc"`
	Content   ContentAttr `xml:"content"`
}

type ContentAttr struct {
	Type    string `xml:"type,attr"`
	Default string `xml:"default,attr"`
	// Possible values are hardcoded
	PossibleValues []string
	// We take CibID and CibValue later from cib, if they are defined
	Required string // string, so that ["true", "false", "" for undefined]
	CibID    string // "" in case of operation attributes, the Action.CibID is used instead
	CibValue string
}

/* TODO: Action struct is messy. It's used for both to parse cib.xml
 * and to store the default values of operations.
 * Maybe there should be two different structures
 * (however I might change my mind, so don't hastle with it (17.05.2025))*/
type Action struct {
	Depth          string `xml:"depth,attr,omitempty"`
	Description    string `xml:"description,attr,omitempty"`
	Enabled        string `xml:"enabled,attr,omitempty"`
	Interval       string `xml:"interval,attr,omitempty"`
	IntervalOrigin string `xml:"interval-origin,attr,omitempty"`
	OnFail         string `xml:"on-fail,attr,omitempty"`
	Name           string `xml:"name,attr"`
	RecordPending  string `xml:"record-pending,attr,omitempty"`
	Requires       string `xml:"requires,attr,omitempty"`
	Role           string `xml:"role,attr,omitempty"`
	StartDelay     string `xml:"start-delay,attr,omitempty"`
	Timeout        string `xml:"timeout,attr,omitempty"`
	// We take CibID later from cib, if they are defined
	CibID string
	// Default values
	OpDefaults []MetaParameter
	// Help info
	Shortdesc string
	Longdesc  string
}

func getResourceMetadata(resourceAgent string) (CrmResourceMetadata, error) {
	//var cmd *exec.Cmd
	cmd := exec.Command("crm_resource", "--show-metadata", resourceAgent)

	out, err := cmd.Output()
	if err != nil {
		return CrmResourceMetadata{}, err
	}

	var metadata CrmResourceMetadata // Directly unmarshal into this
	if err := xml.Unmarshal(out, &metadata); err != nil {
		return CrmResourceMetadata{}, err
	}

	// Additional handling for stonith agents
	if strings.HasPrefix(resourceAgent, "stonith:") {

		stonithPaths := []string{
			"/usr/libexec/pacemaker/pacemaker-fenced",
			"/usr/lib/pacemaker/pacemaker-fenced",
		}

		var stonithOut []byte
		var stonithErr error

		for _, p := range stonithPaths {
			cmd = exec.Command(p, "metadata")
			stonithOut, stonithErr = cmd.Output()
			if stonithErr == nil {
				break // Success → stop trying
			}
		}

		if stonithErr != nil {
			log.Printf("warning: failed to fetch stonith metadata: %v", stonithErr)
			return metadata, stonithErr
		}

		var stonithMetadata CrmResourceMetadata
		if err := xml.Unmarshal(stonithOut, &stonithMetadata); err != nil {
			return CrmResourceMetadata{}, err
		}

		// merge stonith_metadata into metadata
		metadata.Parameters = append(metadata.Parameters, stonithMetadata.Parameters...)
	}

	return metadata, nil
}

func enrichMetadataWithCibValues(metadata *CrmResourceMetadata, resourceID string) error {
	// 1. Query current XML
	queryXPath := fmt.Sprintf("/cib/configuration/resources//primitive[@id='%s']", resourceID)
	cmd := exec.Command("cibadmin", "-Q", "--xpath", queryXPath)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[enrichMetadataWithCibValues] cibadmin -Q error: %v", err)
		return err
	}

	// 2. Unmarshal to struct
	var primitive Primitive
	if err := xml.Unmarshal(out, &primitive); err != nil {
		log.Printf("[enrichMetadataWithCibValues] XML unmarshal error: %v", err)
		return err
	}

	for _, nv := range primitive.InstanceAttributes.NVPairs {
		// search the parameter in InstanceAttributes
		for i := range metadata.Parameters {
			if nv.Name == metadata.Parameters[i].Name {
				metadata.Parameters[i].Content.CibID = nv.ID
				metadata.Parameters[i].Content.CibValue = nv.Value
			}
		}
	}
	for _, nv := range primitive.MetaAttributes.NVPairs {
		// search the parameter in MetaAttributes
		for i := range metadata.RscDefaults {
			if nv.Name == metadata.RscDefaults[i].Name {
				metadata.RscDefaults[i].Content.CibID = nv.ID
				metadata.RscDefaults[i].Content.CibValue = nv.Value
			}
		}
	}
	for _, op := range primitive.Operations {
		// search action in Operations
		for i := range metadata.Actions {
			if op.Name != metadata.Actions[i].Name {
				continue
			}
			metadata.Actions[i].CibID = op.ID
			for j := range metadata.Actions[i].OpDefaults {
				switch metadata.Actions[i].OpDefaults[j].Name {
				case "depth":
					metadata.Actions[i].OpDefaults[j].Content.CibValue = op.Depth
				case "description":
					metadata.Actions[i].OpDefaults[j].Content.CibValue = op.Description
				case "enabled":
					metadata.Actions[i].OpDefaults[j].Content.CibValue = op.Enabled
				case "interval":
					metadata.Actions[i].OpDefaults[j].Content.CibValue = op.Interval
				case "interval-origin":
					metadata.Actions[i].OpDefaults[j].Content.CibValue = op.IntervalOrigin
				case "on-fail":
					metadata.Actions[i].OpDefaults[j].Content.CibValue = op.OnFail
				case "record-pending":
					metadata.Actions[i].OpDefaults[j].Content.CibValue = op.RecordPending
				case "requires":
					metadata.Actions[i].OpDefaults[j].Content.CibValue = op.Requires
				case "role":
					metadata.Actions[i].OpDefaults[j].Content.CibValue = op.Role
				case "start-delay":
					metadata.Actions[i].OpDefaults[j].Content.CibValue = op.StartDelay
				case "timeout":
					metadata.Actions[i].OpDefaults[j].Content.CibValue = op.Timeout

				}
			}
		}
	}

	return nil
}

// This function does the magic routing between Go and Ruby
func ResourceEditHandler(w http.ResponseWriter, r *http.Request) {
	const prefix = "/cib/live/primitives"

	// Normalize (collapse //, removes trailing /)
	cleanPath := path.Clean(r.URL.EscapedPath())

	// must be either exactly the prefix or start with prefix + "/"
	if cleanPath != prefix && !strings.HasPrefix(cleanPath, prefix+"/") {
		http.NotFound(w, r)
		return
	}

	// pre-parsing
	cleanPath = strings.TrimSuffix(cleanPath, "/")    // drop ending /
	cleanPath = strings.TrimPrefix(cleanPath, prefix) // drop prefix
	cleanPath = strings.TrimPrefix(cleanPath, "/")    // drop the leading slash

	// "{id}/edit" --> handle here
	if strings.HasSuffix(cleanPath, "/edit") {
		resourceID := strings.TrimSuffix(cleanPath, "/edit")

		// make sure its {id}, not {id1}/{id2}/...
		if resourceID == "" || strings.Contains(resourceID, "/") {
			http.NotFound(w, r)
			return
		}

		crm, err := GetCIBResources()
		if err != nil {
			http.Error(w, "[ResourceEditHandler] Failed to get CRM resource status: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var resourceRow ResourceRow
		found := false
		for _, rsrc := range crm {
			if rsrc.ID == resourceID {
				resourceRow = rsrc
				found = true
				break
			}
		}
		if !found {
			http.Error(w, "Resource not found", http.StatusNotFound)
			return
		}

		resourceAgent := resourceRow.Class
		if resourceRow.Provider != "" {
			resourceAgent += ":" + resourceRow.Provider
		}
		resourceAgent += ":" + resourceRow.Type

		/* If we do Configuration -> Add Resource -> Primitive -> Create
		 * It would redirect to the cib/live/primitives/{primitive-id}/edit?flash={created|updated}
		 */
		flash := r.URL.Query().Get("flash")
		var alertType, alertMsg string

		switch flash {
		case "created":
			alertType = "success"
			alertMsg = "Primitive created successfully"
		case "updated":
			alertType = "success"
			alertMsg = "Primitive updated successfully"
		case "renamed":
			alertType = "success"
			alertMsg = "Primitive renamed successfully"
		case "error":
			alertType = "danger"
			alertMsg = r.URL.Query().Get("msg")
			if alertMsg == "" {
				alertMsg = "There was an error processing the primitive."
			}
		}

		renderTemplate(w, "primitive_edit", map[string]any{
			"Title":         "Edit Primitive",
			"ResourceID":    resourceID,
			"Class":         resourceRow.Class,
			"Provider":      resourceRow.Provider,
			"Type":          resourceRow.Type,
			"ResourceAgent": resourceAgent,
			"AlertType":     alertType,
			"AlertMessage":  alertMsg,
		})
		return
	}

	// else --> Ruby
	if Routehandler != nil {
		Routehandler.ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}

func NodesEditHandler(w http.ResponseWriter, r *http.Request) {
	const prefix = "/cib/live/nodes"

	// Normalize (collapse //, removes trailing /)
	cleanPath := path.Clean(r.URL.EscapedPath())

	// must be either exactly the prefix or start with prefix + "/"
	if cleanPath != prefix && !strings.HasPrefix(cleanPath, prefix+"/") {
		http.NotFound(w, r)
		return
	}

	// pre-parsing
	cleanPath = strings.TrimSuffix(cleanPath, "/")    // drop ending /
	cleanPath = strings.TrimPrefix(cleanPath, prefix) // drop prefix
	cleanPath = strings.TrimPrefix(cleanPath, "/")    // drop the leading slash

	// "{id}/edit" --> handle here
	if strings.HasSuffix(cleanPath, "/edit") {
		nodeID := strings.TrimSuffix(cleanPath, "/edit")

		// make sure its {id}, not {id1}/{id2}/...
		if nodeID == "" || strings.Contains(nodeID, "/") {
			http.NotFound(w, r)
			return
		}

		nodes, err := GetCIBNodes()
		if err != nil {
			http.Error(w, "[NodesEditHandler] Failed to get nodes in 'cibadmin -Ql': "+err.Error(), http.StatusInternalServerError)
			return
		}

		var thisNode Node
		thisNodeFound := false

		for _, node := range nodes {
			if node.ID == nodeID {
				thisNode = node
				thisNodeFound = true
			}
		}

		if thisNodeFound == false {
			http.Error(w, "[NodesEditHandler] Failed to find nodes with ID "+nodeID, http.StatusInternalServerError)
			return
		}

		/* If we do Configuration -> Add Resource -> Primitive -> Create
		 * It would redirect to the cib/live/primitives/{primitive-id}/edit?flash={created|updated}
		 */
		flash := r.URL.Query().Get("flash")
		var alertType, alertMsg string

		switch flash {
		case "created":
			alertType = "success"
			alertMsg = "Node created successfully"
		case "updated":
			alertType = "success"
			alertMsg = "Node updated successfully"
		case "renamed":
			alertType = "success"
			alertMsg = "Node renamed successfully"
		case "error":
			alertType = "danger"
			alertMsg = r.URL.Query().Get("msg")
			if alertMsg == "" {
				alertMsg = "There was an error processing the primitive."
			}
		}

		renderTemplate(w, "node_edit", map[string]any{
			"Title":        "Edit Node",
			"NodeName":     thisNode.Uname,
			"NodeID":       nodeID,
			"AlertType":    alertType,
			"AlertMessage": alertMsg,
		})
		return
	}

	// else --> Ruby
	if Routehandler != nil {
		Routehandler.ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}

// FIXME (low-prio): it's 90% the same as updateNvpair
func updateOperation(operation Operation, resourceID string) error {
	xmlBytes, err := xml.Marshal(operation)
	if err != nil {
		log.Printf("[updateCibNvpair] XML marshal error: %v", err)
		return err
	}
	xmlStr := string(xmlBytes)
	xmlStr = fmt.Sprintf("<primitive id=\"%s\"><operations>%s</operations></primitive>", resourceID, xmlStr)

	queryXPath := fmt.Sprintf("//primitive[@id='%s']", resourceID)

	var stderr bytes.Buffer
	cmd := exec.Command("cibadmin", "--modify", "--xpath", queryXPath, "--xml-text", xmlStr)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}

func deleteOperation(opID string, resourceID string, removeParent bool) ([]byte, error) {
	var queryXPath string
	if removeParent {
		queryXPath = fmt.Sprintf("//primitive[@id='%s']/operations", resourceID)
	} else {
		queryXPath = fmt.Sprintf("//primitive[@id='%s']/operations/op[@id='%s']", resourceID, opID)
	}
	cmd := exec.Command("cibadmin", "--delete", "--xpath", queryXPath)
	return cmd.CombinedOutput()
}

func updateNvpair(nvpair Nvpair, section string, resourceID string) ([]byte, error) {
	xmlBytes, err := xml.Marshal(nvpair)
	if err != nil {
		log.Printf("[updateCibNvpair] XML marshal error: %v", err)
		return xmlBytes, err
	}
	xmlStr := string(xmlBytes)
	xmlStr = fmt.Sprintf("<primitive id=\"%s\"><%s id=\"%s-%s\">%s</%s></primitive>", resourceID, section, resourceID, section, xmlStr, section)

	queryXPath := fmt.Sprintf("//primitive[@id='%s']", resourceID)
	cmd := exec.Command("cibadmin", "--modify", "--xpath", queryXPath, "--xml-text", xmlStr)
	/* TODO!!! if it fails, check that the id is unique.
	     * I have noticed a bug that id might start with a wrong primitive name like here
		<resources>
	      <primitive id="stonith-sbd" class="stonith" type="fence_sbd"/>
	      <primitive id="dummyH" class="ocf" provider="pacemaker" type="Dummy">
	        <instance_attributes id="dummy1-instance_attributes"/>
	        <meta_attributes id="dummy1-meta_attributes"/>
	        <operations>
	          <op id="dummy1-monitor-10" interval="10" name="monitor" timeout="20"/>      <---- dummy1 (WHY?)
	          <op id="dummyH-monitor-10" interval="10" name="monitor" timeout="20"/>      <---- dummyH (correct)
	          <op id="dummyH-meta-data-5" interval="5" name="meta-data" timeout="10"/>
	          <op id="dummyH-monitor-11" interval="11" name="monitor" timeout="20"/>
	        </operations>
	        <instance_attributes id="dummyH-instance_attributes">
	          <nvpair id="dummyH-instance_attributes-envfile" name="envfile" value="qwe"/>
	        </instance_attributes>
	        <meta_attributes id="dummyH-meta_attributes">
	          <nvpair id="dummyH-meta_attributes-allow-migrate" name="allow-migrate" value="false"/>
	          <nvpair id="dummyH-meta_attributes-failure-timeout" name="failure-timeout" value="0"/>
	          <nvpair id="dummyH-meta_attributes-target-role" name="target-role" value="Stopped"/>
	        </meta_attributes>
	      </primitive>
	      <primitive id="dummy1" class="ocf" provider="pacemaker" type="Dummy"/>
	    </resources>
	*/
	return cmd.CombinedOutput()
}

func deleteNvpair(cibAttributeID string, section string, resourceID string, removeParent bool) ([]byte, error) {
	var queryXPath string
	if removeParent {
		queryXPath = fmt.Sprintf("//instance_attributes[@id='%s-%s']", resourceID, section)
	} else {
		queryXPath = fmt.Sprintf("//nvpair[@id='%s']", cibAttributeID)
	}
	cmd := exec.Command("cibadmin", "--delete", "--xpath", queryXPath)
	return cmd.CombinedOutput()
}

func applyAttributes(cibAttributes []Nvpair, frontendAttributes []Nvpair, primitiveID string, section string, w http.ResponseWriter) {
	// cibAttributes - what exists
	// frontendPrimitives - what should be

	// case: Remove, (attribute exists in cib, but not in frontend)
	attributesExist := len(cibAttributes)
	for i := range cibAttributes {
		var nvpairExistsInFrontend bool = false
		for _, frontendNvpair := range frontendAttributes {
			if cibAttributes[i].ID == frontendNvpair.ID {
				nvpairExistsInFrontend = true
				break
			}
		}
		if !nvpairExistsInFrontend {
			// if there is only 1 nvpair left --> remove it together with <instance_attributes ...>
			_, err := deleteNvpair(cibAttributes[i].ID, section, primitiveID, attributesExist <= 1)
			attributesExist--
			if err != nil {
				http.Error(w, "Failed to encode updated XML", http.StatusInternalServerError)
				log.Printf("[setPrimitive] XML marshal error: %v", err)
				return
			}
		}
	}

	// case: Add + Update
	for _, frontendNvpair := range frontendAttributes {
		var nvpairExistsInCib bool = false
		var nvpairNeedsCibUpdate bool = true
		var newNvpair Nvpair
		for i := range cibAttributes {
			if cibAttributes[i].ID == frontendNvpair.ID {
				nvpairExistsInCib = true
				// if the value hasn't changed, don't do anything
				if cibAttributes[i].Value == frontendNvpair.Value {
					nvpairNeedsCibUpdate = false // to break from the outer loop
					break
				}
				// otherwise --> update it
				cibAttributes[i].Value = frontendNvpair.Value
				newNvpair = cibAttributes[i]
				break
			}
		}
		if nvpairExistsInCib && !nvpairNeedsCibUpdate { // go to the next changed field
			continue
		}
		if !nvpairExistsInCib { // if the nvpair doesn't exist in cib --> create it
			newNvpair = Nvpair{ID: primitiveID + "-" + section + "-" + frontendNvpair.Name, Name: frontendNvpair.Name, Value: frontendNvpair.Value}
		}
		_, err := updateNvpair(newNvpair, section, primitiveID)
		if err != nil {
			http.Error(w, "Failed to execute cibadmin --update", http.StatusInternalServerError)
			log.Printf("[setPrimitive] cibadmin --update error: %v", err)
			return
		}
	}
}

func PrimitiveCreateHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: before creating the primitive try creating it in the shadow-cib
	var frontendPrimitive Primitive

	if err := json.NewDecoder(r.Body).Decode(&frontendPrimitive); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveCreateHandler] JSON decode error: %v", err)
		return
	}

	log.Printf("Creating resource %s with fields: %+v\n", frontendPrimitive.ID, frontendPrimitive)

	raName := frontendPrimitive.Class + ":"
	if frontendPrimitive.Provider != "" {
		raName += frontendPrimitive.Provider + ":"
	}
	raName += frontendPrimitive.Type

	args := []string{"configure", "primitive", frontendPrimitive.ID, raName}

	// Parameters
	for _, nvpair := range frontendPrimitive.InstanceAttributes.NVPairs {
		args = append(args, fmt.Sprintf("%s=%s", nvpair.Name, nvpair.Value))
	}

	// Operations
	for _, op := range frontendPrimitive.Operations {
		args = append(args, "op", op.Name)
		if op.Depth != "" {
			args = append(args, "depth="+op.Depth)
		}
		if op.Description != "" {
			args = append(args, "description="+op.Description)
		}
		if op.Enabled != "" {
			args = append(args, "enabled="+op.Enabled)
		}
		if op.Interval != "" {
			args = append(args, "interval="+op.Interval)
		}
		if op.IntervalOrigin != "" {
			args = append(args, "interval-origin-="+op.IntervalOrigin)
		}
		if op.OnFail != "" {
			args = append(args, "on-fail="+op.OnFail)
		}
		if op.RecordPending != "" {
			args = append(args, "record-pending="+op.RecordPending)
		}
		if op.Requires != "" {
			args = append(args, "requires="+op.Requires)
		}
		if op.Role != "" {
			args = append(args, "role="+op.Role)
		}
		if op.StartDelay != "" {
			args = append(args, "start-delay="+op.StartDelay)
		}
		if op.Timeout != "" {
			args = append(args, "timeout="+op.Timeout)
		}
	}

	// Meta Attributes
	metaStarted := false
	for _, nvpair := range frontendPrimitive.MetaAttributes.NVPairs {
		// skip empty values like target-role="" (which happens in the test_copy_primitive)
		if nvpair.Value == "" {
			continue
		}
		if !metaStarted {
			args = append(args, "meta")
			metaStarted = true
		}
		args = append(args, fmt.Sprintf("%s=%s", nvpair.Name, nvpair.Value))
	}

	cmd := exec.Command("/usr/sbin/crm", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_, err := cmd.Output()
	if err != nil {
		http.Error(w, stderr.String(), http.StatusInternalServerError)
		log.Printf("[PrimitiveCreateHandler] crm conf primitive %s ... : %v", frontendPrimitive.ID, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("Created %s", frontendPrimitive.ID),
	})
}

func PrimitiveUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var frontendPrimitive Primitive

	if err := json.NewDecoder(r.Body).Decode(&frontendPrimitive); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveUpdateHandler] JSON decode error: %v", err)
		return
	}

	log.Printf("Updating resource %s with fields: %+v\n", frontendPrimitive.ID, frontendPrimitive)

	// 1. Query current XML
	queryXPath := fmt.Sprintf("//primitive[@id='%s']", frontendPrimitive.ID)
	cmd := exec.Command("cibadmin", "-Q", "--xpath", queryXPath)
	out, err := cmd.Output()
	if err != nil {
		http.Error(w, "Failed to query primitive XML", http.StatusInternalServerError)
		log.Printf("[PrimitiveUpdateHandler] cibadmin -Q error: %v", err)
		return
	}

	// 2. Unmarshal to struct
	var cibPrimitive Primitive
	if err := xml.Unmarshal(out, &cibPrimitive); err != nil {
		http.Error(w, "Failed to parse primitive XML", http.StatusInternalServerError)
		log.Printf("[PrimitiveUpdateHandler] XML unmarshal error: %v", err)
		return
	}

	// 3. Apply instance_attributes
	applyAttributes(cibPrimitive.InstanceAttributes.NVPairs, frontendPrimitive.InstanceAttributes.NVPairs,
		frontendPrimitive.ID, "instance_attributes", w)

	// 4. Apply meta_attributes
	applyAttributes(cibPrimitive.MetaAttributes.NVPairs, frontendPrimitive.MetaAttributes.NVPairs,
		frontendPrimitive.ID, "meta_attributes", w)

	// 5. Apply operations. (TODO: it repeats the SubmitResourceOperations)
	for _, frontendOp := range frontendPrimitive.Operations {
		var opExists bool = false
		var opUpdated bool = true
		var newOp Operation
		opUpdated = false
		for i := range cibPrimitive.Operations {
			if cibPrimitive.Operations[i].ID == frontendOp.ID {
				opExists = true
				if cibPrimitive.Operations[i].Depth != frontendOp.Depth {
					cibPrimitive.Operations[i].Depth = frontendOp.Depth
					opUpdated = true
				}
				if cibPrimitive.Operations[i].Description != frontendOp.Description {
					cibPrimitive.Operations[i].Description = frontendOp.Description
					opUpdated = true
				}
				if cibPrimitive.Operations[i].Enabled != frontendOp.Enabled {
					cibPrimitive.Operations[i].Enabled = frontendOp.Enabled
					opUpdated = true
				}
				if cibPrimitive.Operations[i].Interval != frontendOp.Interval {
					cibPrimitive.Operations[i].Interval = frontendOp.Interval
					opUpdated = true
				}
				if cibPrimitive.Operations[i].IntervalOrigin != frontendOp.IntervalOrigin {
					cibPrimitive.Operations[i].IntervalOrigin = frontendOp.IntervalOrigin
					opUpdated = true
				}
				if cibPrimitive.Operations[i].OnFail != frontendOp.OnFail {
					cibPrimitive.Operations[i].OnFail = frontendOp.OnFail
					opUpdated = true
				}
				if cibPrimitive.Operations[i].RecordPending != frontendOp.RecordPending {
					cibPrimitive.Operations[i].RecordPending = frontendOp.RecordPending
					opUpdated = true
				}
				if cibPrimitive.Operations[i].Requires != frontendOp.Requires {
					cibPrimitive.Operations[i].Requires = frontendOp.Requires
					opUpdated = true
				}
				if cibPrimitive.Operations[i].Role != frontendOp.Role {
					cibPrimitive.Operations[i].Role = frontendOp.Role
					opUpdated = true
				}
				if cibPrimitive.Operations[i].StartDelay != frontendOp.StartDelay {
					cibPrimitive.Operations[i].StartDelay = frontendOp.StartDelay
					opUpdated = true
				}
				if cibPrimitive.Operations[i].Timeout != frontendOp.Timeout {
					cibPrimitive.Operations[i].Timeout = frontendOp.Timeout
					opUpdated = true
				}

				newOp = cibPrimitive.Operations[i]
				break
			}
		}
		if opExists && !opUpdated { // go to the next changed field
			continue
		}
		if !opExists { // if the op doesn't exist in cib --> create it
			newOp = Operation{ID: frontendPrimitive.ID + "-" + frontendOp.Name + "-" + frontendOp.Interval,
				Depth:          frontendOp.Depth,
				Description:    frontendOp.Description,
				Enabled:        frontendOp.Enabled,
				Interval:       frontendOp.Interval,
				IntervalOrigin: frontendOp.IntervalOrigin,
				OnFail:         frontendOp.OnFail,
				Name:           frontendOp.Name,
				RecordPending:  frontendOp.RecordPending,
				Requires:       frontendOp.Requires,
				Role:           frontendOp.Role,
				StartDelay:     frontendOp.StartDelay,
				Timeout:        frontendOp.Timeout,
			}
		}
		err = updateOperation(newOp, frontendPrimitive.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("[PrimitiveUpdateHandler] XML marshal error: %v", err)
			return
		}
	}

	// 6. Success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("Updated %s", frontendPrimitive.ID),
	})
}

func PrimitiveRenameHandler(w http.ResponseWriter, r *http.Request) {
	var renameID struct {
		OldID string `json:"oldID"`
		NewID string `json:"newID"`
	}

	if err := json.NewDecoder(r.Body).Decode(&renameID); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveRenameHandler] JSON decode error: %v", err)
		return
	}

	cmd := exec.Command("/usr/sbin/crm", "-D", "plain", "configure", "rename", renameID.OldID, renameID.NewID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_, err := cmd.Output()
	if err != nil {
		http.Error(w, stripANSI(stderr.String()), http.StatusInternalServerError)
		log.Printf("[PrimitiveRenameHandler] crm -D plain configure rename error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("%s renamed into %s", renameID.OldID, renameID.NewID),
	})
}

func PrimitiveDeleteHandler(w http.ResponseWriter, r *http.Request) {
	var ResourceID string

	if err := json.NewDecoder(r.Body).Decode(&ResourceID); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveDeleteHandler] JSON decode error: %v", err)
		return
	}

	cmd := exec.Command("/usr/sbin/crm", "--force", "configure", "delete", ResourceID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_, err := cmd.Output()
	if err != nil {
		http.Error(w, stderr.String(), http.StatusInternalServerError)
		log.Printf("[PrimitiveDeleteHandler] crm --force configure delete %s error: %v", ResourceID, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("Primitive %s deleted", ResourceID),
	})
}

func getRaAgents(raClass string, raProvider string) ([]string, error) {
	var cmd *exec.Cmd
	// when stonith or systemd classes --> raProvider is empty
	cmd = exec.Command("/usr/sbin/crm", "ra", "list", raClass, raProvider)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		log.Printf("[getRaAgents] crm ra list: %v", err)
		return nil, err
	}

	agents := strings.Fields(string(out))

	return agents, nil
}

var cachedRaClasses map[string]map[string][]string
var raClassesFetched bool

func RaClassesHandler(w http.ResponseWriter, r *http.Request) {
	if raClassesFetched {
		/* crm ra classes is too slow,
		 * return the cached result if exists.
		 * TODO: implement the cache update */
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"RaClasses": cachedRaClasses})
		return
	}

	cmd := exec.Command("/usr/sbin/crm", "ra", "classes")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		http.Error(w, stderr.String(), http.StatusInternalServerError)
		log.Printf("[RaClassesHandler] crm ra classes: %v", err)
		return
	}

	// Split output into lines and remove empty ones
	lines := strings.Split(string(out), "\n")
	raClasses := make(map[string]map[string][]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by "/" and keep both parts
		parts := strings.SplitN(line, "/", 2)
		raClass := strings.TrimSpace(parts[0])
		raClasses[raClass] = make(map[string][]string)

		if len(parts) > 1 { // ocf
			providersList := strings.Fields(parts[1])
			for _, providerName := range providersList { // heartbeat, pacemaker, suse
				agents, err := getRaAgents(raClass, providerName)
				if err != nil {
					http.Error(w, stderr.String(), http.StatusInternalServerError)
					return
				}

				raClasses[raClass][providerName] = agents
			}
		} else { // stonith, systemd
			agents, err := getRaAgents(raClass, "")
			if err != nil {
				http.Error(w, stderr.String(), http.StatusInternalServerError)
				return
			}
			raClasses[raClass][""] = agents
		}
	}

	cachedRaClasses = raClasses
	raClassesFetched = true

	data := map[string]any{"RaClasses": raClasses}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
