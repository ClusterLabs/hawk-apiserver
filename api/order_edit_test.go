package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestOrderFromXML(t *testing.T) {
	order, err := orderFromXML([]byte(`<rsc_order id="o" kind="Mandatory" first="a" then="b" first-action="promote" then-action="start"/>`))
	require.NoError(t, err)
	require.Equal(t, "Mandatory", order.Score)
	require.True(t, order.Symmetrical)
	require.Equal(t, []colocationSet{{Sequential: true, Action: "promote", Resources: []string{"a"}}, {Sequential: true, Action: "start", Resources: []string{"b"}}}, order.Resources)
	order, err = orderFromXML([]byte(`<rsc_order id="o" score="-10" symmetrical="false"><resource_set id="s1" sequential="false" action="start"><resource_ref id="b"/><resource_ref id="a"/></resource_set><resource_set id="s2" action="stop"><resource_ref id="c"/></resource_set></rsc_order>`))
	require.NoError(t, err)
	require.False(t, order.Symmetrical)
	require.Equal(t, "-10", order.Score)
	require.Equal(t, []colocationSet{{Sequential: false, Action: "start", Resources: []string{"b", "a"}}, {Sequential: true, Action: "stop", Resources: []string{"c"}}}, order.Resources)
	for _, data := range []string{`<rsc_order id="o" first="a" then="b" first-instance="0"/>`, `<rsc_order id="o"><resource_set require-all="false"/></rsc_order>`, `<rsc_order id="o"><resource_set id-ref="shared"/></rsc_order>`} {
		_, err = orderFromXML([]byte(data))
		require.Error(t, err)
	}
}

const orderEditFixture = `<cib><configuration><resources><primitive id="a"/><primitive id="b"/><primitive id="c"/></resources><constraints><rsc_order id="o" kind="Mandatory" first="a" then="b" first-action="start" then-action="promote"/></constraints></configuration></cib>`

func orderEditReader(_ context.Context, name string, args []string, input string) ([]byte, error) {
	if name == "cibadmin" {
		return []byte(orderEditFixture), nil
	}
	return nil, fmt.Errorf("unexpected command")
}
func TestOrderEditAPI(t *testing.T) {
	state, found, err := readOrderEdit(context.Background(), "o", orderEditReader)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "Mandatory", state.Order.Score)
	require.Equal(t, "promote", state.Order.Resources[1].Action)
	require.ElementsMatch(t, []string{"a", "b", "c"}, state.Available)
	for _, tt := range []struct {
		name, id, resource string
		status             int
		fail               bool
	}{
		{name: "update", id: "o", resource: "b", status: 200}, {name: "missing", id: "missing", resource: "b", status: 404}, {name: "primitive", id: "a", resource: "b", status: 404}, {name: "stale", id: "o", resource: "gone", status: 409}, {name: "failed command", id: "o", resource: "b", status: 500, fail: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writes := 0
			run := func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
				if name == "cibadmin" {
					return orderEditReader(ctx, name, args, input)
				}
				writes++
				require.Equal(t, "crm", name)
				require.Equal(t, []string{"-F", "configure", "load", "update", "/dev/stdin"}, args)
				require.Equal(t, "'order' 'o' 'Optional:' 'b:promote' 'a:start' 'symmetrical=false'\n", input)
				if tt.fail {
					return nil, fmt.Errorf("Permission denied")
				}
				return nil, nil
			}
			body := fmt.Sprintf(`{"id":%q,"score":"Optional","symmetrical":false,"resources":[{"sequential":true,"action":"promote","resources":[%q]},{"sequential":true,"action":"start","resources":["a"]}]}`, tt.id, tt.resource)
			w := httptest.NewRecorder()
			updateOrderEdit(w, httptest.NewRequest("POST", "/", strings.NewReader(body)), run)
			require.Equal(t, tt.status, w.Code, w.Body.String())
			if tt.status == 200 || tt.fail {
				require.Equal(t, 1, writes)
			} else {
				require.Zero(t, writes)
			}
		})
	}
	w := httptest.NewRecorder()
	updateOrderEdit(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"id":"o","score":"Mandatory","resources":[]}`)), nil)
	require.Equal(t, 400, w.Code)
	w = httptest.NewRecorder()
	updateOrderEdit(w, httptest.NewRequest("GET", "/", nil), nil)
	require.Equal(t, 405, w.Code)
	// Ensure the CIB fixture is valid independently of the view parser.
	var doc CIB
	require.NoError(t, xml.Unmarshal([]byte(orderEditFixture), &doc))
}
func TestOrderEditPage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	w := httptest.NewRecorder()
	OrderEditHandler(w, httptest.NewRequest("GET", "/cib/live/orders/o/edit", nil))
	require.Equal(t, 200, w.Code)
	for _, text := range []string{"Edit Order", `data-order-id="o"`, `readonly value="o"`, `name="revert"`, `/cib/live/config/edit#constraints`, `RenamePopup.js`} {
		require.Contains(t, w.Body.String(), text)
	}
}
