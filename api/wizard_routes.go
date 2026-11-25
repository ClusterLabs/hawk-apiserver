package api

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type wizardBatchKey struct{}

func wizardIsBatch(ctx context.Context) bool {
	value, _ := ctx.Value(wizardBatchKey{}).(bool)
	return value
}
func wizardCIBBase(r *http.Request) string {
	if strings.HasPrefix(r.URL.Path, batchBase+"/wizards") {
		return batchBase
	}
	return "/cib/live"
}
func wizardRequest(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), wizardBatchKey{}, wizardCIBBase(r) == batchBase))
}
func renderWizardTemplate(w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	base := wizardCIBBase(r)
	data["CIBBase"] = base
	if base == batchBase {
		data["BatchMode"] = true
		data["BatchConfig"] = map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}
	}
	renderTemplate(w, name, data)
}

var wizardHandlers = map[string]http.HandlerFunc{
	"cryptctl":              CryptctlWizardHandler,
	"fence-vmware-rest":     VMwareRESTWizardHandler,
	"fence-virsh":           FenceVirshWizardHandler,
	"libvirt":               LibvirtWizardHandler,
	"sbd":                   SBDWizardHandler,
	"nginx":                 NginxWizardHandler,
	"haproxy":               HAProxyWizardHandler,
	"apache":                ApacheWizardHandler,
	"sap-simple-stack-plus": SAPSimpleStackPlusWizardHandler,
	"sap-simple-stack":      SAPSimpleStackWizardHandler,
	"sap-db":                SAPDBWizardHandler,
	"sap-ci":                SAPCIWizardHandler,
	"sap-as":                SAPASWizardHandler,
	"nfsserver-lvm-drbd":    NFSServerLVMDRBDWizardHandler,
	"nfsserver":             NFSServerWizardHandler,
	"exportfs":              ExportFSWizardHandler,
	"raid-lvm":              RAIDLVMWizardHandler,
	"lvm-drbd":              LVMDRBDWizardHandler,
	"gfs2":                  GFS2WizardHandler,
	"filesystem":            FilesystemWizardHandler,
	"drbd":                  DRBDWizardHandler,
	"oracle":                OracleWizardHandler,
	"db2-hadr":              DB2HADRWizardHandler,
	"db2":                   DB2WizardHandler,
	"database":              DatabaseWizardHandler,
	"virtual-ip":            VirtualIPWizardHandler,
	"mailto":                MailtoWizardHandler,
	"health":                HealthWizardHandler,
}

func WizardRoutesHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, wizardCIBBase(r)+"/wizards")
	if path == "" || path == "/" {
		WizardsHandler(w, r)
		return
	}
	name := strings.Split(strings.TrimPrefix(path, "/"), "/")[0]
	handler, ok := wizardHandlers[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	// Disabled legacy agents remain available only through their existing live routes.
	if wizardCIBBase(r) == batchBase {
		if name == "libvirt" || name == "vmware" {
			http.NotFound(w, r)
			return
		}
		suffix := strings.TrimSuffix(path, "/")
		if suffix != "/"+name && suffix != "/"+name+"/submit" {
			http.NotFound(w, r)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "POST" && r.Method != "PATCH" {
			w.Header().Set("Allow", "GET, HEAD, POST, PATCH")
			http.Error(w, "Method not allowed", 405)
			return
		}
		if strings.HasSuffix(suffix, "/submit") && r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		batch.mu.Lock()
		defer batch.mu.Unlock()
		exists, err := batch.exists()
		if err == nil && !exists {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			_, err = batch.run(ctx, true, "crm_shadow", []string{"-b", "-f", "-c", batchID}, "")
			cancel()
			if err == nil {
				batch.artifacts = batchArtifacts{}
			}
		}
		if err != nil {
			http.Error(w, "Unable to create shadow CIB: "+err.Error(), 500)
			return
		}
	}
	handler(w, wizardRequest(r))
}
