package api

import (
	"context"
	"net/http"
)

var drbdConfig = resourceWizardConfig{Name: "drbd", Title: "DRBD Block Device"}

func runDRBDCommand(ctx context.Context, operation string, params wizardParams) ([]byte, error) {
	return runResourceWizardCommand(ctx, "drbd", operation, params)
}

func DRBDWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveResourceWizard(w, r, runDRBDCommand, drbdConfig)
}
