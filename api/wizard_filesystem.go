package api

import (
	"context"
	"net/http"
)

var filesystemConfig = resourceWizardConfig{Name: "filesystem", Title: "File System (mount point)"}

func runFilesystemCommand(ctx context.Context, operation string, params wizardParams) ([]byte, error) {
	return runResourceWizardCommand(ctx, "filesystem", operation, params)
}

func FilesystemWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveResourceWizard(w, r, runFilesystemCommand, filesystemConfig)
}
