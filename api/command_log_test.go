package api

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCommandLogCompatibility(t *testing.T) {
	store := commandLog{root: t.TempDir()}
	// Existing Ruby records can be read without migration.
	require.NoError(t, os.WriteFile(store.path(false), []byte("# 2026-11-24T12:00:00+01:00\ncrm resource stop dummy1@@COMMAND-END@@\n"), 0640))
	require.NoError(t, store.append(false, "crm configure\nprimitive dummy2 ocf:pacemaker:Dummy\n"))
	require.NoError(t, store.append(true, "crm resource start dummy3"))
	for _, tc := range []struct {
		path    string
		count   int
		content string
	}{
		{"/cib/live/commands.json", 2, "dummy2"}, {batchBase + "/commands.json", 1, "dummy3"},
	} {
		w := httptest.NewRecorder()
		serveCommandsJSON(w, httptest.NewRequest("GET", tc.path, nil), store)
		require.Equal(t, 200, w.Code)
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		var records []string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &records))
		require.Len(t, records, tc.count)
		require.Contains(t, records[len(records)-1], tc.content)
	}
	w := httptest.NewRecorder()
	serveCommandsJSON(w, httptest.NewRequest("HEAD", "/cib/live/commands.json", nil), store)
	require.Equal(t, 200, w.Code)
	require.Empty(t, w.Body.String())
	w = httptest.NewRecorder()
	serveCommandsJSON(w, httptest.NewRequest("POST", "/cib/live/commands.json", nil), store)
	require.Equal(t, 405, w.Code)
	require.Equal(t, "GET, HEAD", w.Header().Get("Allow"))
}

func TestCommandLogMissingAndUnreadable(t *testing.T) {
	store := commandLog{root: t.TempDir()}
	w := httptest.NewRecorder()
	serveCommandsJSON(w, httptest.NewRequest("GET", "/cib/live/commands.json", nil), store)
	require.Equal(t, "[]\n", w.Body.String())
	require.NoError(t, os.Mkdir(store.path(false), 0750))
	w = httptest.NewRecorder()
	serveCommandsJSON(w, httptest.NewRequest("GET", "/cib/live/commands.json", nil), store)
	require.Equal(t, 500, w.Code)
}

func TestCommandLogRotationAndConcurrentWrites(t *testing.T) {
	store := commandLog{root: t.TempDir()}
	for _, tc := range []struct {
		size int
		age  time.Duration
	}{{128*1024 + 1, 0}, {32*1024 + 1, 25 * time.Hour}} {
		require.NoError(t, os.WriteFile(store.path(false), []byte(strings.Repeat("x", tc.size)), 0640))
		stamp := time.Now().Add(-tc.age)
		require.NoError(t, os.Chtimes(store.path(false), stamp, stamp))
		require.NoError(t, store.append(false, "crm resource stop dummy1"))
		records, err := store.read(false)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.NotContains(t, records[0], "xxxxx")
	}
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.append(true, "crm configure\nprimitive example ocf:pacemaker:Dummy\n")
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	records, err := store.read(true)
	require.NoError(t, err)
	require.Len(t, records, 20)
	for _, record := range records {
		require.Equal(t, 1, strings.Count(record, "# "))
		require.Contains(t, record, "primitive example")
	}
}
