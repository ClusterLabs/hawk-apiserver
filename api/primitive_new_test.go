package api

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrimitiveFormModes(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/layout.html", "../templates/primitive_edit.html")
	require.NoError(t, err)
	for _, create := range []bool{true, false} {
		var body bytes.Buffer
		require.NoError(t, tmpl.ExecuteTemplate(&body, "layout.html", map[string]any{"CreateMode": create, "ResourceID": "dummy", "ResourceAgent": "ocf:pacemaker:Dummy", "Class": "ocf", "Provider": "pacemaker", "Type": "Dummy"}))
		if create {
			require.Contains(t, body.String(), "Create Primitive")
			require.NotContains(t, body.String(), `id="copy-rename-delete-container"`)
			require.Contains(t, body.String(), `href="/cib/live/resources/types"`)
			require.Contains(t, body.String(), `<div id="primitive-fields"></div>`)
		} else {
			require.Contains(t, body.String(), "Edit Primitive: dummy")
			require.Contains(t, body.String(), `id="copy-rename-delete-container"`)
			require.Contains(t, body.String(), `href="/cib/live/config/edit"`)
		}
	}
}

func TestPrimitiveMetadataForNewResource(t *testing.T) {
	// A blank ID must not invoke cibadmin for a primitive which does not exist.
	t.Setenv("PATH", t.TempDir())
	metadata := FullPrimitive_CrmResourceMetadata{Name: "Dummy"}
	require.NoError(t, enrichPrimitiveMetadataWithCibValues(&metadata, ""))
	require.Equal(t, "Dummy", metadata.Name)
}

func TestPrimitiveCreateValidation(t *testing.T) {
	for _, body := range []string{`{}`, `{"id":"bad id","class":"ocf","provider":"pacemaker","type":"Dummy"}`, `{"id":"dummy","class":"ocf","type":"Dummy"}`} {
		response := httptest.NewRecorder()
		PrimitiveCreateHandler(response, httptest.NewRequest(http.MethodPost, "/api/cib/primitive/create", strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, response.Code)
	}
	response := httptest.NewRecorder()
	PrimitiveCreateHandler(response, httptest.NewRequest(http.MethodGet, "/api/cib/primitive/create", nil))
	require.Equal(t, http.StatusMethodNotAllowed, response.Code)
}

func TestPrimitiveCreateIncludesAllSections(t *testing.T) {
	primitive := Primitive{ID: "dummy", Class: "ocf", Provider: "pacemaker", Type: "Dummy"}
	primitive.InstanceAttributes.NVPairs = []Nvpair{{Name: "state", Value: "/tmp/a b"}}
	primitive.MetaAttributes.NVPairs = []Nvpair{{Name: "target-role", Value: "Stopped"}, {Name: "empty", Value: ""}}
	primitive.Operations = []Operation{{Name: "monitor", Interval: "10s", IntervalOrigin: "2026-01-01", Timeout: "20s"}}
	primitive.Utilizations = []Nvpair{{Name: "cpu", Value: "2"}}
	require.Equal(t, []string{"configure", "primitive", "dummy", "ocf:pacemaker:Dummy", "params", "state=/tmp/a b", "op", "monitor", "interval=10s", "interval-origin=2026-01-01", "timeout=20s", "meta", "target-role=Stopped", "utilization", "cpu=2"}, primitiveCreateArgs(primitive))
	require.Equal(t, []string{"configure", "primitive", "web", "systemd:apache2"}, primitiveCreateArgs(Primitive{ID: "web", Class: "systemd", Type: "apache2"}))
}

func TestPrimitiveCreateMonitorDepthFollowsOperationAttributes(t *testing.T) {
	primitive := Primitive{ID: "dummy", Class: "ocf", Provider: "heartbeat", Type: "Dummy"}
	primitive.Operations = []Operation{
		{Name: "start", Interval: "0", Timeout: "35s"},
		{Name: "stop", Interval: "0", Timeout: "15s", OnFail: "stop"},
		{Name: "monitor", Depth: "0", Interval: "13s", Timeout: "9s"},
	}
	primitive.MetaAttributes.NVPairs = []Nvpair{{Name: "target-role", Value: "Started"}}

	require.Equal(t, []string{
		"configure", "primitive", "dummy", "ocf:heartbeat:Dummy",
		"op", "start", "interval=0", "timeout=35s",
		"op", "stop", "interval=0", "on-fail=stop", "timeout=15s",
		"op", "monitor", "interval=13s", "timeout=9s", "depth=0",
		"meta", "target-role=Started",
	}, primitiveCreateArgs(primitive))
}
