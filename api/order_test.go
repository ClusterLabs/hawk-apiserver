package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderCreateArgs(t *testing.T) {
	for _, tt := range []struct {
		name        string
		sets        []colocationSet
		symmetrical bool
		want        []string
	}{
		{"simple", []colocationSet{{true, "start", []string{"a"}}, {true, "start", []string{"b"}}}, true, []string{"a:start", "b:start"}},
		{"chain stays in display order", []colocationSet{{true, "start", []string{"a"}}, {true, "promote", []string{"b"}}, {true, "stop", []string{"c"}}}, true, []string{"a:start", "b:promote", "c:stop"}},
		{"linked set", []colocationSet{{false, "start", []string{"a", "b"}}, {true, "demote", []string{"c"}}}, true, []string{"(", "a:start", "b:start", ")", "c:demote"}},
		{"asymmetrical", []colocationSet{{true, "", []string{"a"}}, {true, "", []string{"b"}}}, false, []string{"a", "b", "symmetrical=false"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			order := orderCreate{ID: "order1", Score: " Mandatory ", Symmetrical: tt.symmetrical, Resources: tt.sets}
			before, _ := json.Marshal(order)
			args, err := orderCreateArgs(order)
			require.NoError(t, err)
			require.Equal(t, append([]string{"configure", "order", "order1", "Mandatory:"}, tt.want...), args)
			after, _ := json.Marshal(order)
			require.Equal(t, string(before), string(after))
		})
	}
}

func TestOrderValidation(t *testing.T) {
	valid := orderCreate{ID: "order1", Score: "Mandatory", Symmetrical: true, Resources: []colocationSet{{true, "start", []string{"a"}}, {true, "start", []string{"b"}}}}
	for _, score := range []string{"mandatory", "Optional", "Serialize", "inf", "-inf", "INFINITY", "-INFINITY", "0", "-100"} {
		order := valid
		order.Score = score
		_, err := orderCreateArgs(order)
		require.NoError(t, err)
	}
	for _, tt := range []struct {
		change  func(*orderCreate)
		message string
	}{
		{func(o *orderCreate) { o.Score = "" }, "Score is required"},
		{func(o *orderCreate) { o.Score = "advisory" }, "Invalid score value"},
		{func(o *orderCreate) { o.Resources = o.Resources[:1] }, "at least two separate resources"},
		{func(o *orderCreate) { o.ID = "bad id" }, "Invalid ID"},
		{func(o *orderCreate) {
			o.Resources = []colocationSet{{true, "Started", []string{"a"}}, {true, "start", []string{"b"}}}
		}, "Invalid action"},
	} {
		order := valid
		tt.change(&order)
		_, err := orderCreateArgs(order)
		require.ErrorContains(t, err, tt.message)
	}
}

func TestCreateOrder(t *testing.T) {
	order := orderCreate{ID: "order1", Score: "Mandatory", Symmetrical: true, Resources: []colocationSet{{true, "start", []string{"a"}}, {true, "start", []string{"b"}}}}
	payload, _ := json.Marshal(order)
	for _, tt := range []struct {
		name      string
		available []string
		runErr    error
		status    int
		calls     int
	}{
		{"created", []string{"a", "b"}, nil, 200, 1},
		{"stale resource", []string{"a"}, nil, 409, 0},
		{"crm error", []string{"a", "b"}, fmt.Errorf("ERROR: rejected"), 500, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			response := httptest.NewRecorder()
			createOrder(response, httptest.NewRequest("POST", "/", strings.NewReader(string(payload))), func() (CIB, int, error) { return CIB{ColocationResources: tt.available}, 0, nil }, func(_ context.Context, args ...string) error {
				calls++
				require.Equal(t, []string{"configure", "order", "order1", "Mandatory:", "a:start", "b:start"}, args)
				return tt.runErr
			})
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, tt.calls, calls)
			if tt.status == 200 {
				require.Contains(t, response.Body.String(), "/cib/live/orders/order1/edit?flash=created")
			}
		})
	}
	response := httptest.NewRecorder()
	createOrder(response, httptest.NewRequest("GET", "/", nil), nil, nil)
	require.Equal(t, 405, response.Code)
	response = httptest.NewRecorder()
	createOrder(response, httptest.NewRequest("POST", "/", strings.NewReader("{")), nil, nil)
	require.Equal(t, 400, response.Code)
}

func TestOrderTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/order_new.html")
	require.NoError(t, err)
	var body strings.Builder
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", nil))
	for _, value := range []string{`name="order[id]"`, `name="order[score]"`, `name="order[symmetrical]"`, `data-score="Mandatory"`, `data-score="Optional"`, `data-score="Serialize"`, `<legend>Resources</legend>`, `href="/cib/live/constraints/types"`} {
		require.Contains(t, body.String(), value)
	}
	require.NotContains(t, body.String(), "toggleable")
	response := httptest.NewRecorder()
	OrderNewHandler(response, httptest.NewRequest("POST", "/cib/live/orders/new", nil))
	require.Equal(t, 405, response.Code)
}

func TestBatchOrderCreate(t *testing.T) {
	backend, calls := batchTestBackend(t)
	original := backend.run
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		out, err := original(ctx, live, name, args, input)
		if name == "cibadmin" {
			return []byte(`<cib><configuration><resources><primitive id="r2"/><group id="g"><primitive id="nested"/></group><template id="t"/></resources></configuration></cib>`), nil
		}
		return out, err
	}
	response := batchTestRequest(backend, "POST", "order/options", `{}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	require.JSONEq(t, `["g","r2","t"]`, response.Body.String())
	payload := `{"id":"order1","score":"Mandatory","symmetrical":false,"resources":[{"sequential":true,"action":"promote","resources":["g"]},{"sequential":false,"action":"start","resources":["r2","t"]}]}`
	response = batchTestRequest(backend, "POST", "order/create", payload)
	require.Equal(t, 200, response.Code, response.Body.String())
	require.JSONEq(t, `{"status":"ok","redirect":"/cib/hacluster/orders/order1/edit?flash=created"}`, response.Body.String())
	last := (*calls)[len(*calls)-1]
	require.Equal(t, "crm", last.name)
	require.Equal(t, []string{"configure", "order", "order1", "Mandatory:", "g:promote", "(", "r2:start", "t:start", ")", "symmetrical=false"}, last.args)
	for _, call := range *calls {
		require.False(t, call.live)
	}
	*calls = nil
	response = batchTestRequest(backend, "POST", "order/create", strings.Replace(payload, `"r2"`, `"nested"`, 1))
	require.Equal(t, 409, response.Code, response.Body.String())
	for _, call := range *calls {
		require.NotEqual(t, "crm", call.name)
	}
	*calls = nil
	response = batchTestRequest(backend, "POST", "order/create", strings.Replace(payload, `"Mandatory"`, `"invalid"`, 1))
	require.Equal(t, 400, response.Code, response.Body.String())
	require.Empty(t, *calls)
}

func TestOrderNewCIBContext(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, base := range []string{"/cib/live", batchBase} {
		for _, suffix := range []string{"", "/"} {
			response := httptest.NewRecorder()
			OrderNewHandler(response, httptest.NewRequest("GET", base+"/orders/new"+suffix, nil))
			require.Equal(t, 200, response.Code)
			html := response.Body.String()
			require.Contains(t, html, `href="`+base+`/constraints/types"`)
			api := "/api/cib/order"
			options := api + "/options/fetch"
			if base == batchBase {
				api = batchAPI + "/order"
				options = api + "/options"
			}
			require.Contains(t, html, `data-api-base="`+api+`"`)
			require.Contains(t, html, `data-options-api="`+options+`"`)
			require.Equal(t, base == batchBase, strings.Contains(html, `/static/js/batch.js`))
			require.NotContains(t, html, `id="rename-order"`)
		}
	}
}
