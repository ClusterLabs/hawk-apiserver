package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCommandsHandler(t *testing.T) {
	// Templates are resolved relative to the server's working directory.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	for _, path := range []string{"/cib/live/commands", "/cib/live/commands/", "/cib/hacluster/commands", "/cib/hacluster/commands/"} {
		response := httptest.NewRecorder()
		CommandsHandler(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `/static/js/commands.js`) {
			t.Fatalf("page %s: status %d, body %s", path, response.Code, response.Body.String())
		}
		base := "/cib/live"
		batchPage := strings.HasPrefix(path, batchBase+"/")
		if batchPage {
			base = batchBase
		}
		if !strings.Contains(response.Body.String(), `data-source="`+base+`/commands.json"`) {
			t.Fatalf("page %s uses the wrong command log endpoint", path)
		}
		if strings.Contains(response.Body.String(), `id="btn-show-batch"`) != batchPage {
			t.Fatalf("page %s has incorrect batch controls", path)
		}
		if batchPage && !strings.Contains(response.Body.String(), `/static/js/batch.js`) {
			t.Fatalf("page %s must reuse batch controls", path)
		}
		for _, marker := range []string{"<<<<<<<", "=======", ">>>>>>>"} {
			if strings.Contains(response.Body.String(), marker) {
				t.Fatalf("page %s contains an unresolved merge conflict: %s", path, marker)
			}
		}
	}

	response := httptest.NewRecorder()
	CommandsHandler(response, httptest.NewRequest(http.MethodPost, "/cib/live/commands", nil))
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST: status %d, Allow %q", response.Code, response.Header().Get("Allow"))
	}

	response = httptest.NewRecorder()
	CommandsHandler(response, httptest.NewRequest(http.MethodGet, "/cib/live/commands/unknown", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("unknown path: status %d", response.Code)
	}
}
