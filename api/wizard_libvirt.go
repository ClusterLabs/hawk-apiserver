package api

import (
	"context"
	"net/http"
)

var libvirtConfig = resourceWizardConfig{Name: "libvirt", Title: "STONITH for libvirt (kvm / Xen)"}

func runLibvirtCommand(ctx context.Context, operation string, params wizardParams) ([]byte, error) {
	return runResourceWizardCommand(ctx, "libvirt", operation, params)
}

func LibvirtWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveResourceWizard(w, r, runLibvirtCommand, libvirtConfig)
}
