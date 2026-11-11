package api

import (
	"context"
	"encoding/json"
	"net/http"
)

var sapCIConfig = resourceWizardConfig{Name: "sap-ci", Title: "SAP Central Instance"}

func runSAPCICommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	data, err := runNestedWizardCommand(ctx, "sap-ci", operation, params)
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

func SAPCIWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runSAPCICommand, sapCIConfig)
}
