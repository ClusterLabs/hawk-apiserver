package api

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

//go:embed wizard_scripts/libvirt.yml
var libvirtWizardScript []byte

var legacyLibvirtConfig = resourceWizardConfig{Name: "libvirt", Title: "Deprecated: STONITH for libvirt (kvm / Xen)"}

var fenceVirshConfig = resourceWizardConfig{Name: "fence-virsh", Title: "Fencing for libvirt (fence_virsh)"}

var libvirtIdentifier = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`)
var libvirtHost = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.:%-]*$`)

var libvirtDomain = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.:-]*$`)

// Values are interpolated into crmsh input, not shell commands. Domain names
// must also be safe for fence_virsh's remote virsh command.
func validateLibvirtParams(params db2Params) error {
	allowed := map[string]bool{"id": true, "target": true, "ip": true, "username": true, "plug": true, "identity_file": true, "ipport": true, "install": true}
	for name, value := range params {
		text, ok := value.(string)
		if !allowed[name] || !ok {
			return fmt.Errorf("Invalid libvirt wizard parameter: %s", name)
		}
		if strings.ContainsAny(text, "\"\\\r\n\x00") {
			return fmt.Errorf("Invalid value for %s", name)
		}
	}
	for _, name := range []string{"id", "target", "ip", "username", "plug"} {
		if strings.TrimSpace(db2Value(params, []string{name})) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	for _, name := range []string{"id", "target", "username"} {
		if !libvirtIdentifier.MatchString(db2Value(params, []string{name})) {
			return fmt.Errorf("Invalid %s", name)
		}
	}
	if !libvirtDomain.MatchString(db2Value(params, []string{"plug"})) {
		return fmt.Errorf("Use a libvirt domain name or UUID containing letters, digits, dots, colons, underscores or hyphens")
	}
	if !libvirtHost.MatchString(db2Value(params, []string{"ip"})) {
		return fmt.Errorf("Use a hypervisor IP address or hostname")
	}
	if install := db2Value(params, []string{"install"}); install != "" && install != "true" && install != "false" {
		return fmt.Errorf("Install must be true or false")
	}
	if port := db2Value(params, []string{"ipport"}); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return fmt.Errorf("SSH port must be between 1 and 65535")
		}
	}
	return nil
}

func libvirtScriptCommand(ctx context.Context, operation string, params db2Params, directory string) (*exec.Cmd, error) {
	if operation != "show" {
		if err := validateLibvirtParams(params); err != nil {
			return nil, err
		}
	}
	cmd, err := nestedWizardScriptCommand(ctx, "libvirt", operation, params)
	if err != nil {
		return nil, err
	}
	cmd.Args = append([]string{cmd.Args[0], "--scriptdir", directory}, cmd.Args[1:]...)
	return cmd, nil
}

func runLibvirtCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	return runBundledWizard("libvirt", libvirtWizardScript, func(directory string) (*exec.Cmd, error) {
		return libvirtScriptCommand(ctx, operation, params, directory)
	})
}

func FenceVirshWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runLibvirtCommand, fenceVirshConfig)
}

// Preserve the original cluster-glue wizard alongside the modern fence_virsh one.
func LibvirtWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, func(ctx context.Context, operation string, params db2Params) ([]byte, error) {
		data, err := runNestedWizardCommand(ctx, "libvirt", operation, params)
		if err != nil || operation != "show" {
			return data, err
		}
		var definition resourceWizardDefinition
		if err := json.Unmarshal(data, &definition); err != nil {
			return nil, err
		}
		definition.Shortdesc = legacyLibvirtConfig.Title
		return json.Marshal(definition)
	}, legacyLibvirtConfig)
}
