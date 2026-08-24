package api

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

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
	Name             string `xml:"name,attr"`
	ID               string `xml:"id,attr"`
	Online           bool   `xml:"online,attr"`
	Standby          bool   `xml:"standby,attr"`
	StandbyOnFail    bool   `xml:"standby_onfail,attr"`
	Maintenance      bool   `xml:"maintenance,attr"`
	Pending          bool   `xml:"pending,attr"`
	Unclean          bool   `xml:"unclean,attr"`
	Health           string `xml:"health,attr"`
	FeatureSet       string `xml:"feature_set,attr"`
	Shutdown         bool   `xml:"shutdown,attr"`
	ExpectedUp       bool   `xml:"expected_up,attr"`
	IsDC             bool   `xml:"is_dc,attr"`
	ResourcesRunning int    `xml:"resources_running,attr"`
	Type             string `xml:"type,attr"`
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
	NodesRunningOn int          `xml:"nodes_running_on,attr"`
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

func GetCrmStatus() (CrmStatus, int, error) {
	cmd := exec.Command("crm", "status", "--as-xml")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		stderrText := stderr.String()
		pacemakerRC := 0

		re := regexp.MustCompile(`\brc=(\d+)\b`)
		if match := re.FindStringSubmatch(stderrText); match != nil {
			rc, conversionErr := strconv.Atoi(match[1])
			if conversionErr == nil {
				pacemakerRC = rc
			}
		}

		// Fallback
		if pacemakerRC == 0 && strings.Contains(strings.ToLower(stderrText), "not connected") {
			pacemakerRC = 102
		}

		return CrmStatus{}, pacemakerRC, fmt.Errorf(
			"crm status failed: %w: %s",
			err,
			strings.TrimSpace(stderrText),
		)
	}

	var crm CrmStatus
	if err := xml.Unmarshal(output, &crm); err != nil {
		return CrmStatus{}, 0, err
	}

	return crm, 0, nil
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

func NodeMaintenanceOnHandler(w http.ResponseWriter, r *http.Request) {
	var nodeName string

	if err := json.NewDecoder(r.Body).Decode(&nodeName); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[NodeMaintenanceOnHandler] JSON decode error: %v", err)
		return
	}

	crmExecute(w, []string{"--force", "node", "maintenance", nodeName},
		fmt.Sprintf("Node %s is in maintenance mode", nodeName),
	)
}

func NodeMaintenanceOffHandler(w http.ResponseWriter, r *http.Request) {
	var nodeName string

	if err := json.NewDecoder(r.Body).Decode(&nodeName); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[NodeMaintenanceOffHandler] JSON decode error: %v", err)
		return
	}

	crmExecute(w, []string{"--force", "node", "ready", nodeName},
		fmt.Sprintf("Node %s is no more in maintenance mode", nodeName),
	)
}

func NodeStandbyOnHandler(w http.ResponseWriter, r *http.Request) {
	var nodeName string

	if err := json.NewDecoder(r.Body).Decode(&nodeName); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[NodeStandbyOnHandler] JSON decode error: %v", err)
		return
	}

	crmExecute(w, []string{"--force", "node", "standby", nodeName},
		fmt.Sprintf("Node %s is in standby mode", nodeName),
	)
}

func NodeStandbyOffHandler(w http.ResponseWriter, r *http.Request) {
	var nodeName string

	if err := json.NewDecoder(r.Body).Decode(&nodeName); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[NodeStandbyOffHandler] JSON decode error: %v", err)
		return
	}

	crmArgs := []string{"--force", "node", "online", nodeName}
	crmExecute(w, crmArgs, "Node is online")
}

func NodeFenceHandler(w http.ResponseWriter, r *http.Request) {
	var nodeName string

	if err := json.NewDecoder(r.Body).Decode(&nodeName); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[NodeStandbyOffHandler] JSON decode error: %v", err)
		return
	}

	crmArgs := []string{"--force", "node", "fence", nodeName}
	crmExecute(w, crmArgs, "Node is fenced")
}

func NodeClearstateHandler(w http.ResponseWriter, r *http.Request) {
	var nodeName string

	if err := json.NewDecoder(r.Body).Decode(&nodeName); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[NodeStandbyOffHandler] JSON decode error: %v", err)
		return
	}

	crmArgs := []string{"--force", "node", "clearstate", nodeName}
	crmExecute(w, crmArgs, "Node is fenced")
}

func crmExecute(w http.ResponseWriter, crmArgs []string, successMessage string) {
	cmd := exec.Command("/usr/sbin/crm", crmArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_, err := cmd.Output()
	if err != nil {
		http.Error(w, stderr.String(), http.StatusInternalServerError)
		log.Printf("[crmExecute] crm %s error: %v", strings.Join(crmArgs, " "), err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": successMessage,
	})
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

	crmExecute(
		w,
		args,
		fmt.Sprintf("Created %s", frontendPrimitive.ID),
	)
}

func CloneCreateHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: before creating the clone try creating it in the shadow-cib
	var frontendClone Clone

	if err := json.NewDecoder(r.Body).Decode(&frontendClone); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[CloneCreateHandler] JSON decode error: %v", err)
		return
	}
	if frontendClone.ID == "" || len(frontendClone.Primitives) == 0 || frontendClone.Primitives[0].ID == "" {
		http.Error(w, "Clone ID and child resource are required", http.StatusBadRequest)
		return
	}

	log.Printf("Creating resource %s with fields: %+v\n", frontendClone.ID, frontendClone)

	args := []string{"configure", "clone", frontendClone.ID, frontendClone.Primitives[0].ID}

	// Meta Attributes
	metaStarted := false
	for _, nvpair := range frontendClone.MetaAttributes.NVPairs {
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

	crmExecute(w, args, fmt.Sprintf("Created %s", frontendClone.ID))
}

func ResourceDeleteHandler(w http.ResponseWriter, r *http.Request) {
	var ResourceID string

	if err := json.NewDecoder(r.Body).Decode(&ResourceID); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveDeleteHandler] JSON decode error: %v", err)
		return
	}

	crmExecute(w, []string{"--force", "configure", "delete", ResourceID},
		fmt.Sprintf("Primitive %s deleted", ResourceID),
	)
}

func ResourceStartHandler(w http.ResponseWriter, r *http.Request) {
	var ResourceID string

	if err := json.NewDecoder(r.Body).Decode(&ResourceID); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveStartHandler] JSON decode error: %v", err)
		return
	}

	crmExecute(w, []string{"--force", "resource", "start", ResourceID},
		fmt.Sprintf("Primitive %s started", ResourceID),
	)
}

func ResourceStopHandler(w http.ResponseWriter, r *http.Request) {
	var ResourceID string

	if err := json.NewDecoder(r.Body).Decode(&ResourceID); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveStopHandler] JSON decode error: %v", err)
		return
	}

	crmExecute(w, []string{"--force", "resource", "stop", ResourceID},
		fmt.Sprintf("Primitive %s stopped", ResourceID),
	)
}

func ClonePromoteHandler(w http.ResponseWriter, r *http.Request) {
	var ResourceID string

	if err := json.NewDecoder(r.Body).Decode(&ResourceID); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveStartHandler] JSON decode error: %v", err)
		return
	}

	crmExecute(w, []string{"--force", "resource", "promote", ResourceID},
		fmt.Sprintf("Primitive %s promoted", ResourceID),
	)
}

func CloneDemoteHandler(w http.ResponseWriter, r *http.Request) {
	var ResourceID string

	if err := json.NewDecoder(r.Body).Decode(&ResourceID); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveStartHandler] JSON decode error: %v", err)
		return
	}

	crmExecute(w, []string{"--force", "resource", "demote", ResourceID},
		fmt.Sprintf("Primitive %s demoted", ResourceID),
	)
}

func ResourceMaintenanceOnHandler(w http.ResponseWriter, r *http.Request) {
	var ResourceID string

	if err := json.NewDecoder(r.Body).Decode(&ResourceID); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveMaintenanceOnHandler] JSON decode error: %v", err)
		return
	}

	crmExecute(w, []string{"--force", "resource", "maintenance", ResourceID},
		fmt.Sprintf("Primitive %s is in maintenance mode", ResourceID),
	)
}

func ResourceMaintenanceOffHandler(w http.ResponseWriter, r *http.Request) {
	var ResourceID string

	if err := json.NewDecoder(r.Body).Decode(&ResourceID); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveStartHandler] JSON decode error: %v", err)
		return
	}

	crmExecute(w, []string{"--force", "resource", "maintenance", ResourceID, "off"},
		fmt.Sprintf("Primitive %s is no more in maintenance mode", ResourceID),
	)
}

func ResourceMigrateHandler(w http.ResponseWriter, r *http.Request) {
	var pair struct {
		ResourceID  string `json:"ResourceID"`
		Destination string `json:"Destination"`
	}

	if err := json.NewDecoder(r.Body).Decode(&pair); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveMigrateHandler] JSON decode error: %v", err)
		return
	}

	// Destination can be emtpy (Migrate resource --> "Away from current node")
	crmExecute(w, []string{"--force", "resource", "migrate", pair.ResourceID, pair.Destination},
		fmt.Sprintf("Primitive %s is migrated", pair.ResourceID),
	)
}

func ResourceCleanupHandler(w http.ResponseWriter, r *http.Request) {
	var pair struct {
		ResourceID  string `json:"ResourceID"`
		Destination string `json:"Destination"`
	}

	if err := json.NewDecoder(r.Body).Decode(&pair); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveCleanupHandler] JSON decode error: %v", err)
		return
	}

	// Destination can be emtpy
	crmExecute(w, []string{"--force", "resource", "cleanup", pair.ResourceID, pair.Destination},
		fmt.Sprintf("Primitive %s is cleaned up", pair.ResourceID),
	)
}

func ResourceClearHandler(w http.ResponseWriter, r *http.Request) {
	var ResourceID string

	if err := json.NewDecoder(r.Body).Decode(&ResourceID); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[PrimitiveClearHandler] JSON decode error: %v", err)
		return
	}

	crmExecute(w, []string{"--force", "resource", "clear", ResourceID},
		fmt.Sprintf("Primitive %s is cleared", ResourceID),
	)
}

func FetchCrmStatus(w http.ResponseWriter, r *http.Request) {
	CrmStatus, pacemakerRC, err := GetCrmStatus()
	if err != nil {
		if pacemakerRC == 102 {
			http.Error(w, "Pacemaker cluster is offline: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "Failed to get crm status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(CrmStatus); err != nil {
		log.Printf("[FetchCrmStatus] JSON encode error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func FetchPrimitiveClasses(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("crm", "ra", "classes")
	out, err := cmd.Output()
	if err != nil {
		http.Error(w, "Failed to run 'crm ra classes'", http.StatusInternalServerError)
		log.Printf("[FetchResourceClasses] Command error: %v", err)
		return
	}

	lines := strings.Split(string(out), "\n")
	var classes []string

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		class := fields[0]
		// If second field is "/", it's the ocf line: `ocf / heartbeat ...`
		if len(fields) >= 2 && fields[1] == "/" {
			classes = append(classes, class)
		} else if len(fields) == 1 {
			// E.g., lines like "stonith" or "systemd"
			classes = append(classes, class)
		}
	}

	var content SelectContent
	for _, class := range classes {
		content.Options = append(content.Options, SelectOption{Name: class})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.Printf("[FetchResourceClasses] JSON encode error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func FetchPrimitiveProviders(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Class string `json:"Class"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request: missing class", http.StatusBadRequest)
		log.Printf("[FetchResourceProviders] JSON decode error: %v", err)
		return
	}

	if request.Class == "" {
		http.Error(w, "Missing required 'Class' field when quering provider", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("crm", "ra", "classes")
	out, err := cmd.Output()
	if err != nil {
		http.Error(w, "Failed to run 'crm ra classes'", http.StatusInternalServerError)
		log.Printf("[FetchResourceProviders] Command error: %v", err)
		return
	}

	lines := strings.Split(string(out), "\n")
	var providers []string

	for _, line := range lines {
		tokens := strings.Fields(line)
		if len(tokens) >= 3 && tokens[1] == "/" && tokens[0] == request.Class {
			providers = tokens[2:]
			break
		}
	}

	var content SelectContent
	for _, p := range providers {
		content.Options = append(content.Options, SelectOption{Name: p})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.Printf("[FetchResourceProviders] JSON encode error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func FetchPrimitiveTypes(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Class    string `json:"Class"`
		Provider string `json:"Provider"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[FetchResourceTypes] JSON decode error: %v", err)
		return
	}

	if input.Class == "" {
		http.Error(w, "Missing required 'Class' field when quering types", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("crm", "ra", "list", input.Class, input.Provider)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[FetchResourceTypes] crm ra list error: %v", err)
		http.Error(w, "Failed to list resource types: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Split by any whitespace and filter out empty entries
	lines := strings.Fields(string(out))

	var content SelectContent
	for _, t := range lines {
		content.Options = append(content.Options, SelectOption{Name: t})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.Printf("Failed to encode resource types: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func FetchChildResources(w http.ResponseWriter, r *http.Request) {
	cib, _, err := GetCIB()
	if err != nil {
		log.Printf("[FetchChildResources] Failed to get CIB: %v", err)
		http.Error(w, "Failed to get CIB resources: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var content SelectContent
	for _, primitive := range cib.Configuration.Primitives {
		content.Options = append(content.Options, SelectOption{Name: primitive.ID})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.Printf("[FetchChildResources] Failed to encode child resources: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
