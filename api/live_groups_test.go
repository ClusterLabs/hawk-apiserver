package api

import (
	"encoding/json"
	"encoding/xml"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLiveStatusGroupParsing(t *testing.T) {
	var status CrmStatus
	require.NoError(t, xml.Unmarshal([]byte(`<crm_mon><resources><resource id="free" role="Started" active="true"/><group id="g" disabled="true" managed="true"><resource id="a" role="Stopped" target_role="Stopped" active="false"/><resource id="b" role="Stopped" active="false"/></group><clone id="c"><resource id="p:0" role="Started"/></clone></resources></crm_mon>`), &status))
	require.Len(t, status.Resources, 1)
	require.Len(t, status.Groups, 1)
	require.Len(t, status.Clones, 1)
	require.Equal(t, "g", status.Groups[0].ID)
	require.True(t, status.Groups[0].Disabled)
	require.True(t, status.Groups[0].Managed)
	require.Len(t, status.Groups[0].Resources, 2)
	require.Equal(t, "a", status.Groups[0].Resources[0].ID)
	require.Equal(t, "Stopped", status.Groups[0].Resources[0].TargetRole)
	data, err := json.Marshal(status)
	require.NoError(t, err)
	require.Contains(t, string(data), `"Groups":[`)
}
func TestLiveCIBIncludesGroupsAndMembers(t *testing.T) {
	var cib CIB
	require.NoError(t, xml.Unmarshal([]byte(`<cib><configuration><resources><primitive id="free"/><group id="g"><primitive id="a" class="ocf" provider="pacemaker" type="Dummy"/><primitive id="b" class="ocf" provider="pacemaker" type="Dummy"/><meta_attributes id="g-meta"><nvpair id="role" name="target-role" value="Stopped"/></meta_attributes></group><clone id="c"><primitive id="p"/></clone></resources><constraints><rsc_location id="loc-g" rsc="g" node="node1" score="100"/></constraints></configuration></cib>`), &cib))
	rows := cibResourceRows(cib)
	require.Len(t, rows, 5)
	byID := map[string]ResourceRow{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	require.Equal(t, "group", byID["g"].ObjectType)
	require.Equal(t, []string{"a", "b"}, byID["g"].Children)
	require.Equal(t, "Stopped", byID["g"].TargetRole)
	require.Equal(t, "Offline", byID["g"].Status)
	require.Len(t, byID["g"].Constraints.Locations, 1)
	require.Equal(t, "Dummy", byID["a"].Type)
	require.Equal(t, "primitive", byID["b"].ObjectType)
	require.Contains(t, byID, "free")
	require.Contains(t, byID, "p")
}
