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

func TestColocationResources(t *testing.T) {
	ids, err := colocationResourcesFromXML([]byte(`<cib><configuration><resources><primitive id="r10"/><primitive id="r2"/><group id="g"><primitive id="nested"/></group><clone id="c"/><master id="m"/><bundle id="b"/><template id="t"/></resources></configuration></cib>`))
	require.NoError(t, err)
	require.Equal(t, []string{"b", "c", "g", "m", "r2", "r10", "t"}, ids)
}

func TestColocationCommandOrdering(t *testing.T) {
	tests := []struct {
		name string
		sets []colocationSet
		want []string
	}{
		{"simple", []colocationSet{{true, "Started", []string{"a"}}, {true, "Started", []string{"b"}}}, []string{"a:Started", "b:Started"}},
		{"different roles", []colocationSet{{true, "Master", []string{"a"}}, {true, "Started", []string{"b"}}}, []string{"a:Master", "b:Started"}},
		{"collapsed chain", []colocationSet{{true, "", []string{"a"}}, {true, "", []string{"b"}}, {true, "", []string{"c"}}}, []string{"c", "b", "a"}},
		{"mixed roles", []colocationSet{{true, "Master", []string{"a"}}, {true, "Started", []string{"b"}}, {true, "Started", []string{"c"}}}, []string{"a:Master", "c:Started", "b:Started"}},
		{"linked set", []colocationSet{{false, "Started", []string{"a", "b"}}, {true, "", []string{"c"}}}, []string{"(", "b:Started", "a:Started", ")", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := colocationCreate{ID: "colocation1", Score: " INFINITY ", Resources: tt.sets, NodeAttr: "rack"}
			before, _ := json.Marshal(c)
			args, err := colocationCreateArgs(c)
			require.NoError(t, err)
			require.Equal(t, append(append([]string{"configure", "colocation", "colocation1", "INFINITY:"}, tt.want...), "node-attribute=rack"), args)
			after, _ := json.Marshal(c)
			require.Equal(t, string(before), string(after), "command generation must not change the form model")
		})
	}
}

func TestColocationValidation(t *testing.T) {
	valid := colocationCreate{ID: "c", Score: "100", Resources: []colocationSet{{true, "", []string{"a"}}, {true, "", []string{"b"}}}}
	for _, score := range []string{"mandatory", "advisory", "inf", "-inf", "infinity", "-infinity", "0", "-100"} {
		c := valid
		c.Score = score
		_, err := colocationCreateArgs(c)
		require.NoError(t, err)
	}
	for _, tt := range []struct {
		change  func(*colocationCreate)
		message string
	}{
		{func(c *colocationCreate) { c.Score = "" }, "Score is required"},
		{func(c *colocationCreate) { c.Score = "oops" }, "Invalid score value"},
		{func(c *colocationCreate) { c.NodeAttr = "bad attribute" }, "Invalid node attribute"},
		{func(c *colocationCreate) { c.Resources = c.Resources[:1] }, "at least two separate resources"},
		{func(c *colocationCreate) { c.ID = "bad id" }, "Invalid ID"},
	} {
		c := valid
		tt.change(&c)
		_, err := colocationCreateArgs(c)
		require.ErrorContains(t, err, tt.message)
	}
}

func TestCreateColocation(t *testing.T) {
	c := colocationCreate{ID: "c", Score: "INFINITY", Resources: []colocationSet{{true, "Started", []string{"a"}}, {true, "Started", []string{"b"}}}}
	payload, _ := json.Marshal(c)
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
			createColocation(response, httptest.NewRequest("POST", "/", strings.NewReader(string(payload))), func() (CIB, int, error) { return CIB{ColocationResources: tt.available}, 0, nil }, func(_ context.Context, args ...string) error {
				calls++
				require.Equal(t, []string{"configure", "colocation", "c", "INFINITY:", "a:Started", "b:Started"}, args)
				return tt.runErr
			})
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, tt.calls, calls)
			if tt.status == 200 {
				require.Contains(t, response.Body.String(), "/cib/live/colocations/c/edit?flash=created")
			}
		})
	}
}

func TestColocationTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/colocation_new.html")
	require.NoError(t, err)
	var body strings.Builder
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", nil))
	for _, value := range []string{`name="colocation[id]"`, `name="colocation[score]"`, `name="colocation[node_attr]"`, `data-score="INFINITY"`, `data-score="-INFINITY"`, `href="/cib/live/constraints/types"`} {
		require.Contains(t, body.String(), value)
	}
	response := httptest.NewRecorder()
	ColocationNewHandler(response, httptest.NewRequest("POST", "/cib/live/colocations/new", nil))
	require.Equal(t, 405, response.Code)
}
