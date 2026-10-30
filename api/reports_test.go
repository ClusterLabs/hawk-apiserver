package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"html/template"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func reportRequest(t *testing.T, s *ReportsAPI, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "/api/reports"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestReportsStorage(t *testing.T) {
	s := NewReportsAPI(t.TempDir())
	w := reportRequest(t, s, "GET", "", "")
	require.Equal(t, 200, w.Code)
	require.JSONEq(t, `[]`, w.Body.String())
	require.NoError(t, os.WriteFile(filepath.Join(s.root, "reports", "hawk-2026-10-29T10:00:00Z-2026-10-29T11:00:00Z.tar.gz"), []byte("archive"), 0644))
	require.NoError(t, os.Symlink("/etc/passwd", filepath.Join(s.root, "reports", "link.gz")))
	w = reportRequest(t, s, "GET", "", "")
	var reports []historyReport
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &reports))
	require.Len(t, reports, 1)
	require.Equal(t, "2026-10-29T10:00:00Z", reports[0].FromTime.Format(time.RFC3339))
	require.Len(t, reports[0].ID, 9)
	w = reportRequest(t, s, "GET", "/"+reports[0].ID+"/download", "")
	require.Equal(t, 200, w.Code)
	require.Equal(t, "archive", w.Body.String())
	require.Equal(t, 200, reportRequest(t, s, "DELETE", "/"+reports[0].ID, "").Code)
	require.Equal(t, 404, reportRequest(t, s, "GET", "/"+reports[0].ID+"/download", "").Code)
	require.Equal(t, 404, reportRequest(t, s, "DELETE", "/"+reports[0].ID, "").Code)
}
func TestReportsErrors(t *testing.T) {
	s := NewReportsAPI(t.TempDir())
	for _, tc := range []struct {
		method, path, body string
		code               int
	}{
		{"POST", "/generate", `{"from_time":"bad","to_time":"bad"}`, 422},
		{"POST", "/generate", `{"from_time":"2026-10-29T11:00:00Z","to_time":"2026-10-29T10:00:00Z"}`, 422},
		{"POST", "/generate", `{`, 400}, {"POST", "/running", "", 405}, {"GET", "/unknown", "", 404},
	} {
		w := reportRequest(t, s, tc.method, tc.path, tc.body)
		require.Equal(t, tc.code, w.Code, w.Body.String())
	}
	r := httptest.NewRequest("POST", "/api/reports/cancel", nil)
	r.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	require.Equal(t, 403, w.Code)
	require.NoError(t, os.WriteFile(s.statusPath("pid"), []byte("bad"), 0644))
	require.Equal(t, 500, reportRequest(t, s, "GET", "/running", "").Code)
	blocked := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocked, nil, 0644))
	require.Equal(t, 500, reportRequest(t, NewReportsAPI(blocked), "GET", "", "").Code)
}
func TestReportsUnavailableTools(t *testing.T) {
	s := NewReportsAPI(t.TempDir())
	s.crmPath = filepath.Join(t.TempDir(), "missing")
	w := reportRequest(t, s, "GET", "/capabilities", "")
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"can_generate":false`)
	w = reportRequest(t, s, "POST", "/generate", `{"from_time":"2026-10-29T10:00:00Z","to_time":"2026-10-29T11:00:00Z"}`)
	require.Equal(t, 503, w.Code)
}
func uploadReport(t *testing.T, s *ReportsAPI, name string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("report[upload]", name)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	r := httptest.NewRequest("POST", "/api/reports/upload", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestReportsUpload(t *testing.T) {
	s := NewReportsAPI(t.TempDir())
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	archive := tar.NewWriter(gz)
	require.NoError(t, archive.WriteHeader(&tar.Header{Name: "report/description.txt", Mode: 0644, Size: 4}))
	_, err := io.WriteString(archive, "test")
	require.NoError(t, err)
	require.NoError(t, archive.Close())
	require.NoError(t, gz.Close())
	require.Equal(t, 422, uploadReport(t, s, "report.zip", compressed.Bytes()).Code)
	require.Equal(t, 422, uploadReport(t, s, "report.tar.gz", []byte("broken")).Code)
	w := uploadReport(t, s, "report.tar.gz", compressed.Bytes())
	require.Equal(t, 201, w.Code, w.Body.String())
	require.Equal(t, 409, uploadReport(t, s, "report.tar.gz", compressed.Bytes()).Code)
	require.Equal(t, 422, uploadReport(t, s, "report.tgz", compressed.Bytes()).Code)

}
func TestReportsGenerationAndCancellation(t *testing.T) {
	s := NewReportsAPI(t.TempDir())
	s.crmPath = filepath.Join(t.TempDir(), "crm")
	require.NoError(t, os.WriteFile(s.crmPath, []byte("#!/bin/sh\nsleep 30\n"), 0755))
	const input = `{"from_time":"2026-10-29T10:00:00Z","to_time":"2026-10-29T11:00:00Z"}`
	require.Equal(t, 202, reportRequest(t, s, "POST", "/generate", input).Code)
	t.Cleanup(func() {
		s.mu.Lock()
		if s.cancel != nil {
			s.cancel()
		}
		s.mu.Unlock()
	})
	require.Equal(t, 409, reportRequest(t, s, "POST", "/generate", input).Code)
	require.Contains(t, reportRequest(t, s, "GET", "/running", "").Body.String(), `"running":true`)
	require.Equal(t, 202, reportRequest(t, s, "POST", "/cancel", "").Code)
	require.Eventually(t, func() bool {
		return strings.Contains(reportRequest(t, s, "GET", "/running", "").Body.String(), `"running":false`)
	}, 3*time.Second, 10*time.Millisecond)
	result, err := os.ReadFile(s.statusPath("exit"))
	require.NoError(t, err)
	require.NotEqual(t, "0", string(result))
	// A second click after collection stops should also succeed.
	w := reportRequest(t, s, "POST", "/cancel", "")
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), `"success":true`)
}

func TestReportsGenerationResult(t *testing.T) {
	for _, code := range []string{"0", "7"} {
		t.Run(code, func(t *testing.T) {
			s := NewReportsAPI(t.TempDir())
			s.crmPath = filepath.Join(t.TempDir(), "crm")
			require.NoError(t, os.WriteFile(s.crmPath, []byte("#!/bin/sh\nexit "+code+"\n"), 0755))
			w := reportRequest(t, s, "POST", "/generate", `{"from_time":"2026-10-29T10:00:00Z","to_time":"2026-10-29T11:00:00Z"}`)
			require.Equal(t, 202, w.Code, w.Body.String())
			require.Eventually(t, func() bool {
				return strings.Contains(reportRequest(t, s, "GET", "/running", "").Body.String(), `"running":false`)
			}, 3*time.Second, 10*time.Millisecond)
			result, err := os.ReadFile(s.statusPath("exit"))
			require.NoError(t, err)
			require.Equal(t, code, string(result))
			require.Contains(t, reportRequest(t, s, "GET", "/running", "").Body.String(), `"report_generated":`+map[bool]string{true: "true", false: "false"}[code == "0"])
		})
	}
}

func TestReportsTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/layout.html", "../templates/reports.html")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&body, "layout.html", map[string]any{"Title": "History Explorer"}))
	require.Contains(t, body.String(), "/static/js/reports.js")
	require.Contains(t, body.String(), "generate-report-form")
	require.Contains(t, body.String(), "upload-report-form")
}
