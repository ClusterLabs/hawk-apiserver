package api

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
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

func GetCIB() (CIB, int, error) {
	cmd := exec.Command("cibadmin", "-Ql")
	output, err := cmd.Output()
	if err != nil {
		pacemakerRC := 0
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			pacemakerRC = exitErr.ExitCode()
		}

		return CIB{}, pacemakerRC, fmt.Errorf("cibadmin -Ql failed: %w", err)
	}

	var cib CIB
	if err := xml.Unmarshal(output, &cib); err != nil {
		return CIB{}, 0, err
	}

	return cib, 0, nil
}

type Configuration struct {
	CrmConfig   CrmConfig   `xml:"crm_config"`
	Nodes       []Node      `xml:"nodes>node"`
	Constraints Constraints `xml:"constraints"`
	Primitives  []Primitive `xml:"resources>primitive"`
	Clones      []Clone     `xml:"resources>clone"`
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

type Clone struct {
	ID             string        `xml:"id,attr" json:"id"`
	Primitives     []Primitive   `xml:"primitive" json:"primitive"`
	MetaAttributes MetaAttribute `xml:"meta_attributes" json:"meta_attributes"`
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
	ID                 string
	Class              string
	Provider           string
	Type               string
	Node               string
	Status             string
	TargetRole         string
	Constraints        Constraints
	InstanceAttributes []Nvpair
	MetaAttributes     []Nvpair
	Events             []LRMOp
	Utilizations       []Nvpair
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
	cib, _, err := GetCIB()
	if err != nil {
		return nil, err
	}

	resourceCount := len(cib.Configuration.Primitives)
	for _, clone := range cib.Configuration.Clones {
		resourceCount += len(clone.Primitives)
	}

	resources := make([]Primitive, 0, resourceCount)
	resources = append(resources, cib.Configuration.Primitives...)
	// it's not the clone itself, but the pritimitive that clone
	for _, clone := range cib.Configuration.Clones {
		resources = append(resources, clone.Primitives...)
	}

	rows := make([]ResourceRow, 0, resourceCount)
	for _, resource := range resources {
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
			ID:                 resource.ID,
			Class:              resource.Class,
			Provider:           resource.Provider,
			Type:               resource.Type,
			Node:               node,
			Status:             status,
			TargetRole:         role,
			Constraints:        constraints,
			InstanceAttributes: resource.InstanceAttributes.NVPairs,
			MetaAttributes:     resource.MetaAttributes.NVPairs,
			Events:             events,
			Utilizations:       resource.Utilizations,
		})
	}

	return rows, nil
}

func GetCIBNodes() ([]Node, error) {
	cib, _, err := GetCIB()
	if err != nil {
		return nil, err
	}

	return cib.Configuration.Nodes, nil
}

func firstActionsByName(actions []Action) []Action {
	if len(actions) < 2 {
		return actions
	}

	seen := make(map[string]bool, len(actions))
	uniqueActions := actions[:0]
	for _, action := range actions {
		if seen[action.Name] {
			continue
		}

		seen[action.Name] = true
		uniqueActions = append(uniqueActions, action)
	}

	return uniqueActions
}

func enrichCloneMetaAttributesWithCibValues(metadata *CrmResourceMetadata, cloneID string) error {
	// Query the clone itself because meta_attributes is optional. Querying the
	// child element directly makes cibadmin fail for valid clones without one.
	queryXPath := fmt.Sprintf("/cib/configuration/resources/clone[@id='%s']", cloneID)
	cmd := exec.Command("cibadmin", "-Q", "--xpath", queryXPath)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[enrichCloneMetaAttributesWithCibValues] cibadmin -Q error: %v", err)
		return err
	}

	var clone Clone
	if err := xml.Unmarshal(out, &clone); err != nil {
		log.Printf("[enrichCloneMetaAttributesWithCibValues] XML unmarshal error: %v", err)
		return err
	}

	for _, nv := range clone.MetaAttributes.NVPairs {
		// search the parameter in MetaAttributes
		for i := range metadata.RscDefaults {
			if nv.Name == metadata.RscDefaults[i].Name {
				metadata.RscDefaults[i].Content.CibID = nv.ID
				metadata.RscDefaults[i].Content.CibValue = nv.Value
			}
		}
	}

	return nil
}

func enrichPrimitiveMetadataWithCibValues(metadata *CrmResourceMetadata, resourceID string) error {
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
		found := false
		// First, search the parameter in the schema of InstanceAttributes
		for i := range metadata.Parameters {
			if nv.Name == metadata.Parameters[i].Name {
				found = true
				metadata.Parameters[i].Content.CibID = nv.ID
				metadata.Parameters[i].Content.CibValue = nv.Value
				break
			}
		}
		if found == false {
			// if not found --> extend the schema with the user parameter
			unknownCustomInstanceAttribute := MetaParameter{
				Name: nv.Name,
				Content: ContentAttr{
					CibID:    nv.ID,
					CibValue: nv.Value,
				},
			}
			metadata.Parameters = append(metadata.Parameters, unknownCustomInstanceAttribute)
		}
	}

	for _, nv := range primitive.MetaAttributes.NVPairs {
		found := false
		// First, search the parameter in the schema of MetaAttributes
		for i := range metadata.RscDefaults {
			if nv.Name == metadata.RscDefaults[i].Name {
				found = true
				metadata.RscDefaults[i].Content.CibID = nv.ID
				metadata.RscDefaults[i].Content.CibValue = nv.Value
				break
			}
		}
		if found == false {
			// if not found --> extend the schema with the user parameter
			unknownCustomMetaParameter := MetaParameter{
				Name: nv.Name,
				Content: ContentAttr{
					CibID:    nv.ID,
					CibValue: nv.Value,
				},
			}
			metadata.RscDefaults = append(metadata.RscDefaults, unknownCustomMetaParameter)
		}
	}

	for _, op := range primitive.Operations {
		found := false
		// First, search the action in the schema of Operations
		for i := range metadata.Actions {
			if op.Name == metadata.Actions[i].Name {
				found = true
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
				break
			}
		}
		if found == false {
			/* it's a bad practice to have unsupported custom operation
			 * but the user knows it better. If it exists we preserve it
			 * i.e. if not found --> extend the schema with the user action */
			unknownCustomAction := Action{
				Name:       op.Name,
				CibID:      op.ID,
				OpDefaults: GetOpDefaults(),
			}
			for j := range unknownCustomAction.OpDefaults {
				switch unknownCustomAction.OpDefaults[j].Name {
				case "depth":
					unknownCustomAction.OpDefaults[j].Content.CibValue = op.Depth
				case "description":
					unknownCustomAction.OpDefaults[j].Content.CibValue = op.Description
				case "enabled":
					unknownCustomAction.OpDefaults[j].Content.CibValue = op.Enabled
				case "interval":
					unknownCustomAction.OpDefaults[j].Content.CibValue = op.Interval
				case "interval-origin":
					unknownCustomAction.OpDefaults[j].Content.CibValue = op.IntervalOrigin
				case "on-fail":
					unknownCustomAction.OpDefaults[j].Content.CibValue = op.OnFail
				case "record-pending":
					unknownCustomAction.OpDefaults[j].Content.CibValue = op.RecordPending
				case "requires":
					unknownCustomAction.OpDefaults[j].Content.CibValue = op.Requires
				case "role":
					unknownCustomAction.OpDefaults[j].Content.CibValue = op.Role
				case "start-delay":
					unknownCustomAction.OpDefaults[j].Content.CibValue = op.StartDelay
				case "timeout":
					unknownCustomAction.OpDefaults[j].Content.CibValue = op.Timeout
				}
			}
			metadata.Actions = append(metadata.Actions, unknownCustomAction)
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
				alertMsg = "There was an error processing the clone."
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

func CloneEditHandler(w http.ResponseWriter, r *http.Request) {
	const prefix = "/cib/live/clones"

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
		cloneID := strings.TrimSuffix(cleanPath, "/edit")

		// make sure its {id}, not {id1}/{id2}/...
		if cloneID == "" || strings.Contains(cloneID, "/") {
			http.NotFound(w, r)
			return
		}

		cib, _, err := GetCIB()
		if err != nil {
			http.Error(w, "[CloneEditHandler] Failed to get clones in 'cibadmin -Ql': "+err.Error(),
				http.StatusInternalServerError)
			return
		}

		childResource := ""
		for _, clone := range cib.Configuration.Clones {
			if clone.ID == cloneID {
				if len(clone.Primitives) > 0 {
					childResource = clone.Primitives[0].ID
				}
				break
			}
		}
		if childResource == "" {
			http.Error(w, "Child resource not found", http.StatusNotFound)
			return
		}

		// If we do Configuration -> Add Resource -> Primitive -> Create
		// It would redirect to the cib/live/primitives/{primitive-id}/edit?flash={created|updated}
		flash := r.URL.Query().Get("flash")
		var alertType, alertMsg string

		switch flash {
		case "created":
			alertType = "success"
			alertMsg = "Clone created successfully"
		case "updated":
			alertType = "success"
			alertMsg = "Clone updated successfully"
		case "renamed":
			alertType = "success"
			alertMsg = "Clone renamed successfully"
		case "error":
			alertType = "danger"
			alertMsg = r.URL.Query().Get("msg")
			if alertMsg == "" {
				alertMsg = "There was an error processing the primitive."
			}
		}

		renderTemplate(w, "clone_edit", map[string]any{
			"Title":         "Edit Clone",
			"CloneID":       cloneID,
			"ChildResource": childResource,
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

func CloneNewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/clones/new" && r.URL.Path != "/cib/live/clones/new/" {
		http.NotFound(w, r)
		return
	}

	renderTemplate(w, "clone_new", map[string]any{
		"Title": "Create Clone",
	})
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
				alertMsg = "There was an error processing the node."
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

func updateNvpair(nvpair Nvpair, section string, resourceID string, resourceElement string) ([]byte, error) {
	if resourceElement != "primitive" && resourceElement != "clone" {
		return nil, fmt.Errorf("unsupported resource element %q", resourceElement)
	}

	xmlBytes, err := xml.Marshal(nvpair)
	if err != nil {
		log.Printf("[updateCibNvpair] XML marshal error: %v", err)
		return xmlBytes, err
	}
	xmlStr := string(xmlBytes)
	xmlStr = fmt.Sprintf("<%s id=\"%s\"><%s id=\"%s-%s\">%s</%s></%s>", resourceElement, resourceID, section, resourceID, section, xmlStr, section, resourceElement)

	queryXPath := fmt.Sprintf("//%s[@id='%s']", resourceElement, resourceID) // what about //*[@id='some-id']
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
		queryXPath = fmt.Sprintf("//%s[@id='%s-%s']", section, resourceID, section)
	} else {
		queryXPath = fmt.Sprintf("//nvpair[@id='%s']", cibAttributeID)
	}
	cmd := exec.Command("cibadmin", "--delete", "--xpath", queryXPath)
	return cmd.CombinedOutput()
}

func applyAttributes(cibAttributes []Nvpair, frontendAttributes []Nvpair, resourceID string, section string, resourceElement string, w http.ResponseWriter) {
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
			_, err := deleteNvpair(cibAttributes[i].ID, section, resourceID, attributesExist <= 1)
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
			newNvpair = Nvpair{ID: resourceID + "-" + section + "-" + frontendNvpair.Name, Name: frontendNvpair.Name, Value: frontendNvpair.Value}
		}
		_, err := updateNvpair(newNvpair, section, resourceID, resourceElement)
		if err != nil {
			http.Error(w, "Failed to execute cibadmin --update", http.StatusInternalServerError)
			log.Printf("[setPrimitive] cibadmin --update error: %v", err)
			return
		}
	}
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
		frontendPrimitive.ID, "instance_attributes", "primitive", w)

	// 4. Apply meta_attributes
	applyAttributes(cibPrimitive.MetaAttributes.NVPairs, frontendPrimitive.MetaAttributes.NVPairs,
		frontendPrimitive.ID, "meta_attributes", "primitive", w)

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

func ResourceRenameHandler(w http.ResponseWriter, r *http.Request) {
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

// This function does the magic routing between Go and Ruby
func LiveStatusHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "live_status", map[string]any{
		"Title": "Cluster Live Status",
	})
}
