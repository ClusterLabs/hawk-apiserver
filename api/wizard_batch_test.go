package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWizardCommandCIBIsolation(t *testing.T) {
	t.Setenv("CIB_shadow", "wrong")
	t.Setenv("CIB_file", "wrong.xml")
	t.Setenv("CIB_shadow_dir", "/wrong")
	for _, batchMode := range []bool{false, true} {
		ctx := context.WithValue(context.Background(), wizardBatchKey{}, batchMode)
		for name := range wizardHandlers {
			if name == "health" {
				continue
			}
			if name == "fence-virsh" {
				name = "libvirt"
			}
			cmd, err := nestedWizardScriptCommand(ctx, name, "run", map[string]any{"id": "proof"})
			require.NoError(t, err)
			require.Equal(t, batchMode, wizardEnvValue(cmd.Env, "CIB_shadow") == batchID)
			require.Equal(t, "", wizardEnvValue(cmd.Env, "CIB_file"))
			require.Equal(t, batchShadowDirectory, wizardEnvValue(cmd.Env, "CIB_shadow_dir"))
		}
		cmd, err := healthScriptCommand(ctx, "run")
		require.NoError(t, err)
		require.Equal(t, batchMode, wizardEnvValue(cmd.Env, "CIB_shadow") == batchID)
	}
	require.Equal(t, "wrong", os.Getenv("CIB_shadow"))
}
func wizardEnvValue(env []string, name string) string {
	for _, value := range env {
		if strings.HasPrefix(value, name+"=") {
			return strings.TrimPrefix(value, name+"=")
		}
	}
	return ""
}

func TestBatchWizardSharedFlows(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	definition := []byte(`{"shortdesc":"Wizard","steps":[{"parameters":[{"name":"id","type":"resource","required":true}]}]}`)
	resourceNames := map[string]bool{"virtual-ip": true, "database": true, "oracle": true, "drbd": true, "filesystem": true, "gfs2": true, "exportfs": true, "haproxy": true, "nginx": true}
	for name := range wizardHandlers {
		if name == "libvirt" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			var operations []string
			runner := func(ctx context.Context, op string) ([]byte, error) {
				require.True(t, wizardIsBatch(ctx))
				operations = append(operations, op)
				if op == "show" {
					if name == "health" {
						return []byte(`{"shortdesc":"Health","steps":[]}`), nil
					}
					return definition, nil
				}
				if op == "verify" {
					return []byte(`{"name":"cib","shortdesc":"Configure","text":"primitive proof ocf:pacemaker:Dummy"}`), nil
				}
				return []byte(`{"rc":true,"shortdesc":"Configure"}`), nil
			}
			flat := func(ctx context.Context, op string, params wizardParams) ([]byte, error) { return runner(ctx, op) }
			nested := func(ctx context.Context, op string, params db2Params) ([]byte, error) { return runner(ctx, op) }
			handler := func(w http.ResponseWriter, r *http.Request) {
				switch {
				case name == "health":
					serveHealthWizard(w, r, runner)
				case name == "mailto":
					serveMailtoWizard(w, r, flat)
				case resourceNames[name]:
					serveResourceWizard(w, r, flat, resourceWizardConfig{Name: name, Title: name})
				default:
					serveNestedWizard(w, r, nested, resourceWizardConfig{Name: name, Title: name})
				}
			}
			base := batchBase + "/wizards/" + name
			w := httptest.NewRecorder()
			handler(w, httptest.NewRequest("GET", base, nil))
			require.Equal(t, 200, w.Code)
			require.Contains(t, w.Body.String(), `action="`+base+`"`)
			require.Contains(t, w.Body.String(), `id="btn-show-batch"`)
			form := url.Values{name + ".id": {"proof"}}
			if name == "mailto" {
				form.Set("mailto.email", "test@example.test")
			}
			r := httptest.NewRequest("POST", base, strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w = httptest.NewRecorder()
			handler(w, r)
			require.Equal(t, 200, w.Code)
			require.Contains(t, w.Body.String(), `action="`+base+`/submit"`)
			require.Contains(t, w.Body.String(), `href="`+batchBase+`/wizards"`)
			require.NotContains(t, w.Body.String(), `action="/cib/live/`)
			for _, cookie := range w.Result().Cookies() {
				require.Equal(t, base, cookie.Path)
			}
			params := `{"id":"proof"}`
			if name == "health" {
				params = `{}`
			} else if name == "mailto" {
				params = `{"id":"proof","email":"test@example.test"}`
			}
			r = httptest.NewRequest("POST", base+"/submit", strings.NewReader(url.Values{"pa": {params}}.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w = httptest.NewRecorder()
			handler(w, r)
			require.Equal(t, 200, w.Code, w.Body.String())
			require.Contains(t, operations, "run")
		})
	}
}

// Opt-in test uses only a disposable shadow directory and a synthetic script.
// No live CIB query, host action, real shadow or resource agent execution occurs.
func TestBatchWizardOfflineShadow(t *testing.T) {
	if os.Getenv("HAWK_WIZARD_OFFLINE_TEST") != "1" {
		t.Skip("Set HAWK_WIZARD_OFFLINE_TEST=1 to test installed crmsh with a disposable shadow")
	}
	dir := t.TempDir()
	baseline := `<cib crm_feature_set="3.20.5" validate-with="pacemaker-4.0" admin_epoch="0" epoch="1" num_updates="0"><configuration><crm_config/><nodes/><resources/><constraints/></configuration><status/></cib>`
	live := filepath.Join(dir, "live.xml")
	shadow := filepath.Join(dir, "shadow."+batchID)
	require.NoError(t, os.WriteFile(live, []byte(baseline), 0600))
	require.NoError(t, os.WriteFile(shadow, []byte(baseline), 0600))
	script := `version: 2.2
shortdesc: Offline proof
actions:
  - cib: primitive hawk-wizard-proof ocf:pacemaker:Dummy meta target-role=Stopped
    force: true
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "database.yml"), []byte(script), 0600))
	ctx := context.WithValue(context.Background(), wizardBatchKey{}, true)
	hostname, err := os.Hostname()
	require.NoError(t, err)
	cmd, err := nestedWizardScriptCommand(ctx, "database", "run", map[string]any{"nodes": hostname})
	require.NoError(t, err)
	cmd.Args = append([]string{cmd.Args[0], "--scriptdir", dir}, cmd.Args[1:]...)
	cmd.Env = append(cmd.Env, "CIB_shadow_dir="+dir)
	out, err := runWizardProcess(cmd)
	require.NoError(t, err, string(out))
	require.Contains(t, string(out), `"rc": true`)
	changed, err := os.ReadFile(shadow)
	require.NoError(t, err)
	require.Contains(t, string(changed), "hawk-wizard-proof")
	unchanged, err := os.ReadFile(live)
	require.NoError(t, err)
	require.Equal(t, baseline, string(unchanged))
	// The live constructor must remove the shadow variables even in a batch environment.
	t.Setenv("CIB_shadow", batchID)
	ordinary, err := nestedWizardScriptCommand(context.Background(), "database", "show", nil)
	require.NoError(t, err)
	require.Empty(t, wizardEnvValue(ordinary.Env, "CIB_shadow"))
}

func TestWizardRoutesInitializeShadow(t *testing.T) {
	originalBackend, originalHandler := batch, wizardHandlers["health"]
	t.Cleanup(func() { batch = originalBackend; wizardHandlers["health"] = originalHandler })
	created, called := 0, 0
	batch = &batchBackend{
		exists: func() (bool, error) { return false, nil },
		run: func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
			created++
			require.True(t, live)
			require.Equal(t, "crm_shadow", name)
			require.Equal(t, []string{"-b", "-f", "-c", batchID}, args)
			return nil, nil
		},
	}
	wizardHandlers["health"] = func(w http.ResponseWriter, r *http.Request) {
		called++
		require.Equal(t, strings.HasPrefix(r.URL.Path, batchBase+"/"), wizardIsBatch(r.Context()))
		w.WriteHeader(200)
	}
	w := httptest.NewRecorder()
	WizardRoutesHandler(w, httptest.NewRequest("GET", batchBase+"/wizards/health", nil))
	require.Equal(t, 200, w.Code)
	require.Equal(t, 1, created)
	require.Equal(t, 1, called)
	w = httptest.NewRecorder()
	WizardRoutesHandler(w, httptest.NewRequest("GET", "/cib/live/wizards/health", nil))
	require.Equal(t, 200, w.Code)
	require.Equal(t, 1, created)
	require.Equal(t, 2, called)
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", batchBase + "/wizards/health/submit", 405},
		{"POST", batchBase + "/wizards/health/other", 404},
		{"GET", batchBase + "/wizards/libvirt", 404},
		{"GET", batchBase + "/wizards/not-found", 404},
	} {
		w = httptest.NewRecorder()
		WizardRoutesHandler(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, w.Code)
	}
	require.Equal(t, 1, created)
	require.Equal(t, 2, called)
	// Every enabled catalog entry is served by the same dispatcher in both modes.
	categories, err := wizardCategories(wizardCatalog)
	require.NoError(t, err)
	for _, category := range categories {
		for _, wizard := range category.Wizards {
			if !wizard.Disabled {
				require.Contains(t, wizardHandlers, wizard.Name)
			}
		}
	}
}
