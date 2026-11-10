package api

import (
	"context"
	"encoding/json"
	"net/http"
)

var exportFSConfig = resourceWizardConfig{Name: "exportfs", Title: "NFS Exported File System"}

func runExportFSCommand(ctx context.Context, operation string, params wizardParams) ([]byte, error) {
	data, err := runResourceWizardCommand(ctx, "exportfs", operation, params)
	if err != nil || operation != "show" {
		return data, err
	}
	var definition resourceWizardDefinition
	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, err
	}
	addExportFSExamples(&definition)
	return json.Marshal(definition)
}

func addExportFSExamples(definition *resourceWizardDefinition) {
	var visit func(*wizardStep)
	visit = func(step *wizardStep) {
		for j := range step.Parameters {
			p := &step.Parameters[j]
			if p.Example != "" || p.Value != "" {
				continue
			}
			switch p.Name {
			case "clientspec":
				p.Example = "192.0.2.0/24"
			case "options":
				p.Example = "rw,mountpoint"
			case "directory":
				p.Example = "/srv/nfs/home"
			}
		}
		for i := range step.Steps {
			visit(&step.Steps[i])
		}
	}
	for i := range definition.Steps {
		visit(&definition.Steps[i])
	}
}

func ExportFSWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveResourceWizard(w, r, runExportFSCommand, exportFSConfig)
}
