package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pmezard/go-difflib/difflib"
)

const batchID = "hacluster"
const batchBase = "/cib/" + batchID
const batchAPI = "/api/batch/" + batchID
const batchShadowDirectory = "/var/lib/pacemaker/cib"

type batchCommand func(context.Context, bool, string, []string, string) ([]byte, error)

type batchArtifacts struct{ Input, Output, Summary, Graph, Dot []byte }
type batchBackend struct {
	mu        sync.Mutex
	run       batchCommand
	exists    func() (bool, error)
	prepare   func(string) error
	artifacts batchArtifacts
}

var batch = &batchBackend{run: runBatchCommand, prepare: prepareBatchDirectory, exists: func() (bool, error) {
	_, err := os.Stat(filepath.Join(batchShadowDirectory, "shadow."+batchID))
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}}

// Environment selection belongs to a subprocess, never to the Go server.
// Explicit live reads and shadow creation are separate from all staged writes.
func batchEnvironment(parent []string, live bool) []string {
	env := make([]string, 0, len(parent)+2)
	for _, value := range parent {
		name := strings.SplitN(value, "=", 2)[0]
		if name != "CIB_shadow" && name != "CIB_file" && name != "CIB_shadow_dir" {
			env = append(env, value)
		}
	}
	env = append(env, "CIB_shadow_dir="+batchShadowDirectory)
	if !live {
		env = append(env, "CIB_shadow="+batchID)
	}
	return env
}

// Ruby's batch CIB belongs to hacluster. Use the same OS account so both
// implementations can continue editing it without root-owned replacement files.
func batchCredential() (*syscall.Credential, error) {
	if os.Geteuid() != 0 {
		return nil, nil
	}
	account, err := user.Lookup(batchID)
	if err != nil {
		return nil, err
	}
	uid, err := strconv.ParseUint(account.Uid, 10, 32)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.ParseUint(account.Gid, 10, 32)
	if err != nil {
		return nil, err
	}
	return &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: []uint32{uint32(gid)}}, nil
}

func prepareBatchDirectory(directory string) error {
	credential, err := batchCredential()
	if err != nil {
		return err
	}
	if credential != nil {
		return os.Chown(directory, int(credential.Uid), int(credential.Gid))
	}
	return nil
}

func runBatchCommand(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	credential, err := batchCredential()
	if err != nil {
		return nil, err
	}
	cmd.Env = batchEnvironment(os.Environ(), live)
	// File-based simulation must not inherit CIB_shadow: Pacemaker can otherwise
	// update the shadow while saving the old status to --save-output.
	if name == "crm_simulate" {
		for _, arg := range args {
			if arg == "--xml-file" {
				cmd.Env = batchEnvironment(os.Environ(), true)
				break
			}
		}
	}
	cmd.Dir = "/"
	if credential != nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: credential}
		account, err := user.Lookup(batchID)
		if err != nil {
			return nil, err
		}
		env := make([]string, 0, len(cmd.Env)+1)
		for _, entry := range cmd.Env {
			key := strings.SplitN(entry, "=", 2)[0]
			if key != "HOME" && key != "XDG_CONFIG_HOME" && key != "XDG_CACHE_HOME" && key != "XDG_DATA_HOME" {
				env = append(env, entry)
			}
		}
		cmd.Env = append(env, "HOME="+account.HomeDir)
	}
	cmd.Stdin = strings.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	// Ruby treats crm_simulate's nonzero exit as an error only with stderr.
	if err != nil && !(name == "crm_simulate" && stderr.Len() == 0 && len(out) > 0) {
		return out, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

type batchState struct {
	Details   ClusterDetails `json:"details"`
	CRM       CrmStatus      `json:"crm"`
	Resources []ResourceRow  `json:"resources"`
	Nodes     []Node         `json:"nodes"`
	Tickets   []string       `json:"tickets"`
	LiveEpoch string         `json:"liveEpoch"`
}

func (b *batchBackend) readCIB(ctx context.Context, live bool) (CIB, []byte, error) {
	out, err := b.run(ctx, live, "cibadmin", []string{"-Ql"}, "")
	if err != nil {
		return CIB{}, nil, err
	}
	var cib CIB
	if err := xml.Unmarshal(out, &cib); err != nil {
		return cib, nil, err
	}
	cib.TagObjects, err = tagObjectsFromXML(out)
	if err != nil {
		return cib, nil, err
	}
	cib.LocationResources, err = locationResourcesFromXML(out)
	if err != nil {
		return cib, nil, err
	}
	cib.ColocationResources, err = colocationResourcesFromXML(out)
	if err != nil {
		return cib, nil, err
	}
	return cib, out, nil
}
func (b *batchBackend) state(ctx context.Context) (batchState, error) {
	cib, raw, err := b.readCIB(ctx, false)
	if err != nil {
		return batchState{}, err
	}
	live, _, err := b.readCIB(ctx, true)
	if err != nil {
		return batchState{}, err
	}
	out, err := b.run(ctx, false, "crm", []string{"status", "--as-xml"}, "")
	if err != nil {
		return batchState{}, err
	}
	var crm CrmStatus
	if err := xml.Unmarshal(out, &crm); err != nil {
		return batchState{}, err
	}
	tickets := []string{}
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "ticket_state" {
			for _, a := range start.Attr {
				if a.Name.Local == "id" {
					tickets = append(tickets, a.Value)
				}
			}
		}
	}
	return batchState{Details: clusterDetailsFromCIB(cib), CRM: crm, Resources: cibResourceRows(cib), Nodes: cib.Configuration.Nodes, Tickets: tickets, LiveEpoch: clusterDetailsFromCIB(live).Epoch}, nil
}

func (b *batchBackend) page(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	// Ruby editors still consume Ruby's status JSON. Do not change its schema.
	if r.URL.Query().Get("format") == "json" || strings.Contains(r.Header.Get("Accept"), "application/json") {
		Routehandler.ServeHTTP(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	b.mu.Lock()
	exists, err := b.exists()
	created := false
	if err == nil && (!exists || strings.EqualFold(r.URL.Query().Get("init_cib"), "true")) {
		_, err = b.run(ctx, true, "crm_shadow", []string{"-b", "-f", "-c", batchID}, "")
		created = err == nil
		if created {
			b.artifacts = batchArtifacts{}
		}
	}
	b.mu.Unlock()
	if err != nil {
		http.Error(w, "Unable to create shadow CIB: "+err.Error(), 500)
		return
	}
	if created {
		http.Redirect(w, r, batchBase, http.StatusSeeOther)
		return
	}
	renderTemplate(w, "live_status", map[string]any{"Title": "Status", "BatchMode": true, "CIBBase": batchBase, "BatchConfig": map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}})
}
func BatchHandler(w http.ResponseWriter, r *http.Request)    { batch.page(w, r) }
func BatchAPIHandler(w http.ResponseWriter, r *http.Request) { batch.api(w, r) }

func batchError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
func batchJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func (b *batchBackend) api(w http.ResponseWriter, r *http.Request) {
	operation := strings.TrimPrefix(r.URL.Path, batchAPI+"/")
	methods := map[string]string{"state": "GET", "details": "GET", "review": "GET", "result": "GET", "simulate": "POST", "action": "POST", "apply": "POST", "discard": "POST", "primitive/create": "POST", "group/children": "POST", "group/create": "POST", "clone/children": "POST", "clone/create": "POST", "tag/objects": "POST", "tag/create": "POST", "tag/fetch": "GET", "tag/update": "POST", "tag/rename": "POST", "tag/delete": "POST", "alert/create": "POST", "alert/fetch": "GET", "alert/update": "POST", "alert/rename": "POST", "alert/delete": "POST", "colocation/options": "POST", "colocation/create": "POST", "location/options": "POST", "location/create": "POST", "location/fetch": "POST", "location/update": "POST", "location/rename": "POST", "location/delete": "POST"}
	method, ok := methods[operation]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method != method {
		w.Header().Set("Allow", method)
		batchError(w, 405, fmt.Errorf("Method not allowed"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	b.mu.Lock()
	defer b.mu.Unlock()
	exists, err := b.exists()
	if err != nil {
		batchError(w, 500, err)
		return
	}
	if !exists {
		batchError(w, 409, fmt.Errorf("Batch session no longer exists. Enter batch mode again."))
		return
	}
	switch operation {
	case "details":
		cib, _, err := b.readCIB(ctx, false)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		batchJSON(w, clusterDetailsFromCIB(cib))
	case "state":
		state, err := b.state(ctx)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		batchJSON(w, state)
	case "review":
		diff, epoch, err := b.diff(ctx)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		batchJSON(w, map[string]string{"diff": diff, "epoch": epoch})
	case "simulate":
		var input struct {
			Injections []string `json:"injections"`
		}
		if !readBatchJSON(w, r, &input) {
			return
		}
		if err := b.simulate(ctx, input.Injections); err != nil {
			batchError(w, 422, err)
			return
		}
		state, err := b.state(ctx)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		batchJSON(w, state)
	case "result":
		b.result(w, r, ctx)
	case "action":
		var input batchAction
		if !readBatchJSON(w, r, &input) {
			return
		}
		if err := b.action(ctx, input); err != nil {
			batchError(w, 422, err)
			return
		}
		batchJSON(w, map[string]bool{"success": true})
	case "location/fetch":
		var input struct{}
		if !readBatchJSON(w, r, &input) {
			return
		}
		fetchLocation(w, r, func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
			return b.run(ctx, false, name, args, input)
		})
	case "location/update":
		var location locationCreate
		if !readBatchJSON(w, r, &location) {
			return
		}
		saveLocationUpdate(w, r, location, func() (CIB, int, error) {
			cib, _, err := b.readCIB(ctx, false)
			return cib, 0, err
		}, func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
			return b.run(ctx, false, name, args, input)
		}, batchBase)
	case "colocation/options":
		var input struct{}
		if !readBatchJSON(w, r, &input) {
			return
		}
		cib, _, err := b.readCIB(ctx, false)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		batchJSON(w, cib.ColocationResources)
	case "colocation/create":
		var colocation colocationCreate
		if !readBatchJSON(w, r, &colocation) {
			return
		}
		saveColocation(w, r, colocation, func() (CIB, int, error) {
			cib, _, err := b.readCIB(ctx, false)
			return cib, 0, err
		}, func(ctx context.Context, args ...string) error {
			_, err := b.run(ctx, false, "crm", args, "")
			return err
		}, batchBase)
	case "location/options":
		var input struct{}
		if !readBatchJSON(w, r, &input) {
			return
		}
		cib, _, err := b.readCIB(ctx, false)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		batchJSON(w, locationOptions(cib))
	case "location/create":
		var location locationCreate
		if !readBatchJSON(w, r, &location) {
			return
		}
		saveLocation(w, r, location, func() (CIB, int, error) {
			cib, _, err := b.readCIB(ctx, false)
			return cib, 0, err
		}, func(ctx context.Context, args ...string) error {
			_, err := b.run(ctx, false, "crm", args, "")
			return err
		}, batchBase)
	case "alert/fetch":
		fetchAlert(w, r, func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
			return b.run(ctx, false, name, args, input)
		})
	case "alert/update":
		var alert alertCreate
		if !readBatchJSON(w, r, &alert) {
			return
		}
		saveAlertUpdate(w, r, alert, func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
			return b.run(ctx, false, name, args, input)
		})
	case "alert/create":
		var alert alertCreate
		if !readBatchJSON(w, r, &alert) {
			return
		}
		saveAlert(w, r, alert, func(ctx context.Context, args ...string) error {
			_, err := b.run(ctx, false, "crm", args, "")
			return err
		}, batchBase)
	case "tag/fetch":
		fetchTag(w, r, func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
			return b.run(ctx, false, name, args, input)
		})
	case "tag/update":
		var tag tagCreate
		if !readBatchJSON(w, r, &tag) {
			return
		}
		saveTagUpdate(w, r, tag, func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
			return b.run(ctx, false, name, args, input)
		})
	case "tag/rename", "tag/delete", "alert/rename", "alert/delete", "location/rename", "location/delete":
		var id, newID string
		if strings.HasSuffix(operation, "/rename") {
			var input struct {
				OldID string `json:"oldID"`
				NewID string `json:"newID"`
			}
			if !readBatchJSON(w, r, &input) {
				return
			}
			id, newID = input.OldID, input.NewID
			if newID == "" {
				batchError(w, 400, fmt.Errorf("New ID is required"))
				return
			}
		} else if !readBatchJSON(w, r, &id) {
			return
		}
		action := tagAction
		if strings.HasPrefix(operation, "alert/") {
			action = alertAction
		} else if strings.HasPrefix(operation, "location/") {
			action = locationAction
		}
		status, err := action(ctx, id, newID, func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
			return b.run(ctx, false, name, args, input)
		})
		if err != nil {
			batchError(w, status, err)
			return
		}
		batchJSON(w, map[string]string{"status": "ok"})
	case "tag/objects":
		var input struct{}
		if !readBatchJSON(w, r, &input) {
			return
		}
		cib, _, err := b.readCIB(ctx, false)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		batchJSON(w, cib.TagObjects)
	case "tag/create":
		var tag tagCreate
		if !readBatchJSON(w, r, &tag) {
			return
		}
		saveTag(w, r, tag, func() (CIB, int, error) {
			cib, _, err := b.readCIB(ctx, false)
			return cib, 0, err
		}, func(ctx context.Context, args ...string) error {
			_, err := b.run(ctx, false, "crm", args, "")
			return err
		}, batchBase)
	case "clone/children":
		var input struct{}
		if !readBatchJSON(w, r, &input) {
			return
		}
		cib, _, err := b.readCIB(ctx, false)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		content := SelectContent{}
		for _, id := range cloneChildren(cib) {
			content.Options = append(content.Options, SelectOption{Name: id})
		}
		batchJSON(w, content)
	case "clone/create":
		var clone Clone
		if !readBatchJSON(w, r, &clone) {
			return
		}
		createClone(w, r, clone, func() (CIB, int, error) {
			cib, _, err := b.readCIB(ctx, false)
			return cib, 0, err
		}, func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
			return b.run(ctx, false, name, args, input)
		}, batchBase)
	case "group/children":
		var input struct{}
		if !readBatchJSON(w, r, &input) {
			return
		}
		cib, _, err := b.readCIB(ctx, false)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		batchJSON(w, groupChildren(cib))
	case "group/create":
		var group groupCreate
		if !readBatchJSON(w, r, &group) {
			return
		}
		saveGroup(w, r, group, func() (CIB, int, error) {
			cib, _, err := b.readCIB(ctx, false)
			return cib, 0, err
		}, func(ctx context.Context, args ...string) error {
			_, err := b.run(ctx, false, "crm", args, "")
			return err
		}, batchBase)
	case "primitive/create":
		var primitive Primitive
		if !readBatchJSON(w, r, &primitive) {
			return
		}
		createPrimitive(w, r, primitive, func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
			return b.run(ctx, false, name, args, input)
		}, batchBase)
	case "discard":
		var input struct{}
		if !readBatchJSON(w, r, &input) {
			return
		}
		if err := b.deleteShadow(ctx); err != nil {
			batchError(w, 500, err)
			return
		}
		clearBatchEvents(w)
		batchJSON(w, map[string]string{"redirect": "/cib/live"})
	case "apply":
		var input struct {
			Epoch string `json:"epoch"`
		}
		if !readBatchJSON(w, r, &input) {
			return
		}
		_, raw, err := b.readCIB(ctx, false)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		liveText, err := b.run(ctx, true, "crm", []string{"-D", "plain", "configure", "show"}, "")
		if err != nil {
			batchError(w, 500, err)
			return
		}
		revision, err := batchReviewRevision(raw, liveText)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		if input.Epoch == "" || input.Epoch != revision {
			batchError(w, 409, fmt.Errorf("The batch configuration changed. Review the changes again before applying."))
			return
		}
		if _, err := b.run(ctx, false, "crm", []string{"-F", "configure", "verify"}, ""); err != nil {
			batchError(w, 422, err)
			return
		}
		// This is the sole intentional transition from the shadow to the live CIB.
		if _, err := b.run(ctx, false, "crm", []string{"-F", "-f", "-", "configure"}, "cib commit "+batchID+"\n"); err != nil {
			batchError(w, 422, err)
			return
		}
		if err := b.deleteShadow(ctx); err != nil {
			batchError(w, 500, fmt.Errorf("Configuration was applied, but the shadow CIB could not be deleted: %w", err))
			return
		}
		clearBatchEvents(w)
		batchJSON(w, map[string]string{"message": "Applied configuration successfully", "redirect": "/cib/live?batch_applied=true"})
	}
}
func clearBatchEvents(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "hawk-batch-events", Path: "/", MaxAge: -1, SameSite: http.SameSiteLaxMode})
}

func (b *batchBackend) deleteShadow(ctx context.Context) error {
	if _, err := b.run(ctx, false, "crm_shadow", []string{"-b", "-f", "-D", batchID}, ""); err != nil {
		return err
	}
	b.artifacts = batchArtifacts{}
	return nil
}

func readBatchJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		batchError(w, 415, fmt.Errorf("Expected application/json"))
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		batchError(w, 400, fmt.Errorf("Invalid batch request"))
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		batchError(w, 400, fmt.Errorf("Invalid batch request"))
		return false
	}
	return true
}

func (b *batchBackend) diff(ctx context.Context) (string, string, error) {
	_, raw, err := b.readCIB(ctx, false)
	if err != nil {
		return "", "", err
	}
	live, err := b.run(ctx, true, "crm", []string{"-D", "plain", "configure", "show"}, "")
	if err != nil {
		return "", "", err
	}
	shadow, err := b.run(ctx, false, "crm", []string{"-D", "plain", "configure", "show"}, "")
	if err != nil {
		return "", "", err
	}
	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{A: difflib.SplitLines(string(live)), B: difflib.SplitLines(string(shadow)), Context: 0})
	lines := []string{}
	for _, line := range strings.Split(diff, "\n") {
		if !strings.HasPrefix(line, "---") && !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "@@") {
			lines = append(lines, line)
		}
	}
	revision, revisionErr := batchReviewRevision(raw, live)
	if err == nil {
		err = revisionErr
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), revision, err
}

// Simulation updates status frequently. Review tokens track configuration only,
// plus the live baseline the user reviewed, rather than transient CIB epochs.
func batchReviewRevision(shadow, liveText []byte) (string, error) {
	configuration, err := xmlSubtree(shadow, "configuration")
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, _ = hash.Write(configuration)
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(liveText)
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func xmlSubtree(data []byte, name string) ([]byte, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var out bytes.Buffer
	encoder := xml.NewEncoder(&out)
	depth := 0
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("Missing %s in CIB", name)
		}
		if start, ok := token.(xml.StartElement); ok {
			if depth > 0 || start.Name.Local == name {
				depth++
			}
		}
		if depth == 0 {
			continue
		}
		if err := encoder.EncodeToken(token); err != nil {
			return nil, err
		}
		if _, ok := token.(xml.EndElement); ok {
			depth--
			if depth == 0 {
				if err := encoder.Flush(); err != nil {
					return nil, err
				}
				return out.Bytes(), nil
			}
		}
	}
}

var batchSafeID = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.:-]*$`)

func batchObjects(raw []byte) (map[string]bool, map[string]bool) {
	resources, nodes := map[string]bool{}, map[string]bool{}
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		for _, attr := range start.Attr {
			if (start.Name.Local == "primitive" || start.Name.Local == "group" || start.Name.Local == "clone" || start.Name.Local == "master") && attr.Name.Local == "id" {
				resources[attr.Value] = true
			}
			if start.Name.Local == "node" && attr.Name.Local == "uname" {
				nodes[attr.Value] = true
			}
		}
	}
	return resources, nodes
}
func batchInjectionArgs(injections []string, raw []byte) ([]string, error) {
	if len(injections) > 100 {
		return nil, fmt.Errorf("Too many simulation events")
	}
	resources, nodes := batchObjects(raw)
	args := []string{}
	for _, event := range injections {
		p := strings.Fields(event)
		if len(p) == 3 && p[0] == "node" && nodes[p[1]] {
			flag := map[string]string{"online": "-u", "offline": "-d", "unclean": "-f"}[p[2]]
			if flag != "" {
				args = append(args, flag, p[1])
				continue
			}
		}
		if len(p) == 5 && p[0] == "op" && nodes[p[4]] && batchSafeID.MatchString(p[2]) {
			op := strings.Split(p[1], ":")
			rc, e := strconv.Atoi(p[3])
			known := resources[p[2]]
			if !known {
				base := strings.Split(p[2], ":")[0]
				known = resources[base]
			}
			if len(op) == 2 && known && e == nil && rc >= 0 && rc <= 9 {
				interval, e := strconv.Atoi(op[1])
				allowed := map[string]bool{"monitor": true, "start": true, "stop": true, "promote": true, "demote": true, "notify": true, "migrate_to": true, "migrate_from": true}
				if e == nil && interval >= 0 && allowed[op[0]] {
					args = append(args, "-i", fmt.Sprintf("%s_%s_%d@%s=%d", p[2], op[0], interval, p[4], rc))
					continue
				}
			}
		}
		if len(p) == 3 && p[0] == "ticket" && batchSafeID.MatchString(p[1]) {
			// Only existing tickets can be injected, as in the Ruby selector.
			ticket := false
			decoder := xml.NewDecoder(bytes.NewReader(raw))
			for {
				token, e := decoder.Token()
				if e != nil {
					break
				}
				if start, ok := token.(xml.StartElement); ok && start.Name.Local == "ticket_state" {
					for _, a := range start.Attr {
						if a.Name.Local == "id" && a.Value == p[1] {
							ticket = true
						}
					}
				}
			}
			flag := map[string]string{"grant": "-g", "revoke": "-r", "standby": "-b", "activate": "-e"}[p[2]]
			if ticket && flag != "" {
				args = append(args, flag, p[1])
				continue
			}
		}
		return nil, fmt.Errorf("Invalid simulation event: %s", event)
	}
	return args, nil
}

func (b *batchBackend) simulate(ctx context.Context, injections []string) error {
	_, shadow, err := b.readCIB(ctx, false)
	if err != nil {
		return err
	}
	args, err := batchInjectionArgs(injections, shadow)
	if err != nil {
		return err
	}
	_, live, err := b.readCIB(ctx, true)
	if err != nil {
		return err
	}
	status, err := xmlSubtree(live, "status")
	if err != nil {
		return err
	}
	if _, err := b.run(ctx, false, "cibadmin", []string{"-p", "--replace", "--scope", "status"}, string(status)); err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "hawk-batch-sim-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	if b.prepare != nil {
		if err := b.prepare(directory); err != nil {
			return err
		}
	}
	graph, dot := filepath.Join(directory, "graph.xml"), filepath.Join(directory, "graph.dot")
	result := filepath.Join(directory, "result.xml")
	input := filepath.Join(directory, "input.xml")
	_, initial, err := b.readCIB(ctx, false)
	if err != nil {
		return err
	}
	// Give the simulation account access without exposing the CIB snapshot.
	if err := os.WriteFile(input, initial, 0600); err != nil {
		return err
	}
	if b.prepare != nil {
		if err := b.prepare(input); err != nil {
			return err
		}
	}
	output, err := b.run(ctx, false, "crm_simulate", append([]string{"-S", "--xml-file", input, "-G", graph, "-D", dot, "-O", result}, args...), "")
	if err != nil {
		return err
	}
	// Persist the simulated status explicitly: crm_simulate may write its output
	// to a temporary CIB rather than the selected shadow, depending on version.
	simulated, err := os.ReadFile(result)
	if err != nil {
		return err
	}
	status, err = xmlSubtree(simulated, "status")
	if err != nil {
		return err
	}
	if _, err := b.run(ctx, false, "cibadmin", []string{"-p", "--replace", "--scope", "status"}, string(status)); err != nil {
		return err
	}
	graphData, err := os.ReadFile(graph)
	if err != nil {
		return err
	}
	dotData, err := os.ReadFile(dot)
	if err != nil {
		return err
	}
	_, out, err := b.readCIB(ctx, false)
	if err != nil {
		return err
	}
	b.artifacts = batchArtifacts{Input: live, Output: out, Summary: output, Graph: graphData, Dot: dotData}
	return nil
}
func (b *batchBackend) result(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	file := r.URL.Query().Get("file")
	content := "text/plain; charset=utf-8"
	var out []byte
	switch file {
	case "info":
		out = b.artifacts.Summary
	case "graph-xml":
		out = b.artifacts.Graph
		content = "text/xml; charset=utf-8"
	case "in":
		out = b.artifacts.Input
		content = "text/xml; charset=utf-8"
	case "out":
		out = b.artifacts.Output
		content = "text/xml; charset=utf-8"
	case "graph":
		if len(b.artifacts.Dot) == 0 {
			batchError(w, 409, fmt.Errorf("Run the simulation first"))
			return
		}
		var err error
		out, err = b.run(ctx, false, "dot", []string{"-Tsvg"}, string(b.artifacts.Dot))
		if err != nil {
			batchError(w, 500, err)
			return
		}
		content = "image/svg+xml"
	case "diff":
		diff, _, err := b.diff(ctx)
		if err != nil {
			batchError(w, 500, err)
			return
		}
		out = []byte(diff)
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", content)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(out)
}

type batchAction struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Action string `json:"action"`
	Node   string `json:"node"`
}

func (b *batchBackend) action(ctx context.Context, input batchAction) error {
	_, raw, err := b.readCIB(ctx, false)
	if err != nil {
		return err
	}
	resources, nodes := batchObjects(raw)
	if !batchSafeID.MatchString(input.ID) {
		return fmt.Errorf("Invalid object ID")
	}
	if input.Kind == "resource" && resources[input.ID] {
		command := map[string][]string{"start": {"start"}, "stop": {"stop"}, "promote": {"promote"}, "demote": {"demote"}, "maintenance-on": {"maintenance"}, "maintenance-off": {"maintenance"}, "clear": {"clear"}}[input.Action]
		if input.Action == "migrate" {
			if input.Node != "" && !nodes[input.Node] {
				return fmt.Errorf("Invalid destination node")
			}
			if input.Node == "" {
				command = []string{"ban"}
			} else {
				command = []string{"move"}
			}
		}
		if command == nil {
			return fmt.Errorf("Unsupported staged resource action")
		}
		args := append([]string{"-F", "resource"}, command...)
		args = append(args, input.ID)
		if input.Action == "maintenance-off" {
			args = append(args, "false")
		}
		if input.Action == "migrate" && input.Node != "" {
			args = append(args, input.Node)
		}
		_, err := b.run(ctx, false, "crm", args, "")
		return err
	}
	if input.Kind == "node" && nodes[input.ID] {
		option := map[string]string{"standby": "standby", "online": "standby", "maintenance": "maintenance", "ready": "maintenance"}[input.Action]
		if option == "" {
			return fmt.Errorf("Unsupported staged node action")
		}
		value := "true"
		if input.Action == "online" || input.Action == "ready" {
			value = "false"
		}
		_, err := b.run(ctx, false, "crm_attribute", []string{"--type", "nodes", "--node", input.ID, "--name", option, "--update", value}, "")
		return err
	}
	return fmt.Errorf("The object does not exist in the batch configuration")
}
