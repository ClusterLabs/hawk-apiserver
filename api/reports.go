package api

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const defaultReportsRoot = "/var/lib/hawk/tmp"

var reportID = regexp.MustCompile(`^[a-f0-9]{9}$`)
var reportFilename = regexp.MustCompile(`^[a-zA-Z0-9_-]+\.tar\.(bz2|gz|xz)$`)
var reportLegacyDates = regexp.MustCompile(`\d{4}-\d{2}-\d{2}_\d{2}:\d{2}`)
var reportDates = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})`)

type historyReport struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	FromTime time.Time `json:"from_time"`
	ToTime   time.Time `json:"to_time"`
	archive  string
}

// ReportsAPI shares archives and collection status files with the Rails viewer.
// One instance must be shared by all report routes.
type ReportsAPI struct {
	root    string
	mu      sync.Mutex
	cancel  func()
	crmPath string
	cibPath string
}

func NewReportsAPI(root string) *ReportsAPI {
	if root == "" {
		root = defaultReportsRoot
	}
	return &ReportsAPI{root: root, crmPath: "/usr/sbin/crm", cibPath: "/usr/sbin/cibadmin"}
}

func ReportsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/reports" && r.URL.Path != "/reports/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	renderTemplate(w, "reports", map[string]any{"Title": "History Explorer"})
}

func reportsJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func reportsError(w http.ResponseWriter, status int, message string) {
	reportsJSON(w, status, map[string]string{"error": message})
}

func (s *ReportsAPI) directories() error {
	for _, name := range []string{"reports", "pids"} {
		if err := os.MkdirAll(filepath.Join(s.root, name), 0750); err != nil {
			return err
		}
	}
	return nil
}
func (s *ReportsAPI) statusPath(name string) string {
	return filepath.Join(s.root, "pids", "report."+name)
}
func (s *ReportsAPI) list() ([]historyReport, error) {
	entries, err := os.ReadDir(filepath.Join(s.root, "reports"))
	if err != nil {
		return nil, err
	}
	reports := []historyReport{}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".gz" && ext != ".bz2" && ext != ".xz" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		name := strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ext), ".tar")
		if name == "." || name == ".." || name == "" {
			continue
		}
		digest := sha1.Sum([]byte(name))
		report := historyReport{ID: hex.EncodeToString(digest[:])[:9], Name: name, FromTime: info.ModTime().UTC(), ToTime: info.ModTime().UTC(), archive: filepath.Join(s.root, "reports", entry.Name())}
		dates := reportDates.FindAllString(name, 2)
		if len(dates) == 2 {
			from, e1 := time.Parse(time.RFC3339, dates[0])
			to, e2 := time.Parse(time.RFC3339, dates[1])
			if e1 == nil && e2 == nil {
				report.FromTime = from
				report.ToTime = to
			}
		}
		if len(dates) != 2 {
			legacy := reportLegacyDates.FindAllString(name, 2)
			if len(legacy) == 2 {
				from, e1 := time.Parse("2006-01-02_15:04", legacy[0])
				to, e2 := time.Parse("2006-01-02_15:04", legacy[1])
				if e1 == nil && e2 == nil {
					report.FromTime = from
					report.ToTime = to
				}
			}
		}
		reports = append(reports, report)
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].ToTime.After(reports[j].ToTime) })
	return reports, nil
}

func (s *ReportsAPI) running() (int, error) {
	data, err := os.ReadFile(s.statusPath("pid"))
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid report PID file")
	}
	err = syscall.Kill(pid, 0)
	if errors.Is(err, syscall.ESRCH) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func (s *ReportsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// JSON mutations cannot be submitted by cross-origin HTML forms.
	origin, originErr := url.Parse(r.Header.Get("Origin"))
	if r.Method != http.MethodGet && (r.Header.Get("Sec-Fetch-Site") == "cross-site" || (r.Header.Get("Origin") != "" && (originErr != nil || origin.Host != r.Host))) {
		reportsError(w, http.StatusForbidden, "Cross-site request rejected")
		return
	}
	action := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/reports"), "/")
	expected := http.MethodGet
	switch action {
	case "/generate", "/cancel", "/upload":
		expected = http.MethodPost
	case "", "/running", "/capabilities":
	default:
		parts := strings.Split(strings.TrimPrefix(action, "/"), "/")
		if len(parts) == 1 && reportID.MatchString(parts[0]) {
			expected = http.MethodDelete
		} else if len(parts) != 2 || parts[1] != "download" || !reportID.MatchString(parts[0]) {
			reportsError(w, http.StatusNotFound, "Unknown report endpoint")
			return
		}
	}
	if r.Method != expected {
		w.Header().Set("Allow", expected)
		reportsError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.directories(); err != nil {
		reportsError(w, http.StatusInternalServerError, "Unable to access report storage")
		return
	}
	switch {
	case action == "/capabilities" && r.Method == http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, s.cibPath, "-Ql")
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		reason := ""
		if info, err := os.Stat(s.crmPath); err != nil || info.Mode()&0111 == 0 {
			reason = "Report generation requires an executable crm command."
		} else if err := cmd.Run(); err != nil {
			reason = "Live cluster is unavailable. You can upload or inspect saved reports."
		}
		reportsJSON(w, http.StatusOK, map[string]any{"can_generate": reason == "", "reason": reason})
	case action == "" && r.Method == http.MethodGet:
		reports, err := s.list()
		if err != nil {
			reportsError(w, http.StatusInternalServerError, "Unable to read reports")
			return
		}
		reportsJSON(w, http.StatusOK, reports)
	case action == "/running" && r.Method == http.MethodGet:
		pid, err := s.running()
		if err != nil {
			reportsError(w, http.StatusInternalServerError, "Unable to read generation status")
			return
		}
		times, timeErr := os.ReadFile(s.statusPath("time"))
		exit, exitErr := os.ReadFile(s.statusPath("exit"))
		if (timeErr != nil && !errors.Is(timeErr, os.ErrNotExist)) || (exitErr != nil && !errors.Is(exitErr, os.ErrNotExist)) {
			reportsError(w, http.StatusInternalServerError, "Unable to read collection result")
			return
		}
		message := ""
		if strings.TrimSpace(string(exit)) != "0" && len(exit) > 0 {
			message = "crm report exited with status " + strings.TrimSpace(string(exit)) + "; check the report .stderr file."
		}
		reportsJSON(w, http.StatusOK, map[string]any{"running": pid > 0 || s.cancel != nil, "time": strings.Split(string(times), ","), "report_generated": strings.TrimSpace(string(exit)) == "0", "msg": message})
	case action == "/generate" && r.Method == http.MethodPost:
		s.generate(w, r)
	case action == "/cancel" && r.Method == http.MethodPost:
		pid, err := s.running()
		if err != nil {
			reportsError(w, http.StatusInternalServerError, "Unable to read generation status")
			return
		}
		if pid == 0 && s.cancel == nil {
			// Collection may finish between the status poll and the Cancel click.
			reportsJSON(w, http.StatusOK, map[string]any{"success": true, "message": "Report collection has already stopped."})
			return
		}
		if s.cancel == nil {
			reportsError(w, http.StatusConflict, "Collection was started outside this server; cancel it from its original process")
			return
		}
		s.cancel()
		reportsJSON(w, http.StatusAccepted, map[string]any{"success": true, "message": "Cancelling report collection…"})
	case action == "/upload" && r.Method == http.MethodPost:
		s.upload(w, r)
	default:
		parts := strings.Split(strings.Trim(action, "/"), "/")
		if len(parts) == 1 && r.Method == http.MethodDelete {
			s.archive(w, r, parts[0], false)
			return
		}
		if len(parts) == 2 && parts[1] == "download" && r.Method == http.MethodGet {
			s.archive(w, r, parts[0], true)
			return
		}
		reportsError(w, http.StatusNotFound, "Unknown report endpoint or method")
	}
}

func (s *ReportsAPI) generate(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		reportsError(w, http.StatusUnsupportedMediaType, "Expected application/json")
		return
	}
	var input struct {
		From string `json:"from_time"`
		To   string `json:"to_time"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input); err != nil {
		reportsError(w, http.StatusBadRequest, "Invalid generation request")
		return
	}
	from, e1 := time.Parse(time.RFC3339, input.From)
	to, e2 := time.Parse(time.RFC3339, input.To)
	if e1 != nil || e2 != nil || !to.After(from) {
		reportsError(w, http.StatusUnprocessableEntity, "Select valid dates with Until after From")
		return
	}
	pid, err := s.running()
	if err != nil {
		reportsError(w, http.StatusInternalServerError, "Unable to read generation status")
		return
	}
	if pid > 0 || s.cancel != nil {
		reportsError(w, http.StatusConflict, "A report collection is already running")
		return
	}
	fromString, toString := from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339)
	name := "hawk-" + fromString + "-" + toString
	destination := filepath.Join(s.root, "reports", name)
	// Do not replace an archive already used by the report explorer.
	for _, ext := range []string{".tar.gz", ".tar.bz2", ".tar.xz", ".gz", ".bz2", ".xz"} {
		if _, err := os.Lstat(destination + ext); err == nil {
			reportsError(w, http.StatusConflict, "A report for this range already exists")
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			reportsError(w, http.StatusInternalServerError, "Unable to check existing reports")
			return
		}
	}
	out, err := os.Create(destination + ".stdout")
	if err != nil {
		reportsError(w, http.StatusInternalServerError, "Unable to create report output")
		return
	}
	stderr, err := os.Create(destination + ".stderr")
	if err != nil {
		out.Close()
		reportsError(w, http.StatusInternalServerError, "Unable to create report output")
		return
	}
	cmd := exec.Command(s.crmPath, "report", "-f", fromString, "-t", toString, "-Z", "-Q", destination)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = out
	cmd.Stderr = stderr
	if err := os.WriteFile(s.statusPath("time"), []byte(fromString+","+toString), 0640); err != nil {
		out.Close()
		stderr.Close()
		reportsError(w, http.StatusInternalServerError, "Unable to write collection state")
		return
	}
	if err := os.Remove(s.statusPath("exit")); err != nil && !errors.Is(err, os.ErrNotExist) {
		out.Close()
		stderr.Close()
		reportsError(w, http.StatusInternalServerError, "Unable to reset collection state")
		return
	}
	if err := cmd.Start(); err != nil {
		out.Close()
		stderr.Close()
		reportsError(w, http.StatusServiceUnavailable, "Unable to start crm report; check that crmsh is installed and executable")
		return
	}
	if err := os.WriteFile(s.statusPath("pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0640); err != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
		out.Close()
		stderr.Close()
		reportsError(w, http.StatusInternalServerError, "Unable to write collection PID")
		return
	}
	var cancelTimer *time.Timer
	s.cancel = func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		if cancelTimer == nil {
			cancelTimer = time.AfterFunc(5*time.Second, func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) })
		}
	}
	go func() {
		// Bound hung tools, including subprocesses that keep output files open.
		timer := time.AfterFunc(30*time.Minute, func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) })
		err := cmd.Wait()
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		timer.Stop()
		out.Close()
		stderr.Close()
		s.mu.Lock()
		defer s.mu.Unlock()
		if cancelTimer != nil {
			cancelTimer.Stop()
		}
		result := "0"
		if err != nil {
			result = "1"
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() > 0 {
				result = strconv.Itoa(exitErr.ExitCode())
			}
		}
		_ = os.WriteFile(s.statusPath("exit"), []byte(result), 0640)
		_ = os.Remove(s.statusPath("pid"))
		s.cancel = nil
	}()
	reportsJSON(w, http.StatusAccepted, map[string]any{"success": true, "message": "Please wait while report is generated…"})
}

func (s *ReportsAPI) upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		reportsError(w, http.StatusBadRequest, "Invalid upload")
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, header, err := r.FormFile("report[upload]")
	if err != nil {
		reportsError(w, http.StatusBadRequest, "Select a report archive")
		return
	}
	defer file.Close()
	if !reportFilename.MatchString(header.Filename) {
		reportsError(w, http.StatusUnprocessableEntity, "Use an alphanumeric filename ending in .tar.gz, .tar.bz2 or .tar.xz")
		return
	}
	target := filepath.Join(s.root, "reports", header.Filename)
	temp, err := os.CreateTemp(filepath.Join(s.root, "reports"), ".upload-*")
	if err != nil {
		reportsError(w, http.StatusInternalServerError, "Unable to store upload")
		return
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0644); err != nil {
		temp.Close()
		reportsError(w, http.StatusInternalServerError, "Unable to set archive permissions")
		return
	}
	_, err = io.Copy(temp, file)
	closeErr := temp.Close()
	if err != nil || closeErr != nil {
		reportsError(w, http.StatusInternalServerError, "Unable to store upload")
		return
	}
	// Check archive structure without extracting uploaded paths.
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	check := exec.CommandContext(ctx, "tar", "-tf", temp.Name())
	check.Stdout = io.Discard
	check.Stderr = io.Discard
	if err := check.Run(); err != nil {
		reportsError(w, http.StatusUnprocessableEntity, "Invalid report archive or tar is unavailable")
		return
	}
	if err := os.Link(temp.Name(), target); err != nil {
		if errors.Is(err, os.ErrExist) {
			reportsError(w, http.StatusConflict, "A report with this filename already exists")
		} else {
			reportsError(w, http.StatusInternalServerError, "Unable to save report")
		}
		return
	}
	reportsJSON(w, http.StatusCreated, map[string]any{"success": true, "message": "Upload completed."})
}

func (s *ReportsAPI) archive(w http.ResponseWriter, r *http.Request, id string, download bool) {
	reports, err := s.list()
	if err != nil {
		reportsError(w, http.StatusInternalServerError, "Unable to read reports")
		return
	}
	for _, report := range reports {
		if report.ID != id {
			continue
		}
		pid, pidErr := s.running()
		if pidErr != nil {
			reportsError(w, http.StatusInternalServerError, "Unable to read generation status")
			return
		}
		if pid > 0 || s.cancel != nil {
			times, err := os.ReadFile(s.statusPath("time"))
			if err != nil {
				reportsError(w, http.StatusInternalServerError, "Unable to read generation range")
				return
			}
			if "hawk-"+strings.ReplaceAll(string(times), ",", "-") == report.Name {
				reportsError(w, http.StatusConflict, "This report is still being collected")
				return
			}
		}
		if download {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", strings.NewReplacer("+", "_", " ", "_").Replace(filepath.Base(report.archive))))
			http.ServeFile(w, r, report.archive)
			return
		}
		// Delete the archive last so a partial cleanup can be retried.
		base := filepath.Join(s.root, "reports", report.Name)
		for _, name := range []string{base, base + ".stdout", base + ".stderr"} {
			if err := os.RemoveAll(name); err != nil {
				reportsError(w, http.StatusInternalServerError, "Unable to delete report files")
				return
			}
		}
		if err := os.Remove(report.archive); err != nil {
			reportsError(w, http.StatusInternalServerError, "Unable to delete archive")
			return
		}
		reportsJSON(w, http.StatusOK, map[string]any{"success": true, "message": "Report deleted successfully"})
		return
	}
	reportsError(w, http.StatusNotFound, "The report does not exist")
}
