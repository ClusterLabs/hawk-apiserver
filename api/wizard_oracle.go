package api

import (
	"context"
	"net/http"
)

var oracleConfig = resourceWizardConfig{Name: "oracle", Title: "Oracle Database"}

func runOracleCommand(ctx context.Context, operation string, params wizardParams) ([]byte, error) {
	return runResourceWizardCommand(ctx, "oracle", operation, params)
}

func OracleWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveResourceWizard(w, r, runOracleCommand, oracleConfig)
}
