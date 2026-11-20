package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ClusterLabs/hawk-apiserver/api"
	"github.com/ClusterLabs/hawk-apiserver/internal"
	"github.com/stretchr/testify/require"
)

func TestNativeFallbackHasNoRailsRoutes(t *testing.T) {
	// This proxy is deliberately unreachable. None of these requests may use it.
	target := "unix:///nonexistent/rails.sock"
	config := internal.Config{Route: []internal.ConfigRoute{{Handler: "proxy", Path: "/", Target: &target}}}
	monitorCalls := 0
	fallback := nativeFallback(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { monitorCalls++; w.WriteHeader(204) }))
	require.NotNil(t, internal.NewRouteHandler(&config))
	for _, path := range []string{"/internal/auth", "/cib/live/resources.json", "/cib/hacluster/commands.json", "/assets/legacy.css", "/api/v1/status", "/unknown"} {
		response := httptest.NewRecorder()
		fallback.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		require.Equal(t, 404, response.Code, path)
	}
	require.Zero(t, monitorCalls)
	for _, path := range []string{"/monitor", "/monitor.json"} {
		response := httptest.NewRecorder()
		fallback.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		require.Equal(t, 204, response.Code)
	}
	require.Equal(t, 2, monitorCalls)
	response := httptest.NewRecorder()
	fallback.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
	require.Equal(t, "/cib/live", response.Header().Get("Location"))
}

func TestNativeAuthenticationDoesNotNeedRails(t *testing.T) {
	old := authentication
	t.Cleanup(func() { authentication = old })
	authentication = api.NewAuthentication(filepath.Join(t.TempDir(), "sessions"), func(context.Context, string, string) error { return nil })
	request := httptest.NewRequest("POST", "https://hawk.test/login.json", strings.NewReader(`{"session":{"username":"hacluster","password":"test"}}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	authentication.Login(response, request)
	require.Equal(t, 200, response.Code, response.Body.String())
	var session *http.Cookie
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "hawk_go_session" {
			session = cookie
		}
	}
	require.NotNil(t, session)
	calls := 0
	handler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "hacluster", api.AuthenticatedUser(r))
		w.WriteHeader(204)
	})
	for _, path := range []string{"/cib/live", "/cib/hacluster", "/api/batch/hacluster/state"} {
		request = httptest.NewRequest("GET", "https://hawk.test"+path, nil)
		request.AddCookie(session)
		response = httptest.NewRecorder()
		handler(response, request)
		require.Equal(t, 204, response.Code)
	}
	require.Equal(t, 3, calls)
}
