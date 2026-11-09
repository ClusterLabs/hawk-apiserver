package api

import (
	"context"
	"net/http"
)

var db2HADRConfig = resourceWizardConfig{Name: "db2-hadr", Title: "DB2 HADR"}

func runDB2HADRCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	return runNestedWizardCommand(ctx, "db2-hadr", operation, params)
}
func DB2HADRWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runDB2HADRCommand, db2HADRConfig)
}
