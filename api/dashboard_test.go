package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ClusterLabs/hawk-apiserver/internal"
)

type dashboardTestStore struct {
	clusters map[string]internal.DashboardCluster
	err      error
}

func (s *dashboardTestStore) Load() (map[string]internal.DashboardCluster, error) {
	return s.clusters, s.err
}
func (s *dashboardTestStore) Save(c internal.DashboardCluster) error {
	if s.err != nil {
		return s.err
	}
	s.clusters[c.Name] = c
	return nil
}
func (s *dashboardTestStore) Remove(name string) error {
	if s.err != nil {
		return s.err
	}
	delete(s.clusters, name)
	return nil
}

func TestDashboardClustersHandler(t *testing.T) {
	store := &dashboardTestStore{clusters: make(map[string]internal.DashboardCluster)}
	handler := NewDashboardClustersHandler(store)
	request := func(method, body, contentType string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/api/dashboard/clusters", strings.NewReader(body))
		r.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		handler(w, r)
		return w
	}
	for _, tc := range []struct {
		method, body, contentType string
		status                    int
	}{
		{"GET", "", "", 200},
		{"PUT", "", "", 405},
		{"POST", `{}`, "text/plain", 415},
		{"POST", `{`, "application/json", 400},
		{"POST", `{} {}`, "application/json", 400},
		{"POST", `{"name":"west","host":"node","port":"7630"}`, "application/json", 400},
		{"POST", `{}`, "application/json", 422},
		{"POST", `{"name":"west","host":"node","port":0}`, "application/json", 422},
		{"DELETE", `{}`, "application/json", 422},
	} {
		w := request(tc.method, tc.body, tc.contentType)
		if w.Code != tc.status {
			t.Fatalf("%s %s: %d %s", tc.method, tc.body, w.Code, w.Body.String())
		}
	}
	if len(store.clusters) != 0 {
		t.Fatal("invalid request mutated store")
	}
	w := request("POST", `{"name":"west","host":"node","https":false}`, "application/json")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if c := store.clusters["west"]; c.HTTPS || c.Port != 7630 || c.Interval != 30 {
		t.Fatalf("defaults: %+v", c)
	}
	w = request("GET", "", "")
	var clusters map[string]internal.DashboardCluster
	if err := json.Unmarshal(w.Body.Bytes(), &clusters); err != nil || clusters["west"].Host != "node" {
		t.Fatalf("list: %s", w.Body.String())
	}
	w = request("DELETE", `{"name":"west"}`, "application/json")
	if w.Code != http.StatusNoContent || len(store.clusters) != 0 {
		t.Fatalf("delete: %d", w.Code)
	}
	store.err = errors.New("private filesystem details")
	for _, method := range []string{"GET", "POST", "DELETE"} {
		w = request(method, `{"name":"west","host":"node"}`, "application/json")
		if w.Code != 500 || strings.Contains(w.Body.String(), store.err.Error()) {
			t.Fatalf("storage failure: %d %s", w.Code, w.Body.String())
		}
	}
}

func TestDashboardStatusCORS(t *testing.T) {
	called := false
	handler := DashboardStatusCORS(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusUnauthorized)
	})
	for _, method := range []string{http.MethodPost, http.MethodOptions} {
		called = false
		r := httptest.NewRequest(method, "/api/cib/cluster/dashboard/fetch", nil)
		r.Header.Set("Origin", "https://dashboard.example:7630")
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Header().Get("Access-Control-Allow-Origin") != "https://dashboard.example:7630" || w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Fatal("missing credentialed CORS headers")
		}
		if (method == http.MethodPost) != called {
			t.Fatal("preflight invoked authenticated handler")
		}
	}
	t.Setenv("HAWK_ACCESS_CONTROL_ALLOW_ORIGIN", "https://trusted.example")
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Origin", "https://other.example")
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://trusted.example" {
		t.Fatal("origin restriction ignored")
	}
}
