package api

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

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
	/* `MetaAttributes []MetaParameter`
	 * is not in 'crm_resource --show-metadata'
	 * but it's copied from rscDefaults
	 * and later enriched from `cibadmin` */
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

// Action is almost like Operation in cib.xml, not sure about merging them
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
}

func getResourceMetadata(resourceAgent string) (CrmResourceMetadata, error) {
	//var cmd *exec.Cmd
	cmd := exec.Command("crm_resource", "--show-metadata", resourceAgent)

	out, err := cmd.Output()
	if err != nil {
		return CrmResourceMetadata{}, err
	}

	var metadata CrmResourceMetadata
	if err := xml.Unmarshal(out, &metadata); err != nil {
		return CrmResourceMetadata{}, err
	}
	metadata.Actions = firstActionsByName(metadata.Actions)

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

	// if there is no information found, take the defaults from actionDefaults
	for i := range metadata.Actions {
		if metadata.Actions[i].Depth == "" {
			metadata.Actions[i].Depth = actionDefaults.Depth.Content.Default
		}
		if metadata.Actions[i].Description == "" {
			metadata.Actions[i].Description = actionDefaults.Description.Content.Default
		}
		if metadata.Actions[i].Enabled == "" {
			metadata.Actions[i].Enabled = actionDefaults.Enabled.Content.Default
		}
		if metadata.Actions[i].Interval == "" {
			metadata.Actions[i].Interval = actionDefaults.Interval.Content.Default
		}
		if metadata.Actions[i].IntervalOrigin == "" {
			metadata.Actions[i].IntervalOrigin = actionDefaults.IntervalOrigin.Content.Default
		}
		if metadata.Actions[i].OnFail == "" {
			metadata.Actions[i].OnFail = actionDefaults.OnFail.Content.Default
		}
		if metadata.Actions[i].RecordPending == "" {
			metadata.Actions[i].RecordPending = actionDefaults.RecordPending.Content.Default
		}
		if metadata.Actions[i].Requires == "" {
			metadata.Actions[i].Requires = actionDefaults.Requires.Content.Default
		}
		if metadata.Actions[i].Role == "" {
			metadata.Actions[i].Role = actionDefaults.Role.Content.Default
		}
		if metadata.Actions[i].StartDelay == "" {
			metadata.Actions[i].StartDelay = actionDefaults.StartDelay.Content.Default
		}
		if metadata.Actions[i].Timeout == "" {
			metadata.Actions[i].Timeout = actionDefaults.Timeout.Content.Default
		}
	}

	return metadata, nil
}

func fetchFullCloneFromCib(CloneID string) (FullPrimitive_CrmResourceMetadata, error) {
	metadata := FullPrimitive_CrmResourceMetadata{}

	// 1. Copy the default meta_attributes, default operations and help info
	metadata.MetaAttributes = GetCloneDefaults()

	if CloneID == "" {
		return metadata, nil
	}

	// 2. Get current values of the attributes from cib.xml
	err := enrichCloneMetaAttributesWithCibValues(&metadata, CloneID)
	if err != nil {
		return FullPrimitive_CrmResourceMetadata{}, err
	}

	return metadata, nil
}

func FetchCloneMetaAttributes(w http.ResponseWriter, r *http.Request) {
	cloneID, _ := parseIDandAgent(w, r)

	metadata, err := fetchFullCloneFromCib(cloneID)
	if err != nil {
		log.Printf("Failed to get cib values: %v", err)
		http.Error(w, "Failed to get cib values: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var content SelectContent
	content.Shortdesc = metadata.Shortdesc
	content.Longdesc = metadata.Longdesc
	for _, param := range metadata.MetaAttributes {
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
