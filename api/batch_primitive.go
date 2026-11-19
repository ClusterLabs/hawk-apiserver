package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

func BatchPrimitiveEditHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	batch.mu.Lock()
	defer batch.mu.Unlock()
	servePrimitiveEdit(w, r, func() ([]ResourceRow, error) {
		cib, _, err := batch.readCIB(r.Context(), false)
		return cibResourceRows(cib), err
	})
}

func (b *batchBackend) primitiveFields(w http.ResponseWriter, r *http.Request, ctx context.Context, operation string) {
	run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
		return b.run(ctx, false, name, args, input)
	}
	section := strings.Split(operation, "/")[1]
	if strings.HasSuffix(operation, "/fetch") && section != "utilizations" {
		read := func(id, agent string) (FullPrimitive_CrmResourceMetadata, error) {
			return fetchFullPrimitiveUsing(id, agent, func(metadata *FullPrimitive_CrmResourceMetadata, id string) error {
				if id != "" && !primitiveID.MatchString(id) {
					return fmt.Errorf("Invalid resource ID")
				}
				return enrichPrimitiveMetadataUsing(metadata, id, func(name string, args []string) ([]byte, error) { return run(ctx, name, args, "") })
			})
		}
		switch section {
		case "params":
			FetchResourceParamsUsing(w, r, read)
		case "operations":
			FetchResourceOperationsUsing(w, r, read)
		case "meta-attributes":
			FetchResourceMetaAttributesUsing(w, r, read)
		}
		return
	}
	var input struct {
		Primitive
		CibObject string   `json:"CibObject"`
		NVPairs   []Nvpair `json:"nvpair"`
	}
	if !readBatchJSON(w, r, &input) {
		return
	}
	id := input.ID
	if section == "utilizations" {
		id = input.CibObject
	}
	if !primitiveID.MatchString(id) {
		batchError(w, 400, fmt.Errorf("Invalid resource ID"))
		return
	}
	xpath := "/cib/configuration/resources//primitive[@id='" + id + "']"
	raw, err := run(ctx, "cibadmin", []string{"-Q", "--xpath", xpath}, "")
	if err != nil {
		batchError(w, 404, err)
		return
	}
	if strings.HasSuffix(operation, "/fetch") {
		var primitive Primitive
		if err = xml.Unmarshal(raw, &primitive); err != nil {
			batchError(w, 500, err)
			return
		}
		batchJSON(w, primitive.Utilizations)
		return
	}
	var primitive groupXMLPart
	if err = xml.Unmarshal(raw, &primitive); err != nil {
		batchError(w, 500, err)
		return
	}
	var children struct {
		Items []groupXMLPart `xml:",any"`
	}
	if err = xml.Unmarshal([]byte("<primitive>"+primitive.Inner+"</primitive>"), &children); err != nil {
		batchError(w, 500, err)
		return
	}
	var element string
	var value any
	switch section {
	case "params":
		element = "instance_attributes"
		value = input.InstanceAttributes
	case "meta-attributes":
		element = "meta_attributes"
		value = input.MetaAttributes
	case "operations":
		element = "operations"
		value = struct {
			XMLName xml.Name    `xml:"operations"`
			Ops     []Operation `xml:"op"`
		}{Ops: input.Operations}
	case "utilizations":
		element = "utilization"
		value = MetaAttribute{ID: id + "-utilization", NVPairs: input.NVPairs}
	}
	// Give new entries stable IDs while retaining IDs supplied by the shared controls.
	normalize := func(pairs []Nvpair) {
		for i := range pairs {
			if pairs[i].ID == "" {
				pairs[i].ID = id + "-" + element + "-" + pairs[i].Name
			}
		}
	}
	switch v := value.(type) {
	case InstanceAttribute:
		if v.ID == "" {
			v.ID = id + "-" + element
		}
		normalize(v.NVPairs)
		value = v
	case MetaAttribute:
		if v.ID == "" {
			v.ID = id + "-" + element
		}
		normalize(v.NVPairs)
		value = v
	}
	if section == "operations" {
		for i := range input.Operations {
			op := &input.Operations[i]
			if op.ID == "" {
				op.ID = fmt.Sprintf("%s-%s-%d", id, op.Name, i)
			}
		}
		value = struct {
			XMLName xml.Name    `xml:"operations"`
			Ops     []Operation `xml:"op"`
		}{Ops: input.Operations}
	}
	data, err := xml.Marshal(value)
	if err != nil {
		batchError(w, 400, err)
		return
	}
	// Marshal through an explicit element name, keeping unrelated XML intact.
	var wrapper groupXMLPart
	if err = xml.Unmarshal(data, &wrapper); err != nil {
		batchError(w, 400, err)
		return
	}
	wrapper.XMLName = xml.Name{Local: element}
	data, err = xml.Marshal(wrapper)
	if err != nil {
		batchError(w, 400, err)
		return
	}
	var inner strings.Builder
	for _, child := range children.Items {
		if child.XMLName.Local != element {
			encoded, e := xml.Marshal(child)
			if e != nil {
				batchError(w, 500, e)
				return
			}
			inner.Write(encoded)
		}
	}
	inner.Write(data)
	primitive.Inner = inner.String()
	data, err = xml.Marshal(primitive)
	if err != nil {
		batchError(w, 500, err)
		return
	}
	if _, err = run(ctx, "cibadmin", []string{"--replace", "--xpath", xpath, "--xml-text", string(data)}, ""); err != nil {
		batchError(w, 500, err)
		return
	}
	batchJSON(w, map[string]string{"status": "ok", "message": "Updated " + id})
}

func primitiveAction(ctx context.Context, id, newID string, run clusterConfigCommand) (int, error) {
	if !primitiveID.MatchString(id) || (newID != "" && !primitiveID.MatchString(newID)) {
		return 400, fmt.Errorf("Invalid resource ID")
	}
	if _, err := run(ctx, "cibadmin", []string{"-Q", "--xpath", "/cib/configuration/resources//primitive[@id='" + id + "']"}, ""); err != nil {
		return 404, err
	}
	args := []string{"--force", "configure", "delete", id}
	if newID != "" {
		args = []string{"-D", "plain", "configure", "rename", id, newID}
	}
	_, err := run(ctx, "crm", args, "")
	if err != nil {
		return 500, err
	}
	return 200, nil
}

var primitiveID = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.:-]*$`)
