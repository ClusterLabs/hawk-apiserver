package api

import (
	"context"
	"encoding/json"
	"net/http"
)

var lvmDRBDConfig = resourceWizardConfig{Name: "lvm-drbd", Title: "LVM Group on DRBD"}

func runLVMDRBDCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	data, err := runNestedWizardCommand(ctx, "lvm-drbd", operation, params)
	if err != nil || operation != "show" {
		return data, err
	}
	var definition resourceWizardDefinition
	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, err
	}
	addLVMDRBDExamples(&definition)
	return json.Marshal(definition)
}

// Newer LVM-activate metadata leaves these required fields without examples.
// Use placeholders so the wizard does not choose an activation mode for the user.
func addLVMDRBDExamples(definition *resourceWizardDefinition) {
	var visit func(*wizardStep)
	visit = func(step *wizardStep) {
		for i := range step.Parameters {
			p := &step.Parameters[i]
			if p.Example != "" || p.Value != "" {
				continue
			}
			switch p.Name {
			case "vgname":
				p.Example = "vg1"
			case "vg_access_mode":
				p.Example = "system_id"
			}
		}
		for i := range step.Steps {
			visit(&step.Steps[i])
		}
	}
	for i := range definition.Steps {
		if definition.Steps[i].Name == "lvm" {
			visit(&definition.Steps[i])
		}
	}
}

func LVMDRBDWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runLVMDRBDCommand, lvmDRBDConfig)
}
