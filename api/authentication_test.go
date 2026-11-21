package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func authTestDirectory(t *testing.T) {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
}
func authTestLogin(t *testing.T, a *Authentication, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"session": map[string]string{"username": username, "password": password}})
	r := httptest.NewRequest("POST", "https://hawk.test/login.json", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.Login(w, r)
	return w
}
func authTestCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == goSessionCookie && c.MaxAge > 0 {
			return c
		}
	}
	t.Fatal("No Go session cookie")
	return nil
}
func TestGoSessionsAreIndependentAndPersistAcrossInstances(t *testing.T) {
	checks := 0
	a := NewAuthentication(filepath.Join(t.TempDir(), "sessions"), func(_ context.Context, u, p string) error {
		checks++
		if u != "alice" || p != "correct" {
			return fmt.Errorf("invalid")
		}
		return nil
	})
	w := authTestLogin(t, a, "alice", "correct")
	require.Equal(t, 200, w.Code)
	c := authTestCookie(t, w)
	require.True(t, c.HttpOnly)
	require.True(t, c.Secure)
	require.Equal(t, http.SameSiteNoneMode, c.SameSite)
	require.NotContains(t, w.Header().Get("Set-Cookie"), "hawk_remember_me")
	r := httptest.NewRequest("GET", "https://hawk.test/cib/live", nil)
	r.AddCookie(c)
	require.Equal(t, "alice", a.user(r))
	require.Equal(t, 1, checks)
	data, err := os.ReadFile(a.sessionPath(c.Value))
	require.NoError(t, err)
	require.NotContains(t, string(data), "correct")
	require.NotContains(t, string(data), c.Value)
	info, err := os.Stat(a.sessionPath(c.Value))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	restarted := NewAuthentication(a.directory, func(context.Context, string, string) error {
		t.Fatal("PAM rechecked during session verification")
		return nil
	})
	require.Equal(t, "alice", restarted.user(r))
	restarted.now = func() time.Time { return a.now().Add(sessionLifetime + time.Second) }
	require.Empty(t, restarted.user(r))
	_, err = os.Stat(a.sessionPath(c.Value))
	require.True(t, os.IsNotExist(err))
}
func TestGoLoginFailureAndLegacyCookies(t *testing.T) {
	a := NewAuthentication(filepath.Join(t.TempDir(), "sessions"), func(context.Context, string, string) error { return fmt.Errorf("invalid") })
	w := authTestLogin(t, a, "alice", "wrong")
	require.Equal(t, 403, w.Code)
	for _, c := range w.Result().Cookies() {
		require.NotEqual(t, goSessionCookie, c.Name)
	}
	r := httptest.NewRequest("GET", "/api/cib/resources/fetch", nil)
	for _, c := range []*http.Cookie{{Name: "hawk", Value: "rails-session"}, {Name: "hawk_remember_me_id", Value: "hacluster"}, {Name: "hawk_remember_me_key", Value: "legacy-token"}} {
		r.AddCookie(c)
	}
	called := false
	w = httptest.NewRecorder()
	a.Middleware(func(http.ResponseWriter, *http.Request) { called = true })(w, r)
	require.Equal(t, 401, w.Code)
	require.False(t, called)
	for _, value := range []string{"../secret", "invalid", strings.Repeat("a", 100)} {
		r = httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: goSessionCookie, Value: value})
		require.Empty(t, a.user(r))
	}
}
func TestGoSessionLogoutAndRotation(t *testing.T) {
	a := NewAuthentication(filepath.Join(t.TempDir(), "sessions"), func(context.Context, string, string) error { return nil })
	first := authTestCookie(t, authTestLogin(t, a, "alice", "test"))
	body := `{"session":{"username":"bob","password":"test"}}`
	r := httptest.NewRequest("POST", "https://hawk.test/login.json", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(first)
	w := httptest.NewRecorder()
	a.Login(w, r)
	require.Equal(t, 200, w.Code)
	second := authTestCookie(t, w)
	require.NotEqual(t, first.Value, second.Value)
	r = httptest.NewRequest("GET", "https://hawk.test/cib/live", nil)
	r.AddCookie(first)
	require.Empty(t, a.user(r))
	r = httptest.NewRequest("GET", "https://hawk.test/logout", nil)
	r.AddCookie(second)
	w = httptest.NewRecorder()
	a.Logout(w, r)
	require.Equal(t, 303, w.Code)
	require.Equal(t, "/login", w.Header().Get("Location"))
	require.Empty(t, a.user(r))
	require.Equal(t, -1, w.Result().Cookies()[0].MaxAge)
}
func TestGoHTMLLoginAndReturnURL(t *testing.T) {
	authTestDirectory(t)
	a := NewAuthentication(filepath.Join(t.TempDir(), "sessions"), func(context.Context, string, string) error { return nil })
	w := httptest.NewRecorder()
	a.Login(w, httptest.NewRequest("GET", "https://hawk.test/login?return_to=%2Fcib%2Fhacluster%2Froles", nil))
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `id="session_username"`)
	require.NotContains(t, w.Body.String(), "/assets/")
	cookie := w.Result().Cookies()[0]
	r := httptest.NewRequest("POST", "https://hawk.test/login", strings.NewReader(url.Values{"session[username]": {"alice"}, "session[password]": {"test"}, "authenticity_token": {cookie.Value}, "return_to": {"/cib/hacluster/roles"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	a.Login(w, r)
	require.Equal(t, 303, w.Code)
	require.Equal(t, "/cib/hacluster/roles", w.Header().Get("Location"))
	require.NotNil(t, authTestCookie(t, w))
	r = httptest.NewRequest("POST", "https://hawk.test/login", strings.NewReader("session%5Busername%5D=alice&session%5Bpassword%5D=test"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	a.Login(w, r)
	require.Equal(t, 403, w.Code)
	for _, target := range []string{"https://evil.test/", "//evil.test/", "/\\evil.test/", "/login", "/logout"} {
		require.Equal(t, "/cib/live", authReturnTo(target))
	}
}
func TestGoSessionOriginChecks(t *testing.T) {
	a := NewAuthentication(filepath.Join(t.TempDir(), "sessions"), func(context.Context, string, string) error { return nil })
	cookie := authTestCookie(t, authTestLogin(t, a, "alice", "test"))
	for _, path := range []string{"/api/cib/resource/delete", "/api/batch/hacluster/apply", "/api/cib/cluster/dashboard/fetch"} {
		r := httptest.NewRequest("POST", "https://hawk.test"+path, nil)
		r.AddCookie(cookie)
		r.Header.Set("Origin", "https://other.test")
		w := httptest.NewRecorder()
		a.Middleware(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })(w, r)
		expected := 403
		if path == "/api/cib/cluster/dashboard/fetch" {
			expected = 204
		}
		require.Equal(t, expected, w.Code, path)
	}
	r := httptest.NewRequest("POST", "https://hawk.test/api/batch/hacluster/apply", nil)
	r.AddCookie(cookie)
	r.Header.Set("Origin", "https://hawk.test")
	w := httptest.NewRecorder()
	a.Middleware(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })(w, r)
	require.Equal(t, 204, w.Code)
	r.TLS = &tls.ConnectionState{} // Origin must include the right scheme as well as host.
	r.Header.Set("Origin", "http://hawk.test")
	require.False(t, localAuthRequest(r))
}
func TestGoSessionStorageFailure(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0600))
	a := NewAuthentication(file, func(context.Context, string, string) error { return nil })
	w := authTestLogin(t, a, "alice", "test")
	require.Equal(t, 503, w.Code)
	for _, cookie := range w.Result().Cookies() {
		require.NotEqual(t, goSessionCookie, cookie.Name)
	}
}

func TestGoSessionRejectsPublicStorage(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0755))
	defer os.Chmod(dir, 0700)
	a := NewAuthentication(dir, func(context.Context, string, string) error { return nil })
	response := authTestLogin(t, a, "alice", "test")
	require.Equal(t, 503, response.Code)
}

func TestGoJSONLoginCrossOrigin(t *testing.T) {
	a := NewAuthentication(filepath.Join(t.TempDir(), "sessions"), func(context.Context, string, string) error { return nil })
	for _, accept := range []string{"text/html", "application/json"} {
		r := httptest.NewRequest("POST", "https://hawk.test/login.json", strings.NewReader(url.Values{"session[username]": {"alice"}, "session[password]": {"test"}}.Encode()))
		r.Header.Set("Origin", "https://dashboard.test")
		r.Header.Set("Accept", accept)
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		DashboardStatusCORS(a.Login)(w, r)
		if accept == "text/html" {
			require.Equal(t, 403, w.Code)
		} else {
			require.Equal(t, 200, w.Code)
			require.NotNil(t, authTestCookie(t, w))
		}
		require.Equal(t, "https://dashboard.test", w.Header().Get("Access-Control-Allow-Origin"))
	}
}
