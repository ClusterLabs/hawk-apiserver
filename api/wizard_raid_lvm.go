package api

import (
	"context"
	"encoding/json"
	"net/http"
)

var raidLVMConfig = resourceWizardConfig{Name: "raid-lvm", Title: "RAID Hosting LVM"}

func runRAIDLVMCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	data, err := runNestedWizardCommand(ctx, "raid-lvm", operation, params)
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

func RAIDLVMWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runRAIDLVMCommand, raidLVMConfig)
}
