package api

import (
	"bytes"
	"html"
	"html/template"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWizardCategories(t *testing.T) {
	data := []byte(`{"name":"z","category":" DATABASE ","shortdesc":" Last ","longdesc":"Line one\nline two\n\nSecond paragraph"}
{"name":"a","category":"database","shortdesc":"First","longdesc":{"__content__":"Wrapped description"}}
{"name":"default","shortdesc":"Basic"}
{"name":"script","category":" Script ","shortdesc":"Hidden"}
{"name":"mariadb","category":"wizard","shortdesc":"Hidden workflow"}
{"name":"60-nfsserver","category":"wizard","shortdesc":"Hidden workflow"}
{"name":"ocfs2-single","category":"wizard","shortdesc":"Hidden workflow"}
{"name":"webserver","category":"wizard","shortdesc":"Hidden workflow"}
{"name":"legacy","category":"wizard","shortdesc":"Legacy"}
{"name":"filesystem","category":"File/System: Test","shortdesc":"Filesystem"}
{"name":"missing-title"}
{"error":"bad script"}
`)
	categories, err := wizardCategories(data)
	require.NoError(t, err)
	require.Len(t, categories, 4)
	require.Equal(t, "Basic", categories[0].Title)
	require.Equal(t, "basic", categories[0].ID)
	require.Equal(t, "Database", categories[1].Title)
	require.Equal(t, "database", categories[1].Icon)
	require.Equal(t, "a", categories[1].Wizards[0].Name)
	require.Equal(t, []string{"Wrapped description"}, categories[1].Wizards[0].Paragraphs)
	require.Equal(t, "Last", categories[1].Wizards[1].Title)
	require.Equal(t, []string{"Line one line two", "Second paragraph"}, categories[1].Wizards[1].Paragraphs)
	require.Equal(t, "File System Test", categories[2].Title)
	require.Equal(t, "file_system_test", categories[2].ID)
	require.Equal(t, "Legacy Wizards", categories[3].Title)
	require.Equal(t, "/cib/live/wizards/legacy", categories[3].Wizards[0].Href)
}

func TestWizardsTemplate(t *testing.T) {
	categories, err := wizardCategories([]byte(`{"name":"test","category":"database","shortdesc":"A wizard","longdesc":"<script>unsafe</script>"}`))
	require.NoError(t, err)
	tmpl, err := template.ParseFiles("../templates/wizards.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Categories": categories}))
	for _, value := range []string{`href="/cib/live/wizards/test"`, `aria-expanded="false"`, `<span class="wizard-category resource-type-link"`, `href="#database"`, `id="database"`, `data-toggle="collapse"`, `Database`, `A wizard`, `&lt;script&gt;unsafe&lt;/script&gt;`} {
		require.Contains(t, body.String(), value)
	}
	require.NotContains(t, body.String(), "<script>unsafe</script>")
}

func TestWizardsHandler(t *testing.T) {
	for _, tt := range []struct {
		method, path string
		status       int
	}{
		{"POST", "/cib/live/wizards", 405},
		{"GET", "/cib/live/wizards/test", 404},
	} {
		response := httptest.NewRecorder()
		WizardsHandler(response, httptest.NewRequest(tt.method, tt.path, nil))
		require.Equal(t, tt.status, response.Code)
	}

}

func TestFenceVirshFollowsLegacyLibvirt(t *testing.T) {
	categories, err := wizardCategories([]byte(`{"name":"vmware","category":"stonith","shortdesc":"VMware"}
{"name":"libvirt","category":"stonith","shortdesc":"STONITH for libvirt (kvm / Xen)"}
{"name":"sbd","category":"stonith","shortdesc":"SBD"}`))
	require.NoError(t, err)
	require.Len(t, categories, 1)
	require.Equal(t, "FENCING", categories[0].Title)
	require.Equal(t, "fencing", categories[0].ID)
	items := categories[0].Wizards
	require.Len(t, items, 5)
	require.Equal(t, "libvirt", items[0].Name)
	require.True(t, items[0].Disabled)
	require.Equal(t, "Deprecated: STONITH for libvirt (kvm / Xen)", items[0].Title)
	require.Equal(t, "fence-virsh", items[1].Name)
	require.False(t, items[1].Disabled)
	require.Equal(t, "Fencing for libvirt (fence_virsh)", items[1].Title)
	require.Equal(t, "/cib/live/wizards/libvirt", items[0].Href)
	require.Equal(t, "/cib/live/wizards/fence-virsh", items[1].Href)
	require.NotEqual(t, items[0].HelpID, items[1].HelpID)
	require.Equal(t, "sbd", items[2].Name)
}

func TestDeprecatedVMwareDisabledAndRESTListed(t *testing.T) {
	categories, err := wizardCategories([]byte(`{"name":"vmware","category":"stonith","shortdesc":"Fencing using vCenter / ESX Server"}`))
	require.NoError(t, err)
	require.Len(t, categories, 1)
	require.Equal(t, "FENCING", categories[0].Title)
	require.Equal(t, "fencing", categories[0].ID)
	items := categories[0].Wizards
	require.Len(t, items, 2)
	require.True(t, items[0].Disabled)
	require.Equal(t, deprecatedVMwareTitle, items[0].Title)
	require.Equal(t, "fence-vmware-rest", items[1].Name)
	require.False(t, items[1].Disabled)
	tmpl, err := template.ParseFiles("../templates/wizards.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", map[string]any{"Categories": categories}))
	require.Contains(t, body.String(), `aria-disabled="true"`)
	require.NotContains(t, body.String(), `href="/cib/live/wizards/vmware"`)
	require.Contains(t, body.String(), `href="/cib/live/wizards/fence-vmware-rest"`)
}

func TestFixedWizardCatalog(t *testing.T) {
	// The list is available even if crmsh cannot be executed on this machine.
	t.Setenv("PATH", t.TempDir())
	categories, err := wizardCategories(wizardCatalog)
	require.NoError(t, err)
	var names []string
	helpIDs := map[string]bool{}
	for _, category := range categories {
		require.NotEqual(t, "Stonith", category.Title)
		for _, wizard := range category.Wizards {
			names = append(names, wizard.Name)
			require.NotEmpty(t, wizard.Title)
			require.NotEmpty(t, wizard.Paragraphs, wizard.Name)
			require.False(t, helpIDs[wizard.HelpID], wizard.HelpID)
			helpIDs[wizard.HelpID] = true
		}
	}
	require.ElementsMatch(t, []string{
		"health", "mailto", "virtual-ip", "database", "db2", "db2-hadr", "oracle",
		"drbd", "filesystem", "gfs2", "lvm-drbd", "raid-lvm", "exportfs", "nfsserver", "nfsserver-lvm-drbd",
		"sap-as", "sap-ci", "sap-db", "sap-simple-stack", "sap-simple-stack-plus",
		"apache", "haproxy", "nginx", "libvirt", "fence-virsh", "sbd", "vmware", "fence-vmware-rest", "cryptctl",
	}, names)
}

func TestWizardsHandlerWithoutCRM(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	t.Setenv("PATH", t.TempDir())
	for _, path := range []string{"/cib/live/wizards", "/cib/live/wizards/"} {
		response := httptest.NewRecorder()
		WizardsHandler(response, httptest.NewRequest("GET", path, nil))
		require.Equal(t, 200, response.Code)
		require.Contains(t, response.Body.String(), `id="fencing"`)
		require.Contains(t, response.Body.String(), `FENCING`)
		require.NotContains(t, response.Body.String(), `id="stonith"`)
		require.Contains(t, response.Body.String(), `href="/cib/live/wizards/fence-virsh"`)
		require.Contains(t, response.Body.String(), `href="/cib/live/wizards/sap-simple-stack-plus"`)
		require.NotContains(t, response.Body.String(), `href="/cib/live/wizards/libvirt"`)
	}
}

func TestBatchWizardsMenu(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	t.Setenv("PATH", t.TempDir())
	categories, err := wizardCategories(wizardCatalog)
	require.NoError(t, err)
	count := 0
	for _, category := range categories {
		count += len(category.Wizards)
	}
	for _, path := range []string{batchBase + "/wizards", batchBase + "/wizards/"} {
		response := httptest.NewRecorder()
		WizardsHandler(response, httptest.NewRequest("GET", path, nil))
		require.Equal(t, 200, response.Code)
		body := response.Body.String()
		require.Equal(t, 2, strings.Count(body, `aria-disabled="true"`))
		require.Equal(t, count-2, strings.Count(body, `class="wizard-link"`))
		require.Contains(t, body, `href="/cib/hacluster/wizards/health"`)
		require.NotContains(t, body, `href="/cib/live/wizards/health"`)
		require.Contains(t, body, `id="fencing"`)
		require.Contains(t, body, `data-wizard-help=`)
		require.Contains(t, body, `id="btn-show-batch"`)
		require.Contains(t, body, `/static/js/batch.js`)
		for _, category := range categories {
			for _, wizard := range category.Wizards {
				require.Contains(t, html.UnescapeString(body), wizard.Title)
			}
		}
	}
	// Rendering batch must not disable a subsequent live request.
	response := httptest.NewRecorder()
	WizardsHandler(response, httptest.NewRequest("GET", "/cib/live/wizards", nil))
	require.Contains(t, response.Body.String(), `href="/cib/live/wizards/health"`)
	require.NotContains(t, response.Body.String(), `id="btn-show-batch"`)
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"POST", batchBase + "/wizards", 405},
		{"GET", batchBase + "/wizards/health", 404},
		{"POST", batchBase + "/wizards/health/submit", 404},
	} {
		response := httptest.NewRecorder()
		WizardsHandler(response, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, response.Code)
	}
}
