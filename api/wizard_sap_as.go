package api

import (
	"context"
	"encoding/json"
	"net/http"
)

var sapASConfig = resourceWizardConfig{Name: "sap-as", Title: "SAP ASCS Instance"}

func runSAPASCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	data, err := runNestedWizardCommand(ctx, "sap-as", operation, params)
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

func SAPASWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runSAPASCommand, sapASConfig)
}
