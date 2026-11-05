package api

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAlertCreateArgs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent with spaces")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0600))
	alert := alertCreate{ID: "notify", Path: path, Params: map[string]string{"message": "hello world"}, Meta: map[string]string{"timeout": "30s"}, Recipients: []alertRecipient{{ID: "recipient1", Value: "some user@example.com", Params: map[string]string{"subject": "cluster event"}, Meta: map[string]string{"timestamp-format": "%H:%M:%S"}}}}
	args, err := alertCreateArgs(alert)
	require.NoError(t, err)
	require.Equal(t, []string{"configure", "alert", "notify", path, "attributes", "message=hello world", "meta", "timeout=30s", "to", "{", "some user@example.com", "attributes", "subject=cluster event", "meta", "timestamp-format=%H:%M:%S", "}"}, args)
	payload, err := json.Marshal(alert)
	require.NoError(t, err)
	response := httptest.NewRecorder()
	calls := 0
	createAlert(response, httptest.NewRequest("POST", "/", strings.NewReader(string(payload))), func(_ context.Context, got ...string) error { calls++; require.Equal(t, args, got); return nil })
	require.Equal(t, 200, response.Code)
	require.Equal(t, 1, calls)
	require.Contains(t, response.Body.String(), "/cib/live/alerts/notify/edit?flash=created")
	response = httptest.NewRecorder()
	createAlert(response, httptest.NewRequest("POST", "/", strings.NewReader(string(payload))), func(context.Context, ...string) error {
		return errors.New("\x1b[31mERROR\x1b[0m: agent failure: exit status 1")
	})
	require.Equal(t, 500, response.Code)
	require.Equal(t, "agent failure\n", response.Body.String())
	alert.Path = filepath.Join(t.TempDir(), "missing")
	_, err = alertCreateArgs(alert)
	require.ErrorContains(t, err, "File not found")
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(alert.Path, link))
	alert.Path = link
	_, err = alertCreateArgs(alert)
	require.NoError(t, err) // Ruby allows dangling symlinks.
	alert.Recipients[0].Value = ""
	_, err = alertCreateArgs(alert)
	require.ErrorContains(t, err, "Recipient path is required")
	alert.Recipients = nil
	alert.Meta["invented"] = "1"
	_, err = alertCreateArgs(alert)
	require.ErrorContains(t, err, "Unknown alert meta attribute")
}

func TestAlertTemplateAndMapping(t *testing.T) {
	tmpl, err := template.ParseFiles("../templates/alert_new.html")
	require.NoError(t, err)
	var body strings.Builder
	require.NoError(t, tmpl.ExecuteTemplate(&body, "content", nil))
	require.Contains(t, body.String(), `name="alert[id]"`)
	require.Contains(t, body.String(), `name="alert[path]"`)
	options := alertMetaOptions()
	require.Len(t, options, 2)
	require.Equal(t, "30s", options[0].DefaultValue)
	require.Equal(t, "%H:%M:%S.%06N", options[1].DefaultValue)
	response := httptest.NewRecorder()
	AlertNewHandler(response, httptest.NewRequest("POST", "/cib/live/alerts/new", nil))
	require.Equal(t, 405, response.Code)
}
