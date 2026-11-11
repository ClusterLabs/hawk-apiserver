package api

import (
	"context"
	"encoding/json"
	"net/http"
)

var sbdConfig = resourceWizardConfig{Name: "sbd", Title: "SBD, Shared storage based fencing"}

func runSBDCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	data, err := runNestedWizardCommand(ctx, "sbd", operation, params)
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

func SBDWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runSBDCommand, sbdConfig)
}
