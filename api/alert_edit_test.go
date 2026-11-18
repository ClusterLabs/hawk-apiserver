package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const alertEditFixture = `<cib><configuration><alerts><alert id="notify" path="/tmp/agent"><instance_attributes id="params"><nvpair id="p1" name="MESSAGE" value="a &amp; b"/></instance_attributes><meta_attributes id="meta"><nvpair id="m1" name="timeout" value="60s"/></meta_attributes><recipient id="recipient1" value="ops@example.com"><instance_attributes id="rp"><nvpair id="rp1" name="SUBJECT" value="cluster"/></instance_attributes><meta_attributes id="rm"><nvpair id="rm1" name="timeout" value="10s"/></meta_attributes></recipient></alert><alert id="empty" path="/tmp/agent"/></alerts><resources><primitive id="dummy"/></resources></configuration></cib>`

func TestFetchAlert(t *testing.T) {
	run := func(_ context.Context, name string, args []string, input string) ([]byte, error) {
		require.Equal(t, "cibadmin", name)
		require.Equal(t, []string{"-Ql"}, args)
		return []byte(alertEditFixture), nil
	}
	for _, id := range []string{"notify", "empty", "dummy", "missing"} {
		w := httptest.NewRecorder()
		fetchAlert(w, httptest.NewRequest("GET", "/?id="+id, nil), run)
		if id == "dummy" || id == "missing" {
			require.Equal(t, 404, w.Code)
			continue
		}
		require.Equal(t, 200, w.Code)
		var alert alertCreate
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &alert))
		require.Equal(t, id, alert.ID)
		require.NotNil(t, alert.Params)
		require.NotNil(t, alert.Meta)
		require.NotNil(t, alert.Recipients)
		if id == "notify" {
			require.Equal(t, map[string]string{"MESSAGE": "a & b"}, alert.Params)
			require.Equal(t, map[string]string{"timeout": "60s"}, alert.Meta)
			require.Equal(t, []alertRecipient{{ID: "recipient1", Value: "ops@example.com", Params: map[string]string{"SUBJECT": "cluster"}, Meta: map[string]string{"timeout": "10s"}}}, alert.Recipients)
		}
	}
}

func TestUpdateAlert(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent with spaces")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh"), 0600))
	for _, tt := range []struct {
		name, id   string
		recipients []alertRecipient
		status     int
		failure    bool
	}{
		{name: "update", id: "notify", recipients: []alertRecipient{{ID: "r1", Value: "ops@example.com", Params: map[string]string{"TEXT": "it's okay"}}}, status: 200},
		{name: "remove all recipients", id: "notify", status: 200},
		{name: "command failure", id: "notify", status: 500, failure: true},
		{name: "missing", id: "missing", status: 404},
		{name: "primitive", id: "dummy", status: 404},
	} {
		t.Run(tt.name, func(t *testing.T) {
			alert := alertCreate{ID: tt.id, Path: path, Recipients: tt.recipients, Params: map[string]string{"FOO": "hello world"}, Meta: map[string]string{"timeout": "90s"}}
			payload, err := json.Marshal(alert)
			require.NoError(t, err)
			writes := 0
			run := func(_ context.Context, name string, args []string, input string) ([]byte, error) {
				if name == "cibadmin" {
					require.Equal(t, []string{"-Ql"}, args)
					return []byte(alertEditFixture), nil
				}
				writes++
				require.Equal(t, "crm", name)
				require.Equal(t, []string{"-F", "configure", "load", "update", "-"}, args)
				expected := "'alert' 'notify' '" + path + "' 'attributes' 'FOO=hello world' 'meta' 'timeout=90s'"
				if len(tt.recipients) > 0 {
					expected += " 'to' '{' 'ops@example.com' 'attributes' 'TEXT=it'\"'\"'s okay' '}'"
				}
				require.Equal(t, expected+"\n", input)
				if tt.failure {
					return nil, fmt.Errorf("Permission denied")
				}
				return nil, nil
			}
			w := httptest.NewRecorder()
			updateAlert(w, httptest.NewRequest("POST", "/", strings.NewReader(string(payload))), run)
			require.Equal(t, tt.status, w.Code, w.Body.String())
			if tt.status == 404 {
				require.Zero(t, writes)
			} else {
				require.Equal(t, 1, writes)
			}
		})
	}
	_, err := alertUpdateCommand(alertCreate{ID: "notify", Path: path, Params: map[string]string{"FOO": "line\nprimitive dummy"}})
	require.ErrorContains(t, err, "line breaks")
	_, err = alertUpdateCommand(alertCreate{ID: "notify", Path: path + "missing"})
	require.ErrorContains(t, err, "File not found")
	w := httptest.NewRecorder()
	updateAlert(w, httptest.NewRequest("GET", "/", nil), nil)
	require.Equal(t, 405, w.Code)
	w = httptest.NewRecorder()
	updateAlert(w, httptest.NewRequest("POST", "/", strings.NewReader("{")), nil)
	require.Equal(t, 400, w.Code)
}

func TestAlertEditPage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	w := httptest.NewRecorder()
	AlertEditHandler(w, httptest.NewRequest("GET", "/cib/live/alerts/notify/edit", nil))
	require.Equal(t, 200, w.Code)
	for _, value := range []string{"Edit Alert", `data-alert-id="notify"`, `readonly value="notify"`, `name="revert"`, `href="/cib/live/config/edit#alerts"`, "RenamePopup.js", "DeletePopup.js"} {
		require.Contains(t, w.Body.String(), value)
	}
}

func TestBatchAlertEditPage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, suffix := range []string{"", "/"} {
		response := httptest.NewRecorder()
		AlertEditHandler(response, httptest.NewRequest("GET", batchBase+"/alerts/notify/edit"+suffix+"?flash=created", nil))
		require.Equal(t, 200, response.Code)
		html := response.Body.String()
		for _, text := range []string{"Edit Alert", `data-editing="true"`, `data-alert-id="notify"`, `readonly value="notify"`, `data-api-base="/api/batch/hacluster/alert"`, `data-rename-api="/api/batch/hacluster/alert/rename"`, `data-delete-api="/api/batch/hacluster/alert/delete"`, `href="/cib/hacluster/config/edit#alerts"`, `id="btn-show-batch"`} {
			require.Contains(t, html, text)
		}
		require.NotContains(t, html, `href="/cib/live/`)
	}
}

func TestBatchAlertEditOperationsStayScoped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0600))
	backend, calls := batchTestBackend(t)
	original := backend.run
	backend.run = func(ctx context.Context, live bool, name string, args []string, input string) ([]byte, error) {
		out, err := original(ctx, live, name, args, input)
		if name == "cibadmin" {
			return []byte(alertEditFixture), nil
		}
		return out, err
	}
	response := batchTestRequest(backend, "GET", "alert/fetch?id=notify", "")
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `"MESSAGE":"a \u0026 b"`)
	require.Contains(t, response.Body.String(), `"value":"ops@example.com"`)
	*calls = nil
	alert := alertCreate{ID: "notify", Path: path, Recipients: []alertRecipient{{ID: "r1", Value: "new@example.com", Params: map[string]string{"SUBJECT": "updated"}, Meta: map[string]string{"timeout": "15s"}}}, Params: map[string]string{"MESSAGE": "updated event"}, Meta: map[string]string{"timeout": "60s"}}
	payload, err := json.Marshal(alert)
	require.NoError(t, err)
	response = batchTestRequest(backend, "POST", "alert/update", string(payload))
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Len(t, *calls, 2)
	require.Equal(t, []string{"-F", "configure", "load", "update", "-"}, (*calls)[1].args)
	for _, text := range []string{"'MESSAGE=updated event'", "'new@example.com'", "'SUBJECT=updated'", "'timeout=15s'"} {
		require.Contains(t, (*calls)[1].input, text)
	}
	for _, call := range *calls {
		require.False(t, call.live)
	}
	for _, tt := range []struct {
		operation, body string
		args            []string
	}{
		{"alert/rename", `{"oldID":"notify","newID":"renamed"}`, []string{"-D", "plain", "configure", "rename", "notify", "renamed"}},
		{"alert/delete", `"notify"`, []string{"--force", "configure", "delete", "notify"}},
	} {
		*calls = nil
		response = batchTestRequest(backend, "POST", tt.operation, tt.body)
		require.Equal(t, 200, response.Code, response.Body.String())
		require.Len(t, *calls, 2)
		require.Equal(t, tt.args, (*calls)[1].args)
		for _, call := range *calls {
			require.False(t, call.live)
		}
	}
	for _, tt := range []struct{ operation, body string }{
		{"alert/delete", `"dummy"`}, {"alert/rename", `{"oldID":"dummy","newID":"renamed"}`}, {"alert/rename", `{"oldID":"notify","newID":""}`},
	} {
		*calls = nil
		response = batchTestRequest(backend, "POST", tt.operation, tt.body)
		require.GreaterOrEqual(t, response.Code, 400)
		for _, call := range *calls {
			require.Equal(t, "cibadmin", call.name)
		}
	}
}
