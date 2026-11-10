package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const exportfsFixture = `{"category": "nfs", "longdesc": " Exportfs uses the exportfs command to add/remove nfs exports. It does\nNOT manage the nfs server daemon. It depends on Linux specific NFS\nimplementation details, so is considered not portable to other\nplatforms yet.", "name": "exportfs", "shortdesc": "NFS Exported File System", "steps": [{"longdesc": "", "parameters": [{"advanced": false, "longdesc": "Cluster Resource ID", "name": "id", "required": false, "shortdesc": "Resource ID", "type": "resource", "unique": true, "value": "exportfs"}, {"advanced": false, "longdesc": " The client specification allowing remote machines to mount the\ndirectory (or directories) over NFS.\n\nNote: it follows the format defined in \"man exportfs\". For example, in\nthe use case to export the directory(-ies) for multiple subnets,\nplease do config a dedicated primitive for each subnet CIDR ip\naddress, and do not attempt to use multiple CIDR ip addresses in a\nspace separated list, like in /etc/exports.", "name": "clientspec", "required": true, "shortdesc": "Client ACL.", "type": "string", "unique": false}, {"advanced": false, "longdesc": "Any additional options to be given to the mount command, for example\nrw,mountpoint", "name": "options", "required": true, "shortdesc": "Mount Options", "type": "string", "unique": false}, {"advanced": false, "longdesc": "The mount point for the file system, e.g.: /srv/nfs/home", "name": "directory", "required": true, "shortdesc": "Mount Point (Directory)", "type": "string", "unique": false}, {"advanced": false, "longdesc": " The fsid option to pass to exportfs. This can be a unique positive\ninteger, a UUID (assuredly sans comma characters), or the special\nstring \"root\" which is functionally identical to numeric fsid of 0. If\nmultiple directories are being exported, then they are assigned ids\nsequentially starting with this fsid (fsid, fsid+1, fsid+2, ...).\nObviously, in that case the fsid must be an integer. 0 (root)\nidentifies the export as the root of an NFSv4 pseudofilesystem --\navoid this setting unless you understand its special status. This\nvalue will override any fsid provided via the options parameter.", "name": "fsid", "required": false, "shortdesc": "Unique FSID Within Cluster or Starting FSID for Multiple Exports", "type": "integer", "unique": false, "value": 1}, {"advanced": true, "example": "1", "longdesc": " Relinquish NFS locks associated with this filesystem when the\nresource stops. Enabling this parameter is highly recommended unless\nthe path exported by this exportfs resource is also exported by a\ndifferent resource.\n\nNote: Unlocking is only possible on Linux systems where\n/proc/fs/nfsd/unlock_filesystem exists and is writable. If your system\ndoes not fulfill this requirement (on account of having an nonrecent\nkernel, for example), you may set this parameter to 0 to silence the\nassociated warning.", "name": "unlock_on_stop", "required": false, "shortdesc": "Unlock filesystem on stop?", "type": "boolean", "unique": false}, {"advanced": false, "example": "0", "longdesc": "If set to true, wait for lease on stop.", "name": "wait_for_leasetime_on_stop", "required": false, "shortdesc": "Wait for Lease Time on Stop", "type": "boolean", "unique": false, "value": true}, {"advanced": true, "example": ".rmtab", "longdesc": " Back up those entries from the NFS rmtab that apply to the exported\ndirectory, to the specified backup file. The filename is interpreted\nas relative to the exported directory. This backup is required if\nclients are connecting to the export via NFSv3 over TCP. Note that a\nconfigured monitor operation is required for this functionality.\n\nTo disable rmtab backups, set this parameter to the special string\n\"none\".", "name": "rmtab_backup", "required": false, "shortdesc": "Location of the rmtab backup, relative to directory.", "type": "string", "unique": false}], "required": true, "shortdesc": "Manages NFS exports"}]}`

func TestExportFSForm(t *testing.T) {
	definition, err := loadResourceWizardDefinition(context.Background(), func(_ context.Context, op string, _ wizardParams) ([]byte, error) {
		require.Equal(t, "show", op)
		return []byte(exportfsFixture), nil
	}, exportFSConfig.Title)
	require.NoError(t, err)
	require.Len(t, definition.Steps, 1)
	parameters := definition.Steps[0].Parameters
	require.Len(t, parameters, 8)
	basic, advanced := []wizardParameter{}, []wizardParameter{}
	for _, p := range parameters {
		if p.Advanced {
			advanced = append(advanced, p)
		} else {
			basic = append(basic, p)
		}
	}
	require.Len(t, basic, 6)
	require.Len(t, advanced, 2)
	require.Equal(t, "true", advanced[0].Example)
	tmpl, err := template.ParseFiles("../templates/wizard_resource.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "exportfs", "Basic": basic, "Advanced": advanced, "Parameters": parameters, "Values": wizardParams{}}))
	for _, field := range basic {
		require.Contains(t, body.String(), `name="exportfs.`+field.Name+`"`)
		if field.Type == "boolean" {
			require.Contains(t, body.String(), `value="true" selected`)
		} else {
			require.Contains(t, body.String(), `value="`+field.Value+`"`)
		}
	}
	require.Contains(t, body.String(), `id="edit_wizard_exportfs"`)
	require.Contains(t, body.String(), `id="wizard-advanced"`)
	require.Contains(t, body.String(), `value="1"`)
	require.Contains(t, body.String(), `inputmode="numeric"`)
	params := wizardParams{"id": "export-data", "clientspec": "192.0.2.0/24", "options": "rw,mountpoint", "directory": "/srv/nfs/home", "fsid": "1", "wait_for_leasetime_on_stop": "true", "unlock_on_stop": "false"}
	require.NoError(t, validateResourceWizardParams(params, definition, exportFSConfig.Title))
	for _, name := range []string{"clientspec", "options", "directory"} {
		missing := wizardParams{}
		for key, value := range params {
			if key != name {
				missing[key] = value
			}
		}
		require.Error(t, validateResourceWizardParams(missing, definition, exportFSConfig.Title))
	}
	params["nodes"] = "node1"
	require.Error(t, validateResourceWizardParams(params, definition, exportFSConfig.Title))
}

func TestExportFSScriptCommand(t *testing.T) {
	params := wizardParams{"id": "export-data", "clientspec": "192.0.2.0/24", "options": "rw,mountpoint", "directory": "/srv/nfs/home", "fsid": "1", "wait_for_leasetime_on_stop": "true", "unlock_on_stop": "false"}
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := resourceWizardScriptCommand(context.Background(), "exportfs", op, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		stdin, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(stdin), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"exportfs"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var actual wizardParams
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}

func TestExportFSVerifyAndApply(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	params := wizardParams{"id": "export-data", "clientspec": "192.0.2.0/24", "options": "rw,mountpoint", "directory": "/srv/nfs/home", "fsid": "1", "wait_for_leasetime_on_stop": "true", "unlock_on_stop": "false"}
	for _, failure := range []string{"", "verify", "run"} {
		t.Run(failure, func(t *testing.T) {
			operations := []string{}
			run := func(_ context.Context, op string, actual wizardParams) ([]byte, error) {
				operations = append(operations, op)
				if op == "show" {
					return []byte(exportfsFixture), nil
				}
				require.Equal(t, params, actual)
				if op == failure {
					return nil, fmt.Errorf("ExportFS test failure")
				}
				if op == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure cluster resources","text":"primitive export-data ocf:heartbeat:exportfs"}`), nil
				}
				return []byte(`{"rc":true,"shortdesc":"Configure cluster resources"}`), nil
			}
			form := url.Values{}
			for key, value := range params {
				form.Set("exportfs."+key, value)
			}
			req := httptest.NewRequest("POST", "/cib/live/wizards/exportfs", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			serveResourceWizard(response, req, run, exportFSConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/exportfs/submit"`)
			if failure == "verify" {
				require.Contains(t, response.Body.String(), `disabled`)
			} else {
				require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
			}
			back := httptest.NewRequest("GET", "/cib/live/wizards/exportfs", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			require.Equal(t, params, recalledWizardValues(back, "exportfs"))
			operations = nil
			pa, _ := json.Marshal(params)
			req = httptest.NewRequest("POST", "/cib/live/wizards/exportfs/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveResourceWizard(response, req, run, exportFSConfig)
			status := 200
			if failure != "" {
				status = 422
			}
			require.Equal(t, status, response.Code)
			expected := []string{"show", "verify", "run"}
			if failure == "verify" {
				expected = expected[:2]
			}
			require.Equal(t, expected, operations)
		})
	}
}
