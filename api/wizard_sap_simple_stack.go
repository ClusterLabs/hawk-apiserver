package api

import (
	"context"
	"encoding/json"
	"net/http"
)

var sapSimpleStackConfig = resourceWizardConfig{Name: "sap-simple-stack", Title: "SAP SimpleStack Instance"}

func runSAPSimpleStackCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	data, err := runNestedWizardCommand(ctx, "sap-simple-stack", operation, params)
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

func SAPSimpleStackWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runSAPSimpleStackCommand, sapSimpleStackConfig)
}
