package api

import (
	"context"
	"encoding/json"
	"net/http"
)

var cryptctlConfig = resourceWizardConfig{Name: "cryptctl", Title: "A utility for setting up LUKS-based disk encryption"}

func runCryptctlCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	data, err := runNestedWizardCommand(ctx, "cryptctl", operation, params)
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

func CryptctlWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runCryptctlCommand, cryptctlConfig)
}
