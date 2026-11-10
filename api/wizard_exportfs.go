package api

import (
	"context"
	"net/http"
)

var exportFSConfig = resourceWizardConfig{Name: "exportfs", Title: "NFS Exported File System"}

func runExportFSCommand(ctx context.Context, operation string, params wizardParams) ([]byte, error) {
	return runResourceWizardCommand(ctx, "exportfs", operation, params)
}

func ExportFSWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveResourceWizard(w, r, runExportFSCommand, exportFSConfig)
}
