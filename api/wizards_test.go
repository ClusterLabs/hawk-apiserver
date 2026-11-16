package api

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http/httptest"
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
		serveWizards(response, httptest.NewRequest(tt.method, tt.path, nil), func(context.Context) ([]byte, error) { t.Fatal("unexpected command"); return nil, nil })
		require.Equal(t, tt.status, response.Code)
	}
	response := httptest.NewRecorder()
	serveWizards(response, httptest.NewRequest("GET", "/cib/live/wizards", nil), func(context.Context) ([]byte, error) { return nil, fmt.Errorf("crmsh unavailable") })
	require.Equal(t, 500, response.Code)
	require.Contains(t, response.Body.String(), "Unable to load wizards")
}

func TestFenceVirshFollowsLegacyLibvirt(t *testing.T) {
	categories, err := wizardCategories([]byte(`{"name":"vmware","category":"stonith","shortdesc":"VMware"}
{"name":"libvirt","category":"stonith","shortdesc":"STONITH for libvirt (kvm / Xen)"}
{"name":"sbd","category":"stonith","shortdesc":"SBD"}`))
	require.NoError(t, err)
	require.Len(t, categories, 1)
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
