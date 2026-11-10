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

func lvmDRBDDefinition(t *testing.T, platform string) (resourceWizardDefinition, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/lvm_drbd_" + platform + ".json")
	require.NoError(t, err)
	var definition resourceWizardDefinition
	require.NoError(t, json.Unmarshal(data, &definition))
	return definition, data
}

func lvmDRBDForm(view db2View, filesystem bool) url.Values {
	values := url.Values{}
	for _, field := range view.Fields {
		if !field.Advanced && field.Current != "" {
			values.Set(field.FieldID, field.Current)
		}
	}
	values.Set("lvm-drbd.lvm.vgname", "vg1")
	values.Set("lvm-drbd.lvm.vg_access_mode", "system_id")
	values.Set("lvm-drbd.example_fs.id", "example-fs")
	values.Set("lvm-drbd.example_fs.fast_stop", "false")
	values.Set("enable:lvm-drbd.example_fs", strconv.FormatBool(filesystem))
	return values
}

func TestLVMDRBDOptionalStep(t *testing.T) {
	for _, platform := range []string{"leap", "tumbleweed"} {
		t.Run(platform, func(t *testing.T) {
			definition, _ := lvmDRBDDefinition(t, platform)
			view := buildNestedWizardView(definition, nil, "lvm-drbd")
			require.Len(t, view.Steps, 4)
			require.Equal(t, "lvm drbd", view.Steps[0].Title)
			require.True(t, view.Steps[3].Optional)
			require.False(t, view.Steps[3].Enabled)
			require.Equal(t, "enable:lvm-drbd.example_fs", view.Steps[3].EnableID)
			for _, enabled := range []bool{false, true} {
				req := httptest.NewRequest("POST", "/", nil)
				req.Form = lvmDRBDForm(view, enabled)
				params := db2FormParams(req, definition, view)
				require.NoError(t, validateDB2Params(params, view))
				_, present := params["example_fs"]
				require.Equal(t, enabled, present)
				require.Equal(t, "g-lvm", params["group_id"])
				require.Equal(t, "vg1", params["drbd"].(map[string]any)["drbd_resource"])
				restored := buildNestedWizardView(definition, params, "lvm-drbd")
				require.Equal(t, enabled, restored.Steps[3].Enabled)
				if enabled {
					require.Equal(t, "false", db2Value(params, []string{"example_fs", "fast_stop"}))
					params["example_fs"].(map[string]any)["id"] = ""
					require.Error(t, validateDB2Params(params, view))
				}
			}
			req := httptest.NewRequest("POST", "/", nil)
			req.Form = lvmDRBDForm(view, false)
			params := db2FormParams(req, definition, view)
			params["lvm"].(map[string]any)["nodes"] = "other"
			require.Error(t, validateDB2Params(params, view))
			if platform == "tumbleweed" {
				params = db2FormParams(req, definition, view)
				delete(params["lvm"].(map[string]any), "vgname")
				require.Error(t, validateDB2Params(params, view))
			}
			tmpl, err := template.ParseFiles("../templates/wizard_db2.html")
			require.NoError(t, err)
			var body bytes.Buffer
			require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Title": definition.Shortdesc, "WizardName": "lvm-drbd", "Steps": view.Steps, "Fields": view.Fields, "AdvancedGroups": view.Advanced}))
			for _, expected := range []string{`id="edit_wizard_lvm-drbd"`, `name="lvm-drbd.group_id"`, `name="lvm-drbd.drbd.id"`, `name="lvm-drbd.lvm.volgrpname"`, `name="lvm-drbd.example_fs.id"`, `name="enable:lvm-drbd.example_fs"`, `data-wizard-enable="true"`, `value="false" selected`, `id="step3"`} {
				require.Contains(t, body.String(), expected)
			}
		})
	}
}

func TestLVMDRBDCommand(t *testing.T) {
	params := db2Params{"group_id": "g-lvm", "drbd": map[string]any{"id": "drbd-data", "drbd_resource": "vg1"}, "lvm": map[string]any{"id": "lvm", "volgrpname": "vg1"}}
	for _, op := range []string{"show", "verify", "run"} {
		cmd, err := nestedWizardScriptCommand(context.Background(), "lvm-drbd", op, params)
		require.NoError(t, err)
		require.Equal(t, []string{"/usr/sbin/crm", "-f", "/dev/stdin"}, cmd.Args)
		input, err := io.ReadAll(cmd.Stdin)
		require.NoError(t, err)
		payload, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(string(input), "script json ")))
		require.NoError(t, err)
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(payload), &args))
		require.JSONEq(t, `"lvm-drbd"`, string(args[1]))
		if op == "show" {
			require.Len(t, args, 2)
		} else {
			var actual db2Params
			require.NoError(t, json.Unmarshal(args[2], &actual))
			require.Equal(t, params, actual)
		}
	}
}

func TestLVMDRBDVerifyApplyAndBack(t *testing.T) {
	definition, fixture := lvmDRBDDefinition(t, "leap")
	view := buildNestedWizardView(definition, nil, "lvm-drbd")
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, enabled := range []bool{false, true} {
		for _, failure := range []string{"", "verify", "run", "partial"} {
			t.Run(fmt.Sprintf("%t/%s", enabled, failure), func(t *testing.T) {
				form := lvmDRBDForm(view, enabled)
				req := httptest.NewRequest("POST", "/cib/live/wizards/lvm-drbd", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				require.NoError(t, req.ParseForm())
				expected := db2FormParams(req, definition, view)
				operations := []string{}
				run := func(_ context.Context, op string, params db2Params) ([]byte, error) {
					operations = append(operations, op)
					if op == "show" {
						return fixture, nil
					}
					require.Equal(t, expected, params)
					if failure == op {
						return nil, fmt.Errorf("test command failed")
					}
					if op == "verify" {
						return []byte("{\"name\":\"cib\",\"text\":\"primitive drbd-data ocf:linbit:drbd\"}\n{\"name\":\"cib\"}\n{\"name\":\"cib\"}"), nil
					}
					if failure == "partial" {
						return []byte(`{"rc":true}`), nil
					}
					return []byte("{\"rc\":true}\n{\"rc\":true}\n{\"rc\":true}"), nil
				}
				response := httptest.NewRecorder()
				serveNestedWizard(response, req, run, lvmDRBDConfig)
				require.Equal(t, 200, response.Code)
				require.Equal(t, []string{"show", "verify"}, operations)
				require.Contains(t, response.Body.String(), `action="/cib/live/wizards/lvm-drbd/submit"`)
				back := httptest.NewRequest("GET", "/cib/live/wizards/lvm-drbd", nil)
				for _, cookie := range response.Result().Cookies() {
					back.AddCookie(cookie)
				}
				var remembered db2Params
				require.NoError(t, json.Unmarshal(recalledEncryptedWizardValues(back, "lvm-drbd"), &remembered))
				require.Equal(t, expected, remembered)
				operations = nil
				pa, _ := json.Marshal(expected)
				apply := httptest.NewRequest("POST", "/cib/live/wizards/lvm-drbd/submit", strings.NewReader(url.Values{"pa": {string(pa)}}.Encode()))
				apply.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response = httptest.NewRecorder()
				serveNestedWizard(response, apply, run, lvmDRBDConfig)
				status := 200
				if failure != "" {
					status = 422
				}
				require.Equal(t, status, response.Code)
				expectedOps := []string{"show", "verify", "run"}
				if failure == "verify" {
					expectedOps = expectedOps[:2]
				}
				require.Equal(t, expectedOps, operations)
			})
		}
	}
}

func TestLVMDRBDExamples(t *testing.T) {
	for _, platform := range []string{"leap", "tumbleweed"} {
		definition, _ := lvmDRBDDefinition(t, platform)
		original, _ := json.Marshal(definition)
		addLVMDRBDExamples(&definition)
		if platform == "leap" {
			actual, _ := json.Marshal(definition)
			require.JSONEq(t, string(original), string(actual))
			continue
		}
		view := buildNestedWizardView(definition, nil, "lvm-drbd")
		fields := map[string]db2Field{}
		for _, field := range view.Fields {
			fields[field.FieldID] = field
		}
		require.Equal(t, "vg1", fields["lvm-drbd.lvm.vgname"].Example)
		require.Equal(t, "system_id", fields["lvm-drbd.lvm.vg_access_mode"].Example)
		require.Empty(t, fields["lvm-drbd.lvm.vg_access_mode"].Current)
		definition.Steps[2].Steps[0].Parameters[1].Example = "custom-example"
		addLVMDRBDExamples(&definition)
		require.Equal(t, "custom-example", definition.Steps[2].Steps[0].Parameters[1].Example)
	}
}
