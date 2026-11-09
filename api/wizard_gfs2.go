package api

import (
	"context"
	"net/http"
)

var gfs2Config = resourceWizardConfig{Name: "gfs2", Title: "GFS2 File System (Cloned)"}

func runGFS2Command(ctx context.Context, operation string, params wizardParams) ([]byte, error) {
	return runResourceWizardCommand(ctx, "gfs2", operation, params)
}

func GFS2WizardHandler(w http.ResponseWriter, r *http.Request) {
	serveResourceWizard(w, r, runGFS2Command, gfs2Config)
}
