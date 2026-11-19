package api

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestBatchPrimitiveSections(t *testing.T) {
	for _, section := range []string{"params", "meta-attributes", "operations", "utilizations"} {
		t.Run(section, func(t *testing.T) {
			b, calls := batchTestBackend(t)
			original := b.run
			b.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
				out, err := original(ctx, live, name, args, input)
				if name == "cibadmin" && len(args) > 0 && args[0] == "-Q" {
					return []byte(`<primitive id="r1" class="ocf" provider="pacemaker" type="Dummy"><instance_attributes id="params"><nvpair id="p" name="state" value="old"/></instance_attributes><meta_attributes id="meta"><nvpair id="m" name="target-role" value="Stopped"/></meta_attributes><operations><op id="monitor" name="monitor" interval="10s"/></operations><utilization id="u"><nvpair id="cpu" name="cpu" value="1"/></utilization></primitive>`), nil
				}
				return out, err
			}
			body := `{"id":"r1","instance_attributes":{"nvpair":[{"name":"state","value":"new"}]},"meta_attributes":{"nvpair":[{"name":"target-role","value":"Started"}]},"operations":[{"name":"monitor","interval":"20s"}]}`
			if section == "utilizations" {
				body = `{"CibObject":"r1","nvpair":[{"name":"cpu","value":"2"}]}`
			}
			response := batchTestRequest(b, "POST", "resource/"+section+"/submit", body)
			require.Equal(t, 200, response.Code, response.Body.String())
			last := (*calls)[len(*calls)-1]
			require.Equal(t, "--replace", last.args[0])
			xml := last.args[len(last.args)-1]
			require.Contains(t, xml, `type="Dummy"`)
			require.Contains(t, xml, `<primitive`)
			for _, call := range *calls {
				require.False(t, call.live)
			}
			if section != "meta-attributes" {
				require.Contains(t, xml, `value="Stopped"`)
			}
			if section != "params" {
				require.Contains(t, xml, `value="old"`)
			}
			require.NotContains(t, strings.Join(last.args, " "), "/dev/stdin")
		})
	}
}

func TestBatchPrimitiveEditTemplate(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	defer os.Chdir(cwd)
	response := httptest.NewRecorder()
	servePrimitiveEdit(response, httptest.NewRequest("GET", batchBase+"/primitives/r1/edit?flash=created", nil), func() ([]ResourceRow, error) {
		return []ResourceRow{{ID: "r1", Class: "ocf", Provider: "pacemaker", Type: "Dummy"}}, nil
	})
	require.Equal(t, 200, response.Code)
	for _, s := range []string{"Primitive created successfully", batchAPI + "/resource/params/submit", batchAPI + "/resource/operations/fetch", batchAPI + "/resource/utilizations/submit", batchBase + "/config/edit", "batch.js"} {
		require.Contains(t, response.Body.String(), s)
	}
	require.NotContains(t, response.Body.String(), `submit-api="/api/cib/resource/`)
}

func TestPrimitiveMetadataShadowValues(t *testing.T) {
	var metadata FullPrimitive_CrmResourceMetadata
	calls := 0
	err := enrichPrimitiveMetadataUsing(&metadata, "r1", func(name string, args []string) ([]byte, error) {
		calls++
		require.Equal(t, "cibadmin", name)
		require.Contains(t, args[2], "primitive[@id='r1']")
		return []byte(`<primitive id="r1"><instance_attributes id="p"><nvpair id="state" name="state" value="shadow-state"/></instance_attributes><meta_attributes id="m"><nvpair id="role" name="target-role" value="Stopped"/></meta_attributes></primitive>`), nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, "shadow-state", metadata.Parameters[0].Content.CibValue)
	require.Equal(t, "Stopped", metadata.MetaAttributes[0].Content.CibValue)
}
