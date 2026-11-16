package api

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Scope custom crmsh definitions to one invocation, without changing installed
// scripts or global configuration. The directory is private and always removed.
func runBundledWizard(name string, script []byte, command func(string) (*exec.Cmd, error)) ([]byte, error) {
	directory, err := os.MkdirTemp("", "hawk-"+name+"-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	if err := os.WriteFile(filepath.Join(directory, name+".yml"), script, 0600); err != nil {
		return nil, err
	}
	cmd, err := command(directory)
	if err != nil {
		return nil, err
	}
	return runWizardProcess(cmd)
}
