package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http/httptest"
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
