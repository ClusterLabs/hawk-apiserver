package api

import (
	"context"
	"net/http"
)

var virtualIPConfig = resourceWizardConfig{Name: "virtual-ip", Title: "Virtual IP"}

func runVirtualIPCommand(ctx context.Context, operation string, params wizardParams) ([]byte, error) {
	return runResourceWizardCommand(ctx, "virtual-ip", operation, params)
}
func loadVirtualIPDefinition(ctx context.Context, run resourceWizardCommand) (resourceWizardDefinition, error) {
	return loadResourceWizardDefinition(ctx, run, virtualIPConfig.Title)
}
func validateVirtualIPParams(params wizardParams, definition resourceWizardDefinition) error {
	return validateResourceWizardParams(params, definition, virtualIPConfig.Title)
}
func VirtualIPWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveVirtualIPWizard(w, r, runVirtualIPCommand)
}
func serveVirtualIPWizard(w http.ResponseWriter, r *http.Request, run resourceWizardCommand) {
	serveResourceWizard(w, r, run, virtualIPConfig)
}
func submitVirtualIPWizard(w http.ResponseWriter, r *http.Request, run resourceWizardCommand) {
	submitResourceWizard(w, r, run, virtualIPConfig)
}
