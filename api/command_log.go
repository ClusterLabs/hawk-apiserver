package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const commandLogRoot = "/var/lib/hawk/tmp"
const commandLogDelimiter = "@@COMMAND-END@@\n"

type commandLog struct{ root string }

func (s commandLog) path(batchMode bool) string {
	name := "live"
	if batchMode {
		name = batchID
	}
	return filepath.Join(s.root, "commands-"+name+".log")
}

func (s commandLog) append(batchMode bool, command string) error {
	if err := os.MkdirAll(s.root, 0750); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path(batchMode), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() > 128*1024 || (info.Size() > 32*1024 && info.ModTime().Before(time.Now().Add(-24*time.Hour))) {
		if err = f.Truncate(0); err != nil {
			return err
		}
	}
	_, err = f.WriteString("# " + time.Now().Format(time.RFC3339) + "\n" + command + commandLogDelimiter)
	return err
}

func (s commandLog) read(batchMode bool) ([]string, error) {
	commands := []string{}
	f, err := os.Open(s.path(batchMode))
	if errors.Is(err, os.ErrNotExist) {
		return commands, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_SH); err != nil {
		return nil, err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	// Read through the locked descriptor; the writer rotates this file in place.
	var text strings.Builder
	_, err = io.Copy(&text, f)
	if err != nil {
		return nil, err
	}
	for _, entry := range strings.Split(text.String(), commandLogDelimiter) {
		if entry = strings.TrimSpace(entry); entry != "" {
			commands = append(commands, entry)
		}
	}
	return commands, nil
}

func CommandsJSONHandler(w http.ResponseWriter, r *http.Request) {
	serveCommandsJSON(w, r, commandLog{root: commandLogRoot})
}

func serveCommandsJSON(w http.ResponseWriter, r *http.Request, store commandLog) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	commands, err := store.read(strings.HasPrefix(r.URL.Path, batchBase+"/"))
	if err != nil {
		http.Error(w, "Unable to read command log", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodHead {
		_ = json.NewEncoder(w).Encode(commands)
	}
}

// Keep monitoring and metadata queries out of the command history.
func recordClusterCommand(name string, args []string, input string, batchMode bool) {
	kind := filepath.Base(name)
	mutation := false
	if kind == "cibadmin" {
		for _, arg := range args {
			if arg == "--modify" || arg == "--delete" || arg == "--replace" || arg == "--create" || arg == "-M" || arg == "-D" || arg == "-R" || arg == "-C" {
				mutation = true
			}
		}
	} else if kind == "crm" {
		for i, arg := range args {
			if arg == "resource" || arg == "node" {
				mutation = true
			}
			if arg == "configure" && i+1 < len(args) {
				switch args[i+1] {
				case "show", "graph", "get_property", "verify":
				default:
					mutation = true
				}
			}
		}
		if strings.TrimSpace(input) != "" && !strings.HasPrefix(strings.TrimSpace(input), "script json") {
			mutation = true
		}
	}
	if !mutation {
		return
	}
	command := name + " " + strings.Join(args, " ")
	if input != "" {
		command += "\n" + input
	}
	if err := (commandLog{root: commandLogRoot}).append(batchMode, command); err != nil {
		log.Printf("[commands] Unable to record command: %v", err)
	}
}

// Legacy mutation handlers use exec.Cmd directly; retain their execution behavior.
func loggedClusterCommand(name string, args ...string) *exec.Cmd {
	recordClusterCommand(name, args, "", false)
	return exec.Command(name, args...)
}

// Ruby records the equivalent script-run command, rather than the JSON transport.
func recordWizardCommand(ctx context.Context, name, operation string, params any) {
	if operation != "run" {
		return
	}
	data, err := json.Marshal(params)
	if err != nil {
		return
	}
	values := map[string]any{}
	if err = json.Unmarshal(data, &values); err != nil {
		return
	}
	args := []string{"/usr/sbin/crm", "script", "run", name}
	var add func(string, map[string]any)
	add = func(prefix string, values map[string]any) {
		names := make([]string, 0, len(values))
		for key := range values {
			names = append(names, key)
		}
		sort.Strings(names)
		for _, key := range names {
			if nested, ok := values[key].(map[string]any); ok {
				add(prefix+key+":", nested)
			} else {
				value := strings.ReplaceAll(fmt.Sprint(values[key]), "'", "'\"'\"'")
				args = append(args, prefix+key+"='"+value+"'")
			}
		}
	}
	add("", values)
	if err = (commandLog{root: commandLogRoot}).append(wizardIsBatch(ctx), strings.Join(args, " ")); err != nil {
		log.Printf("[commands] Unable to record wizard command: %v", err)
	}
}
