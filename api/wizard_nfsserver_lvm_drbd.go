package api

import (
	"context"
	"encoding/json"
	"net/http"
)

var nfsServerLVMDRBDConfig = resourceWizardConfig{Name: "nfsserver-lvm-drbd", Title: "NFS Server on LVM and DRBD"}

func runNFSServerLVMDRBDCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	data, err := runNestedWizardCommand(ctx, "nfsserver-lvm-drbd", operation, params)
	if err != nil || operation != "show" {
		return data, err
	}
	var definition resourceWizardDefinition
	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, err
	}
	addExportFSExamples(&definition)
	addLVMDRBDExamples(&definition)
	return json.Marshal(definition)
}

func NFSServerLVMDRBDWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runNFSServerLVMDRBDCommand, nfsServerLVMDRBDConfig)
}
