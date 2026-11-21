package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const goSessionCookie = "hawk_go_session"
const loginCookie = "hawk_go_login"
const sessionLifetime = 10 * 24 * time.Hour

type passwordChecker func(context.Context, string, string) error
type authenticatedUserKey struct{}
type goSession struct {
	Username string    `json:"username"`
	Expires  time.Time `json:"expires"`
}

// Authentication owns its session files and cookies. It neither reads Rails
// cookies/secrets nor writes cluster attributes. Session files hold no passwords.
type Authentication struct {
	directory string
	check     passwordChecker
	now       func() time.Time
	mu        sync.Mutex
}

func NewAuthentication(directory string, check passwordChecker) *Authentication {
	return &Authentication{directory: directory, check: check, now: time.Now}
}

func AuthenticatedUser(r *http.Request) string {
	user, _ := r.Context().Value(authenticatedUserKey{}).(string)
	return user
}

func randomAuthToken() (string, error) {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token[:]), nil
}
func (a *Authentication) sessionPath(token string) string {
	sum := sha256.Sum256([]byte(token))
	return filepath.Join(a.directory, hex.EncodeToString(sum[:])+".json")
}
func validSessionToken(token string) bool {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(raw) == 32
}
func privateSessionDirectory(directory string) bool {
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}
func (a *Authentication) user(r *http.Request) string {
	cookie, err := r.Cookie(goSessionCookie)
	if err != nil || !validSessionToken(cookie.Value) {
		return ""
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !privateSessionDirectory(a.directory) {
		return ""
	}
	info, err := os.Lstat(a.sessionPath(cookie.Value))
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return ""
	}
	file, err := os.Open(a.sessionPath(cookie.Value))
	if err != nil {
		return ""
	}
	defer file.Close()
	var session goSession
	if json.NewDecoder(io.LimitReader(file, 4096)).Decode(&session) != nil || session.Username == "" {
		return ""
	}
	if !session.Expires.After(a.now()) {
		_ = os.Remove(a.sessionPath(cookie.Value))
		return ""
	}
	return session.Username
}
func (a *Authentication) forget(r *http.Request) {
	if cookie, err := r.Cookie(goSessionCookie); err == nil && validSessionToken(cookie.Value) {
		a.mu.Lock()
		defer a.mu.Unlock()
		_ = os.Remove(a.sessionPath(cookie.Value))
	}
}
func (a *Authentication) create(w http.ResponseWriter, r *http.Request, username string) error {
	token, err := randomAuthToken()
	if err != nil {
		return err
	}
	session := goSession{Username: username, Expires: a.now().Add(sessionLifetime)}
	a.mu.Lock()
	defer a.mu.Unlock()
	if err = os.MkdirAll(a.directory, 0700); err != nil {
		return err
	}
	if !privateSessionDirectory(a.directory) {
		return fmt.Errorf("Go session directory must be owned by the service account and private (mode 0700)")
	}
	// Remove expired Go sessions; leave all other application state untouched.
	if entries, e := os.ReadDir(a.directory); e == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			path := filepath.Join(a.directory, entry.Name())
			file, e := os.Open(path)
			if e != nil {
				continue
			}
			data, e := io.ReadAll(io.LimitReader(file, 4096))
			_ = file.Close()
			var old goSession
			if e == nil && json.Unmarshal(data, &old) == nil && !old.Expires.After(a.now()) {
				_ = os.Remove(path)
			}
		}
	}
	file, err := os.OpenFile(a.sessionPath(token), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = json.NewEncoder(file).Encode(session)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(a.sessionPath(token))
		return err
	}
	if cookie, e := r.Cookie(goSessionCookie); e == nil && validSessionToken(cookie.Value) {
		_ = os.Remove(a.sessionPath(cookie.Value))
	}
	sameSite := http.SameSiteLaxMode
	// Dashboard clients log into remote clusters via this explicit JSON endpoint.
	if r.URL.Path == "/login.json" && r.TLS != nil {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{Name: goSessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: r.TLS != nil, SameSite: sameSite, MaxAge: int(sessionLifetime.Seconds()), Expires: session.Expires})
	return nil
}

func authJSONRequest(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/") || strings.HasSuffix(r.URL.Path, ".json") || strings.Contains(r.Header.Get("Accept"), "application/json")
}
func authJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func localAuthRequest(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		parsed, err := url.Parse(origin)
		return err == nil && parsed.Host == r.Host && ((r.TLS != nil && parsed.Scheme == "https") || (r.TLS == nil && parsed.Scheme == "http"))
	}
	return r.Header.Get("Sec-Fetch-Site") != "cross-site"
}
func (a *Authentication) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		username := a.user(r)
		if username == "" {
			if authJSONRequest(r) {
				authJSON(w, 401, map[string]string{"error": "Authentication required"})
			} else {
				http.Redirect(w, r, "/login?return_to="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
			}
			return
		}
		// Sessions created for cross-cluster dashboard requests can use SameSite=None.
		// Allow its read-only POST endpoint; reject cross-origin configuration writes.
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" && r.URL.Path != "/api/cib/cluster/dashboard/fetch" && !localAuthRequest(r) {
			http.Error(w, "Cross-origin configuration changes are not allowed", 403)
			return
		}
		if r.URL.Query().Get("init_cib") == "true" && !localAuthRequest(r) {
			http.Error(w, "Cross-origin configuration changes are not allowed", 403)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), authenticatedUserKey{}, username)))
	}
}
func authReturnTo(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, "\\\r\n") || strings.HasPrefix(parsed.Path, "/login") || strings.HasPrefix(parsed.Path, "/logout") {
		return "/cib/live"
	}
	return parsed.RequestURI()
}
func (a *Authentication) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	jsonRequest := r.URL.Path == "/login.json" || authJSONRequest(r)
	if r.Method != "GET" && r.Method != "HEAD" && r.Method != "POST" {
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	if r.Method != "POST" {
		if jsonRequest {
			authJSON(w, 200, map[string]any{})
			return
		}
		if a.user(r) != "" {
			http.Redirect(w, r, authReturnTo(r.URL.Query().Get("return_to")), 303)
			return
		}
		a.loginPage(w, r, 200, "", "", r.URL.Query().Get("return_to"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	var credentials struct {
		Session struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"session"`
	}
	var username, password, next, csrf string
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&credentials); err != nil {
			authJSON(w, 400, map[string]string{"error": "Invalid login request"})
			return
		}
		username, password = credentials.Session.Username, credentials.Session.Password
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid login request", 400)
			return
		}
		username, password = r.PostForm.Get("session[username]"), r.PostForm.Get("session[password]")
		next, csrf = r.PostForm.Get("return_to"), r.PostForm.Get("authenticity_token")
	}
	if jsonRequest && !localAuthRequest(r) && !strings.Contains(r.Header.Get("Accept"), "application/json") && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Cross-origin login requires an explicit JSON request", 403)
		return
	}
	if !jsonRequest {
		cookie, err := r.Cookie(loginCookie)
		if !localAuthRequest(r) || err != nil || csrf == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(csrf)) != 1 {
			http.Error(w, "Invalid login request; reload the login page", 403)
			return
		}
	}
	// The existing hawk_chkpwd helper has a 64-byte password buffer. Reject
	// longer input rather than silently authenticating a truncated password.
	if strings.TrimSpace(username) == "" || len(username) > 256 || strings.ContainsAny(username, "\x00\r\n") || password == "" || strings.ContainsRune(password, 0) || len(password) > 63 {
		a.loginFailure(w, r, jsonRequest, username, next)
		return
	}
	if err := a.check(r.Context(), username, password); err != nil {
		a.loginFailure(w, r, jsonRequest, username, next)
		return
	}
	if err := a.create(w, r, username); err != nil {
		log.Printf("[auth] Unable to create Go session: %v", err)
		if jsonRequest {
			authJSON(w, 503, map[string]string{"error": "Unable to create session"})
		} else {
			a.loginPage(w, r, 503, "Unable to create session", username, next)
		}
		return
	}
	http.SetCookie(w, &http.Cookie{Name: loginCookie, Path: "/", MaxAge: -1, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})
	if jsonRequest {
		authJSON(w, 200, map[string]any{})
		return
	}
	http.Redirect(w, r, authReturnTo(next), 303)
}
func (a *Authentication) loginFailure(w http.ResponseWriter, r *http.Request, jsonRequest bool, username, next string) {
	if jsonRequest {
		authJSON(w, 403, map[string]any{"errors": []string{"Invalid username or password"}})
		return
	}
	a.loginPage(w, r, 403, "Invalid username or password", username, next)
}
func (a *Authentication) loginPage(w http.ResponseWriter, r *http.Request, status int, message, username, next string) {
	token, err := randomAuthToken()
	if err != nil {
		http.Error(w, "Unable to create login form", 500)
		return
	}
	tmpl, err := template.ParseFiles("templates/login.html")
	if err != nil {
		http.Error(w, "Unable to load login form", 500)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: loginCookie, Value: token, Path: "/", MaxAge: 600, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.Execute(w, map[string]string{"Error": message, "Username": username, "Token": token, "ReturnTo": authReturnTo(next), "Year": a.now().Format("2006")}); err != nil {
		log.Printf("[auth] Login template: %v", err)
	}
}
func (a *Authentication) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	a.forget(r)
	http.SetCookie(w, &http.Cookie{Name: goSessionCookie, Path: "/", MaxAge: -1, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})
	w.Header().Set("Cache-Control", "no-store")
	if authJSONRequest(r) {
		authJSON(w, 200, map[string]any{})
		return
	}
	http.Redirect(w, r, "/login", 303)
}
