package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const tagEditFixture = `<cib><configuration><resources><primitive id="r1"/><primitive id="r2"/></resources><constraints><rsc_location id="loc1"/></constraints><tags><tag id="database" description="Database &amp; app"><obj_ref id="r1"/></tag><tag id="other"><obj_ref id="r2"/></tag></tags></configuration></cib>`

func TestFetchTag(t *testing.T) {
	run := func(_ context.Context, name string, args []string, input string) ([]byte, error) {
		require.Equal(t, "cibadmin", name)
		require.Equal(t, []string{"-Ql"}, args)
		return []byte(tagEditFixture), nil
	}
	for _, id := range []string{"database", "missing", "r1"} {
		w := httptest.NewRecorder()
		fetchTag(w, httptest.NewRequest("GET", "/?id="+id, nil), run)
		if id != "database" {
			require.Equal(t, 404, w.Code)
			continue
		}
		require.Equal(t, 200, w.Code)
		var data struct {
			ID            string
			Refs, Objects []string
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &data))
		require.Equal(t, "database", data.ID)
		require.Equal(t, []string{"r1"}, data.Refs)
		require.ElementsMatch(t, []string{"r1", "r2", "loc1"}, data.Objects)
	}
}

func TestUpdateTag(t *testing.T) {
	for _, tt := range []struct {
		name, payload  string
		status, writes int
		failure        bool
	}{
		{name: "replace", payload: `{"id":"database","refs":["r2","loc1"]}`, status: 200, writes: 1},
		{name: "command error", payload: `{"id":"database","refs":["r2","loc1"]}`, status: 500, writes: 1, failure: true},
		{name: "empty", payload: `{"id":"database","refs":[]}`, status: 400},
		{name: "invalid reference", payload: `{"id":"database","refs":["bad.ref"]}`, status: 400},
		{name: "removed reference", payload: `{"id":"database","refs":["gone"]}`, status: 409},
		{name: "missing", payload: `{"id":"missing","refs":["r1"]}`, status: 404},
		{name: "primitive", payload: `{"id":"r1","refs":["r2"]}`, status: 404},
		{name: "invalid json", payload: `{`, status: 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writes := 0
			run := func(_ context.Context, name string, args []string, input string) ([]byte, error) {
				require.Equal(t, "cibadmin", name)
				if len(args) == 1 {
					require.Equal(t, []string{"-Ql"}, args)
					return []byte(tagEditFixture), nil
				}
				writes++
				require.Equal(t, []string{"-p", "--replace"}, args)
				var tag tagElement
				require.NoError(t, xml.Unmarshal([]byte(input), &tag))
				require.Equal(t, "database", tag.id())
				require.Equal(t, []tagReference{{ID: "r2"}, {ID: "loc1"}}, tag.Refs)
				require.Contains(t, input, `description="Database &amp; app"`)
				require.NotContains(t, input, `id="other"`)
				if tt.failure {
					return nil, fmt.Errorf("Permission denied")
				}
				return nil, nil
			}
			w := httptest.NewRecorder()
			updateTag(w, httptest.NewRequest("POST", "/", strings.NewReader(tt.payload)), run)
			require.Equal(t, tt.status, w.Code, w.Body.String())
			require.Equal(t, tt.writes, writes)
			if tt.status == 200 {
				require.Contains(t, w.Body.String(), "Tag updated successfully")
			}
		})
	}
	w := httptest.NewRecorder()
	updateTag(w, httptest.NewRequest("GET", "/", nil), nil)
	require.Equal(t, 405, w.Code)
}

func TestTagEditPage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	w := httptest.NewRecorder()
	TagEditHandler(w, httptest.NewRequest("GET", "/cib/live/tags/database/edit", nil))
	require.Equal(t, 200, w.Code)
	for _, text := range []string{"Edit Tag", `data-tag-id="database"`, `readonly value="database"`, `name="revert"`, `href="/cib/live/config/edit#tags"`, `RenamePopup.js`, `DeletePopup.js`} {
		require.Contains(t, w.Body.String(), text)
	}
	w = httptest.NewRecorder()
	TagEditHandler(w, httptest.NewRequest("POST", "/cib/live/tags/database/edit", nil))
	require.Equal(t, 405, w.Code)
}
