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

const filesystemFixture = `{"category": "file system", "longdesc": " Resource script for Filesystem. It manages a Filesystem on a shared\nstorage medium.\n\nThe standard monitor operation of depth 0 (also known as probe) checks\nif the filesystem is mounted. If you want deeper tests, set\nOCF_CHECK_LEVEL to one of the following values:\n\n10: read first 16 blocks of the device (raw read)\n\nThis doesn't exercise the filesystem at all, but the device on which\nthe filesystem lives. This is noop for non-block devices such as NFS,\nSMBFS, or bind mounts.\n\n20: test if a status file can be written and read\n\nThe status file must be writable by root. This is not always the case\nwith an NFS mount, as NFS exports usually have the \"root_squash\"\noption set. In such a setup, you must either use read-only monitoring\n(depth=10), export with \"no_root_squash\" on your NFS server, or grant\nworld write permissions on the directory where the status file is to\nbe placed.", "name": "filesystem", "shortdesc": "File System (mount point)", "steps": [{"longdesc": "", "parameters": [{"advanced": false, "longdesc": "Identifier for the cluster resource", "name": "id", "required": true, "shortdesc": "Identifier for the cluster resource", "type": "resource", "unique": true}, {"advanced": false, "example": "", "longdesc": "block device", "name": "device", "required": true, "shortdesc": "block device", "type": "string", "unique": false}, {"advanced": false, "example": "", "longdesc": "mount point", "name": "directory", "required": true, "shortdesc": "mount point", "type": "string", "unique": false}, {"advanced": false, "example": "", "longdesc": "filesystem type", "name": "fstype", "required": true, "shortdesc": "filesystem type", "type": "string", "unique": false}, {"advanced": false, "example": "", "longdesc": "options", "name": "options", "required": false, "shortdesc": "options", "type": "string", "unique": false}, {"advanced": true, "example": ".Filesystem_status/", "longdesc": "status file prefix", "name": "statusfile_prefix", "required": false, "shortdesc": "status file prefix", "type": "string", "unique": false}, {"advanced": true, "example": "auto", "longdesc": "run_fsck", "name": "run_fsck", "required": false, "shortdesc": "run_fsck", "type": "string", "unique": false}, {"advanced": true, "example": "no", "longdesc": "fast stop", "name": "fast_stop", "required": false, "shortdesc": "fast stop", "type": "boolean", "unique": false}, {"advanced": true, "example": "false", "longdesc": "allow running as a clone, regardless of filesystem type", "name": "force_clones", "required": false, "shortdesc": "allow running as a clone, regardless of filesystem type", "type": "boolean", "unique": false}, {"advanced": true, "example": "true", "longdesc": "Kill processes before unmount", "name": "force_unmount", "required": false, "shortdesc": "Kill processes before unmount", "type": "string", "unique": false}, {"advanced": true, "example": "TERM", "longdesc": "Signals (names or numbers, whitespace separated) to send processes during graceful termination phase in stop-action", "name": "term_signals", "required": false, "shortdesc": "Signals (names or numbers, whitespace separated) to send processes during graceful termination phase in stop-action", "type": "string", "unique": false}, {"advanced": true, "example": "KILL", "longdesc": "Signals (names or numbers, whitespace separated) to send processes during forceful killing phase in stop-action", "name": "kill_signals", "required": false, "shortdesc": "Signals (names or numbers, whitespace separated) to send processes during forceful killing phase in stop-action", "type": "string", "unique": false}, {"advanced": true, "example": "1", "longdesc": "How many seconds to wait after sending term/kill signals to processes in stop-action", "name": "signal_delay", "required": false, "shortdesc": "How many seconds to wait after sending term/kill signals to processes in stop-action", "type": "string", "unique": false}], "required": true, "shortdesc": "Manages filesystem mounts"}]}`

func TestFilesystemForm(t *testing.T) {
	definition, err := loadResourceWizardDefinition(context.Background(), func(_ context.Context, op string, _ wizardParams) ([]byte, error) {
		require.Equal(t, "show", op)
		return []byte(filesystemFixture), nil
	}, filesystemConfig.Title)
	require.NoError(t, err)
	require.Len(t, definition.Steps, 1)
	parameters := definition.Steps[0].Parameters
	require.Len(t, parameters, 13)
	tmpl, err := template.ParseFiles("../templates/wizard_resource.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "filesystem", "Basic": parameters[:5], "Advanced": parameters[5:], "Parameters": parameters, "Values": wizardParams{}}))
	for _, field := range parameters[:5] {
		require.Contains(t, body.String(), `name="filesystem.`+field.Name+`"`)
		require.Contains(t, body.String(), `value="`+field.Value+`"`)
	}
	require.Contains(t, body.String(), `id="edit_wizard_filesystem"`)
	require.Contains(t, body.String(), `id="wizard-advanced"`)
	require.Equal(t, "false", parameters[7].Example)
	require.Equal(t, "fast stop", parameters[7].Shortdesc)
	require.Contains(t, body.String(), `filesystem-help-fast_stop`)
	params := wizardParams{"id": "fs-data", "device": "/dev/disk/by-label/data", "directory": "/srv/data", "fstype": "xfs", "options": "noatime", "fast_stop": "false"}
	require.NoError(t, validateResourceWizardParams(params, definition, filesystemConfig.Title))
	for _, name := range []string{"id", "device", "directory", "fstype"} {
		missing := wizardParams{}
		for key, value := range params {
			if key != name {
				missing[key] = value
			}
		}
		require.Error(t, validateResourceWizardParams(missing, definition, filesystemConfig.Title))
	}
	params["nodes"] = "node1"
	require.Error(t, validateResourceWizardParams(params, definition, filesystemConfig.Title))
}

func TestFilesystemScriptCommand(t *testing.T) {
	params := wizardParams{"id": "fs-data", "device": "/dev/disk/by-label/data", "directory": "/srv/data", "fstype": "xfs", "options": "noatime", "fast_stop": "false"}
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := resourceWizardScriptCommand(context.Background(), "filesystem", op, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		stdin, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(stdin), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"filesystem"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var actual wizardParams
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}

func TestFilesystemVerifyAndApply(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	params := wizardParams{"id": "fs-data", "device": "/dev/disk/by-label/data", "directory": "/srv/data", "fstype": "xfs", "options": "noatime", "fast_stop": "false"}
	for _, failure := range []string{"", "verify", "run"} {
		t.Run(failure, func(t *testing.T) {
			operations := []string{}
			run := func(_ context.Context, op string, actual wizardParams) ([]byte, error) {
				operations = append(operations, op)
				if op == "show" {
					return []byte(filesystemFixture), nil
				}
				require.Equal(t, params, actual)
				if op == failure {
					return nil, fmt.Errorf("Filesystem test failure")
				}
				if op == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure cluster resources","text":"primitive fs-data ocf:heartbeat:Filesystem"}`), nil
				}
				return []byte(`{"rc":true,"shortdesc":"Configure cluster resources"}`), nil
			}
			form := url.Values{}
			for key, value := range params {
				form.Set("filesystem."+key, value)
			}
			req := httptest.NewRequest("POST", "/cib/live/wizards/filesystem", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			serveResourceWizard(response, req, run, filesystemConfig)
			require.Equal(t, 200, response.Code)
			require.Equal(t, []string{"show", "verify"}, operations)
			require.Contains(t, response.Body.String(), `action="/cib/live/wizards/filesystem/submit"`)
			if failure == "verify" {
				require.Contains(t, response.Body.String(), `disabled`)
			} else {
				require.Contains(t, response.Body.String(), `class="crmsh hljs"`)
			}
			back := httptest.NewRequest("GET", "/cib/live/wizards/filesystem", nil)
			for _, cookie := range response.Result().Cookies() {
				back.AddCookie(cookie)
			}
			require.Equal(t, params, recalledWizardValues(back, "filesystem"))
			operations = nil
			pa, _ := json.Marshal(params)
			req = httptest.NewRequest("POST", "/cib/live/wizards/filesystem/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			serveResourceWizard(response, req, run, filesystemConfig)
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
