package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strings"
)

type NameValue struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

// it's like MetaParameter, but ContentAttr is flattened
// it has nothing to do with parsing xml,
// it's in the structure conveniet to work with in the js
type SelectOption struct {
	Name           string   `json:"Name"`
	DefaultValue   string   `json:"DefaultValue"`
	Shortdesc      string   `json:"Shortdesc"`
	Longdesc       string   `json:"Longdesc"`
	Type           string   `json:"Type"`
	PossibleValues []string `json:"PossibleValues"`
	Required       string   `json:"Required"` // string, so that ["true", "false", "" for undefined]
	CibID          string   `json:"CibID"`
	CibValue       string   `json:"CibValue"`
}

type OperationOption struct {
	Name           string      `json:"Name"`
	DefaultValues  []NameValue `json:"DefaultValues"`
	Shortdesc      string      `json:"Shortdesc"`
	Longdesc       string      `json:"Longdesc"`
	Type           string      `json:"Type"`
	PossibleValues []string    `json:"PossibleValues"`
	Required       string      `json:"Required"` // string, so that ["true", "false", "" for undefined]
	// FIXME: in case of operations, there might be many CibIDs and each id has several values [interval, timeout,...]
	CibID string `json:"CibID"`
	/* CibNameValues is kinda hacky thing.
	 * If it's instance or meta attribute there should be
	 *    `CibValue string` instead, not an array `[]NameValue`
	 * For example
	 *    `<nvpair id="dummy1-instance_attributes-envfile" name="envfile" value="/etc/sysconfyg/hawk"/>`
	 * Hoever an operation may contain many key-values
	 *     `<op id="dummy1-monitor-5" interval="5" name="monitor" timeout="22"/>`
	 * e.i. interval=5, timeout=22, so we use []NameValue for both
	 * The convention is that the name is empty for instance and meta attributes NameValue{"", CibValue} */
	CibNameValues []NameValue `json:"CibNameValues"`
}

// Response data.
type SelectContent struct {
	Longdesc  string         `json:"Longdesc"`
	Shortdesc string         `json:"Shortdesc"`
	Options   []SelectOption `json:"Options"`
}

type OperationContent struct {
	Options []OperationOption `json:"Options"`
}

func parseIDandAgent(w http.ResponseWriter, r *http.Request) (string, string) {
	var pair struct {
		ResourceID    string `json:"ResourceID"`
		ResourceAgent string `json:"ResourceAgent"`
	}

	if err := json.NewDecoder(r.Body).Decode(&pair); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[fetchPrimitiveFromCib] JSON decode error: %v", err)
		return "", ""
	}
	return pair.ResourceID, pair.ResourceAgent
}

func fetchFullPrimitiveFromCib(ResourceID string, ResourceAgent string) (CrmResourceMetadata, error) {
	// 1. Get the main content 'crm_resource --show-metadata'
	metadata, err := getResourceMetadata(ResourceAgent)
	if err != nil {
		return CrmResourceMetadata{}, err
	}

	// 2. Copy the default meta_attributes, default operations and help info
	metadata.RscDefaults = GetRscDefaults()
	descriptions := GetOpDescriptions()
	for i := range metadata.Actions {
		metadata.Actions[i].OpDefaults = GetOpDefaults()
		// It's a special case. In hawk we also handle this case in the code in oplist.js
		if metadata.Actions[i].Name == "monitor" {
			// T.B.A. (#TODO)
		}
		for _, desc := range descriptions {
			// no idea why we need those 'op-' prefixes, but they exist in hawk
			if metadata.Actions[i].Name == desc.Name || "op-"+metadata.Actions[i].Name == desc.Name {
				metadata.Actions[i].Shortdesc = desc.Shortdesc
				metadata.Actions[i].Longdesc = desc.Longdesc
			}
		}
	}

	// 4. Get current values of the attributes from cib.xml
	err = enrichMetadataWithCibValues(&metadata, ResourceID)
	if err != nil {
		return CrmResourceMetadata{}, err
	}

	return metadata, nil
}

func fetchShortPrimitiveFromCib(ResourceID string) (Primitive, error) {
	// 1. Query current XML
	queryXPath := fmt.Sprintf("//primitive[@id='%s']", ResourceID)
	cmd := exec.Command("cibadmin", "-Q", "--xpath", queryXPath)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[setPrimitive] cibadmin -Q error: %v", err)
		return Primitive{}, err
	}

	// 2. Unmarshal to struct
	var cibPrimitive Primitive
	if err := xml.Unmarshal(out, &cibPrimitive); err != nil {
		log.Printf("[setPrimitive] XML unmarshal error: %v", err)
		return Primitive{}, err
	}

	return cibPrimitive, nil
}

func fetchPrimitiveFromFrontend(w http.ResponseWriter, r *http.Request) (Primitive, error) {
	var frontendPrimitive Primitive

	if err := json.NewDecoder(r.Body).Decode(&frontendPrimitive); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[UpdatePrimitiveHandler] JSON decode error: %v", err)
		return Primitive{}, err
	}

	log.Printf("Updating resource %s with fields: %+v\n", frontendPrimitive.ID, frontendPrimitive)

	return frontendPrimitive, nil
}

func FetchClusterDetails(w http.ResponseWriter, r *http.Request) {
	var frontendAgruments struct {
		Host string `json:"host"`
	}
	type ClusterDetails struct {
		Summary    string      `json:"Summary"`
		NameValues []NameValue `json:"NameValues"`
	}

	if err := json.NewDecoder(r.Body).Decode(&frontendAgruments); err != nil {
		log.Printf("[FetchClusterDetails] decode error: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("cibadmin", "-Ql")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[FetchClusterDetails] cibadmin error, pacemaker is offline: %v", err)

		result := ClusterDetails{"Error invoking /usr/sbin/cibadmin -Ql: " +
			"Could not connect to the CIB: Transport endpoint is not connected cibadmin: " +
			"Init failed, could not perform requested operations: " +
			"Transport endpoint is not connected", []NameValue{{"Status", "offline"}}}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("[FetchClusterDetails] JSON encode error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	var cib CIB
	if err := xml.Unmarshal(out, &cib); err != nil {
		log.Printf("[FetchClusterDetails] XML unmarshal error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	version := ""
	stack := ""
	for _, nvpair := range cib.Configuration.CrmConfig.ClusterPropertySet.NVPairs {
		if nvpair.Name == "dc-version" {
			version = nvpair.Value
		}
		if nvpair.Name == "cluster-infrastructure" {
			stack = nvpair.Value
		}
	}

	dc := ""
	for _, node := range cib.Configuration.Node {
		if node.ID == cib.DcUuid {
			dc = node.Uname
		}
	}

	status := "ok"
	summary := "OK"
	if cib.HaveQuorum == "0" {
		status = "errors"
		summary = "Partition without quorum! Fencing and resource management is disabled."
	}

	hostname := frontendAgruments.Host
	names, err := net.LookupAddr(frontendAgruments.Host)
	if err == nil && len(names) > 0 {
		hostname = names[0]
	}

	result := ClusterDetails{
		Summary: summary,
		NameValues: []NameValue{
			{"Status", status},
			{"Epoch", cib.AdminEpoch + ":" + cib.Epoch + ":" + cib.NumUpdates},
			{"Host", hostname},
			{"DC", dc},
			{"Schema", cib.ValidateWith},
			{"Last Written", cib.CibLastWritten},
			{"Update Origin", cib.UpdateOrigin},
			{"Update User", cib.UpdateUser},
			{"Have Quorum", cib.HaveQuorum},
			{"Version", version},
			{"Stack", stack},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("[FetchClusterDetails] JSON encode error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func FetchResourceClasses(w http.ResponseWriter, r *http.Request) {
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

func FetchResourceProviders(w http.ResponseWriter, r *http.Request) {
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

func FetchResourceTypes(w http.ResponseWriter, r *http.Request) {
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

func FetchResourceParams(w http.ResponseWriter, r *http.Request) {
	id, agent := parseIDandAgent(w, r)
	metadata, err := fetchFullPrimitiveFromCib(id, agent)
	if err != nil {
		log.Printf("Failed to get cib values: %v", err)
		http.Error(w, "Failed to get cib values: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var content SelectContent
	content.Shortdesc = metadata.Shortdesc
	content.Longdesc = metadata.Longdesc
	for _, param := range metadata.Parameters {
		content.Options = append(content.Options,
			SelectOption{
				param.Name,
				param.Content.Default,
				param.Shortdesc,
				param.Longdesc,
				param.Content.Type,
				param.Content.PossibleValues,
				param.Content.Required,
				param.Content.CibID,
				param.Content.CibValue,
			})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.Printf("Failed to encode data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func SubmitResourceParams(w http.ResponseWriter, r *http.Request) {
	frontendPrimitive, err := fetchPrimitiveFromFrontend(w, r)
	if err != nil {
		return
	}

	cibPrimitive, err := fetchShortPrimitiveFromCib(frontendPrimitive.ID)
	if err != nil {
		return
	}

	// 2. Apply instance_attributes
	applyAttributes(cibPrimitive.InstanceAttributes.NVPairs, frontendPrimitive.InstanceAttributes.NVPairs,
		frontendPrimitive.ID, "instance_attributes", w)

	// 3. Success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("Updated %s", frontendPrimitive.ID),
	})
}

func FetchResourceMetaAttributes(w http.ResponseWriter, r *http.Request) {
	id, agent := parseIDandAgent(w, r)
	metadata, err := fetchFullPrimitiveFromCib(id, agent)
	if err != nil {
		log.Printf("Failed to get cib values: %v", err)
		http.Error(w, "Failed to get cib values: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var content SelectContent
	content.Shortdesc = metadata.Shortdesc
	content.Longdesc = metadata.Longdesc
	for _, param := range metadata.RscDefaults {
		content.Options = append(content.Options,
			SelectOption{
				param.Name,
				param.Content.Default,
				param.Shortdesc,
				param.Longdesc,
				param.Content.Type,
				param.Content.PossibleValues,
				param.Content.Required,
				param.Content.CibID,
				param.Content.CibValue,
			})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.Printf("Failed to encode data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func SubmitResourceMetaAttributes(w http.ResponseWriter, r *http.Request) {
	frontendPrimitive, err := fetchPrimitiveFromFrontend(w, r)
	if err != nil {
		return
	}

	cibPrimitive, err := fetchShortPrimitiveFromCib(frontendPrimitive.ID)
	if err != nil {
		return
	}

	// 2. Apply instance_attributes
	applyAttributes(cibPrimitive.MetaAttributes.NVPairs, frontendPrimitive.MetaAttributes.NVPairs,
		frontendPrimitive.ID, "meta_attributes", w)

	// 3. Success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("Updated %s", frontendPrimitive.ID),
	})
}

func FetchResourceOperations(w http.ResponseWriter, r *http.Request) {
	id, agent := parseIDandAgent(w, r)
	metadata, err := fetchFullPrimitiveFromCib(id, agent)
	if err != nil {
		log.Printf("Failed to get cib values: %v", err)
		http.Error(w, "Failed to get cib values: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var content OperationContent
	for _, action := range metadata.Actions {
		var nameValues []NameValue
		if action.CibID != "" {
			for _, opdef := range action.OpDefaults {
				if opdef.Content.CibValue != "" {
					nameValues = append(nameValues, NameValue{opdef.Name, opdef.Content.CibValue})
				}
			}
		}
		newOption := OperationOption{
			action.Name,
			[]NameValue{
				// action.Interval is what we parse
				// from crm_resource --show-metadata
				{"interval", action.Interval},
				{"timeout", action.Timeout},
				{"depth", action.Depth},
			},
			action.Shortdesc, //param.Shortdesc,
			action.Longdesc,  //param.Longdesc,
			"",               //param.Content.Type,
			[]string{""},     //param.Content.PossibleValues,
			"",               //param.Content.Required,
			action.CibID,     //param.Content.CibID,
			nameValues,
		}
		content.Options = append(content.Options, newOption)
	}

	// Convert to JSON.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.Printf("Failed to fetch select data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func SubmitResourceOperations(w http.ResponseWriter, r *http.Request) {
	frontendPrimitive, err := fetchPrimitiveFromFrontend(w, r)
	if err != nil {
		return
	}

	cibPrimitive, err := fetchShortPrimitiveFromCib(frontendPrimitive.ID)
	if err != nil {
		return
	}

	// 2. Apply operations
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
				if cibPrimitive.Operations[i].Timeout != frontendOp.Timeout {
					cibPrimitive.Operations[i].Timeout = frontendOp.Timeout
					opUpdated = true
				}
				if cibPrimitive.Operations[i].Interval != frontendOp.Interval {
					cibPrimitive.Operations[i].Interval = frontendOp.Interval
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
				Name:     frontendOp.Name,
				Interval: frontendOp.Interval,
				Timeout:  frontendOp.Timeout,
				Depth:    frontendOp.Depth,
			}
		}
		_, err = updateOperation(newOp, frontendPrimitive.ID)
		if err != nil {
			http.Error(w, "Failed to encode updated XML", http.StatusInternalServerError)
			log.Printf("[setPrimitive] XML marshal error: %v", err)
			return
		}
	}

	// 3. Remove operations that exist in CIB but not in frontend (by op ID)
	frontendIDs := make(map[string]struct{}, len(frontendPrimitive.Operations))
	for _, op := range frontendPrimitive.Operations {
		if op.ID == "" {
			continue
		}
		frontendIDs[op.ID] = struct{}{}
	}

	operationsExist := len(cibPrimitive.Operations)
	for _, cibOp := range cibPrimitive.Operations {
		_, operationExistsInFrontend := frontendIDs[cibOp.ID]
		if operationExistsInFrontend {
			continue
		}

		_, err := deleteOperation(cibOp.ID, frontendPrimitive.ID, operationsExist <= 1)
		operationsExist--
		if err != nil {
			http.Error(w, "Failed to delete operation: "+err.Error(), http.StatusInternalServerError)
			log.Printf("[SubmitResourceOperations] deleteOperation error: %v", err)
			return
		}
	}

	// 4. Success
	// FIXME! if there were 0 updates --> is't not a successful update, it's neutral OK.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("Updated %s", frontendPrimitive.ID),
	})
}

func enrichOpDefaultsWithCibValues(opDefaults []MetaParameter, resourceID string,
	operation string, operationID string) error {

	// 1. Query current XML
	queryXPath := fmt.Sprintf("/cib/configuration/resources/primitive[@id='%s']", resourceID)
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

	for _, op := range primitive.Operations {
		if op.ID != operationID {
			continue
		}
		for _, opDef := range opDefaults {
			/* It looks ugly that we compare the strings this way.
			 * One would require a more generic approach.
			 * Thus we should change the Primitive structure,
			 * i.e. that Interval, Timeout, etc. are an array of values,
			 * not fields of the strutcure.
			 * However we can't to this, because this structure is used
			 * for parsing the xml response. */
			if opDef.Name == "interval" {
				opDef.Content.CibValue = op.Interval
			}
			if opDef.Name == "timeout" {
				opDef.Content.CibValue = op.Timeout
			}
		}
	}

	return nil
}

func FetchResourceOperationAttributes(w http.ResponseWriter, r *http.Request) {
	var frontendPrimitive struct {
		ID            string `json:"ResourceID"`
		ResourceAgent string `json:"ResourceAgent"`
		Operation     string `json:"Operation"`
		OperationID   string `json:"OperationID"` // "" if Create, like "dummy1-monitor-11s" if Update
	}

	if err := json.NewDecoder(r.Body).Decode(&frontendPrimitive); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[FetchResourceOperationAttributes] JSON decode error: %v", err)
	}

	opDefaults := GetOpDefaults()
	if frontendPrimitive.Operation == "monitor" {
		// TODO: append(opDefaults, OCF_CHECK_LEVEL)
	}

	// No need here. Do it in
	enrichOpDefaultsWithCibValues(opDefaults, frontendPrimitive.ID,
		frontendPrimitive.Operation, frontendPrimitive.OperationID)

	var content SelectContent
	for _, opAttr := range opDefaults {
		content.Options = append(content.Options,
			SelectOption{
				opAttr.Name,
				opAttr.Content.Default,
				opAttr.Shortdesc,
				opAttr.Longdesc,
				opAttr.Content.Type,
				opAttr.Content.PossibleValues,
				opAttr.Content.Required,
				opAttr.Content.CibID,
				opAttr.Content.CibValue,
			})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.Printf("Failed to encode data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func SubmitResourceOperationAttributes(w http.ResponseWriter, r *http.Request) {
	// OLD comment: TBA. It's just a stab
	// New comment (JAN 2026): I don't think it's needed.
	// if you still see this comment --> delete this function.

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(""); err != nil {
		log.Printf("Failed to encode data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
