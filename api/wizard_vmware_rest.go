package api

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

//go:embed wizard_scripts/fence-vmware-rest.yml
var vmwareRESTScript []byte

var vmwareRESTConfig = resourceWizardConfig{Name: "fence-vmware-rest", Title: "Fencing using vCenter REST API"}

const deprecatedVMwareTitle = "Deprecated: Fencing using vCenter / ESX Server"

func vmwareRESTScriptCommand(ctx context.Context, operation string, params db2Params, directory string) (*exec.Cmd, error) {
	// The crmsh template interpolates quoted values. Escape them without changing
	// the browser's saved values, and reject newlines that could add statements.
	escaped := db2Params{}
	if operation != "show" {
		allowed := map[string]bool{"id": true, "node_name": true, "machine_name": true, "server": true, "username": true, "password": true, "ipport": true, "api_path": true}
		for name, value := range params {
			text, ok := value.(string)
			if !allowed[name] || !ok {
				return nil, fmt.Errorf("Invalid VMware REST parameter: %s", name)
			}
			if strings.ContainsAny(text, "\r\n\x00") {
				return nil, fmt.Errorf("Invalid value for %s", name)
			}
			escaped[name] = strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(text)
		}
		for _, name := range []string{"id", "node_name", "machine_name", "server", "username", "password"} {
			if strings.TrimSpace(db2Value(params, []string{name})) == "" {
				return nil, fmt.Errorf("%s is required", name)
			}
		}
		for _, name := range []string{"id", "node_name"} {
			if !libvirtIdentifier.MatchString(db2Value(params, []string{name})) {
				return nil, fmt.Errorf("Invalid %s", name)
			}
		}
		if !libvirtHost.MatchString(db2Value(params, []string{"server"})) {
			return nil, fmt.Errorf("Use a vCenter IP address or hostname without a URL scheme or path")
		}
		if strings.Contains(db2Value(params, []string{"machine_name"}), ";") {
			return nil, fmt.Errorf("VM name cannot contain the host-map separator ';'")
		}
		if port := db2Value(params, []string{"ipport"}); port != "" {
			n, err := strconv.Atoi(port)
			if err != nil || n < 1 || n > 65535 {
				return nil, fmt.Errorf("HTTPS port must be between 1 and 65535")
			}
		}
		if path := db2Value(params, []string{"api_path"}); path != "" && (!strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?# ")) {
			return nil, fmt.Errorf("Use an API path such as /rest")
		}
	}
	cmd, err := nestedWizardScriptCommand(ctx, "fence-vmware-rest", operation, escaped)
	if err != nil {
		return nil, err
	}
	cmd.Args = append([]string{cmd.Args[0], "--scriptdir", directory}, cmd.Args[1:]...)
	return cmd, nil
}

func runVMwareRESTCommand(ctx context.Context, operation string, params db2Params) ([]byte, error) {
	data, err := runBundledWizard("fence-vmware-rest", vmwareRESTScript, func(directory string) (*exec.Cmd, error) {
		return vmwareRESTScriptCommand(ctx, operation, params, directory)
	})
	if err != nil || operation != "show" {
		return data, err
	}
	var definition resourceWizardDefinition
	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, err
	}
	for i := range definition.Steps {
		for j := range definition.Steps[i].Parameters {
			if definition.Steps[i].Parameters[j].Name == "password" {
				definition.Steps[i].Parameters[j].Type = "password"
			}
		}
	}
	return json.Marshal(definition)
}

func VMwareRESTWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveNestedWizard(w, r, runVMwareRESTCommand, vmwareRESTConfig)
}
