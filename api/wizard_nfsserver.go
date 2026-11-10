package api

import (
	"context"
	"encoding/json"
	"net/http"
)

var nfsServerConfig = resourceWizardConfig{Name: "nfsserver", Title: "NFS Server"}

func runNFSServerCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	data, err := runNestedWizardCommand(ctx, "nfsserver", operation, params)
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

func NFSServerWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runNFSServerCommand, nfsServerConfig)
}
