package api

import (
	"context"
	"net/http"
)

var haProxyConfig = resourceWizardConfig{Name: "haproxy", Title: "HAProxy"}

func runHAProxyCommand(ctx context.Context, operation string, params wizardParams) ([]byte, error) {
	return runResourceWizardCommand(ctx, "haproxy", operation, params)
}

func HAProxyWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveResourceWizard(w, r, runHAProxyCommand, haProxyConfig)
}
