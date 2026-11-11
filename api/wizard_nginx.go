package api

import (
	"context"
	"net/http"
)

var nginxConfig = resourceWizardConfig{Name: "nginx", Title: "Nginx Webserver"}

func runNginxCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	return runNestedWizardCommand(ctx, "nginx", operation, params)
}

func NginxWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runNginxCommand, nginxConfig)
}
