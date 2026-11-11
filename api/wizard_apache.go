package api

import (
	"context"
	"net/http"
)

var apacheConfig = resourceWizardConfig{Name: "apache", Title: "Apache Webserver"}

func runApacheCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	return runNestedWizardCommand(ctx, "apache", operation, params)
}

func ApacheWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runApacheCommand, apacheConfig)
}
