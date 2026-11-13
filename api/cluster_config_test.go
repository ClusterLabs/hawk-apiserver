package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const clusterConfigMetadataFixture = `<pacemaker-result><resource-agent><parameters>
<parameter name="dc-version"><shortdesc lang="en">Version</shortdesc><content type="string"/></parameter>
<parameter name="cluster-name"><shortdesc lang="en">Cluster name</shortdesc><longdesc lang="en">Name of the cluster</longdesc><content type="string"/></parameter>
<parameter name="maintenance-mode"><shortdesc lang="en">Maintenance</shortdesc><content type="boolean" default="false"/></parameter>
<parameter name="no-quorum-policy"><longdesc lang="en">Choose a policy</longdesc><content type="enum" default="stop"><option value="stop"/><option value="ignore"/></content></parameter>
<parameter name="advanced-option" advanced="1"><content type="string"/></parameter>
<parameter name="advanced-text"><longdesc lang="en">For advanced use only</longdesc><content type="string"/></parameter>
</parameters></resource-agent></pacemaker-result>`
const clusterConfigCIBFixture = `<cib><configuration>
<crm_config><cluster_property_set id="cib-bootstrap-options"><nvpair id="name-pair" name="cluster-name" value="test"/><nvpair id="maintenance-pair" name="maintenance-mode" value="false"/><nvpair id="version-pair" name="dc-version" value="3"/><nvpair id="unknown-pair" name="custom" value="retain"/><nvpair id="advanced-pair" name="advanced-option" value="retain"/></cluster_property_set><cluster_property_set id="other"><nvpair name="cluster-name" value="other"/></cluster_property_set></crm_config>
<rsc_defaults><meta_attributes id="rsc-options"><nvpair id="priority-pair" name="priority" value="2"/></meta_attributes><meta_attributes id="other-defaults"><nvpair name="resource-stickiness" value="99"/></meta_attributes></rsc_defaults>
<op_defaults><meta_attributes id="op-options"><nvpair id="timeout-pair" name="timeout" value="600"/></meta_attributes></op_defaults>
</configuration></cib>`

func clusterConfigTestRunner(_ context.Context, name string, args []string, input string) ([]byte, error) {
	if name == "crm_attribute" && len(args) > 0 && args[0] == "--list-options=cluster" {
		return []byte(clusterConfigMetadataFixture), nil
	}
	if name == "cibadmin" {
		return []byte(clusterConfigCIBFixture), nil
	}
	return nil, fmt.Errorf("unexpected command: %s %v", name, args)
}
func TestClusterConfigMetadataAndLoad(t *testing.T) {
	state, err := loadClusterConfig(context.Background(), clusterConfigTestRunner)
	require.NoError(t, err)
	require.Len(t, state.Options["crm_config"], 5)
	require.Equal(t, "cluster-name", state.Options["crm_config"][0].Name)
	require.Equal(t, "dc-version", state.Options["crm_config"][2].Name)
	options := map[string]SelectOption{}
	for _, o := range state.Options["crm_config"] {
		options[o.Name] = o
	}
	require.Equal(t, "name-pair", options["cluster-name"].CibID)
	require.Equal(t, "test", options["cluster-name"].CibValue)
	require.Equal(t, "false", options["maintenance-mode"].DefaultValue)
	require.Equal(t, []string{"stop", "ignore"}, options["no-quorum-policy"].PossibleValues)
	require.Equal(t, "2", state.Current["rsc_defaults"]["priority"].Value)
	require.NotContains(t, state.Current["rsc_defaults"], "resource-stickiness")
	require.Equal(t, "600", state.Current["op_defaults"]["timeout"].Value)
	_, err = clusterPropertyOptions([]byte(`<broken`))
	require.Error(t, err)
}
func clusterConfigValues() map[string]map[string]string {
	return map[string]map[string]string{"crm_config": {"cluster-name": "test", "maintenance-mode": "false"}, "rsc_defaults": {"priority": "2"}, "op_defaults": {"timeout": "600"}}
}
func TestClusterConfigMutations(t *testing.T) {
	state, err := loadClusterConfig(context.Background(), clusterConfigTestRunner)
	require.NoError(t, err)
	values := clusterConfigValues()
	mutations, err := clusterConfigMutations(state, values)
	require.NoError(t, err)
	require.Empty(t, mutations)
	values["crm_config"]["cluster-name"] = "quoted '\" $HOME\nvalue"
	values["rsc_defaults"] = map[string]string{}
	values["op_defaults"]["timeout"] = "900"
	mutations, err = clusterConfigMutations(state, values)
	require.NoError(t, err)
	require.Len(t, mutations, 3)
	require.Equal(t, []string{"--type", "crm_config", "--set-name", "cib-bootstrap-options", "--name", "cluster-name", "--update", "quoted '\" $HOME\nvalue"}, mutations[0].Args)
	require.Equal(t, []string{"--type", "rsc_defaults", "--set-name", "rsc-options", "--name", "priority", "--delete"}, mutations[1].Args)
	for _, name := range []string{"custom", "unknown"} {
		values = clusterConfigValues()
		values["crm_config"][name] = "bad"
		_, err = clusterConfigMutations(state, values)
		require.Error(t, err)
	}
	values = clusterConfigValues()
	values["crm_config"]["dc-version"] = "ignored change"
	values["crm_config"]["advanced-option"] = "ignored change"
	mutations, err = clusterConfigMutations(state, values)
	require.NoError(t, err)
	require.Empty(t, mutations)
	values = clusterConfigValues()
	values["crm_config"]["maintenance-mode"] = "bad"
	_, err = clusterConfigMutations(state, values)
	require.Error(t, err)
	values = clusterConfigValues()
	delete(values, "op_defaults")
	_, err = clusterConfigMutations(state, values)
	require.Error(t, err)
}
func TestClusterConfigAPI(t *testing.T) {
	for _, failure := range []bool{false, true} {
		mutations := 0
		run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
			if name == "crm_attribute" && len(args) > 0 && args[0] == "--type" {
				mutations++
				if failure {
					return nil, fmt.Errorf("Permission denied")
				}
				return nil, nil
			}
			return clusterConfigTestRunner(ctx, name, args, input)
		}
		response := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"section":"crm_config"}`))
		clusterConfigAPI(response, req, run, false)
		require.Equal(t, 200, response.Code)
		require.Contains(t, response.Body.String(), `"CibValue":"test"`)
		require.Zero(t, mutations)
		values := clusterConfigValues()
		values["crm_config"]["maintenance-mode"] = "true"
		payload, _ := json.Marshal(map[string]any{"values": values})
		response = httptest.NewRecorder()
		clusterConfigAPI(response, httptest.NewRequest("POST", "/", strings.NewReader(string(payload))), run, true)
		expected := 200
		if failure {
			expected = 422
		}
		require.Equal(t, expected, response.Code)
		require.Equal(t, 1, mutations)
		if !failure {
			require.Contains(t, response.Body.String(), "Configuration updated successfully")
		}
	}
	response := httptest.NewRecorder()
	clusterConfigAPI(response, httptest.NewRequest("POST", "/", strings.NewReader(`{"section":"arbitrary"}`)), clusterConfigTestRunner, false)
	require.Equal(t, 400, response.Code)
	response = httptest.NewRecorder()
	clusterConfigAPI(response, httptest.NewRequest("GET", "/", nil), clusterConfigTestRunner, true)
	require.Equal(t, 405, response.Code)
}
func TestClusterConfigPage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	response := httptest.NewRecorder()
	ClusterConfigHandler(response, httptest.NewRequest("GET", "/cib/live/crm_config/edit", nil))
	require.Equal(t, 200, response.Code)
	require.Equal(t, 3, strings.Count(response.Body.String(), "<x-kvgroup"))
	require.Equal(t, 3, strings.Count(response.Body.String(), "light-dom legacy-picker"))
	for _, value := range []string{"Resource Defaults", "Operation Defaults", `name="revert"`, `name="submit"`, "/static/js/classes/XAlert.js"} {
		require.Contains(t, response.Body.String(), value)
	}
}
