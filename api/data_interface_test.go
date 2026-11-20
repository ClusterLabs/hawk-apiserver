package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOperationCibNameValuesUsesAttributeNames(t *testing.T) {
	action := NewFullPrimitiveActionFromOperation(Operation{
		Name:     "monitor",
		ID:       "dummy-monitor-10s",
		Interval: "10s",
		Timeout:  "20s",
	})

	assert.Equal(t, []NameValue{
		{Name: "interval", Value: "10s"},
		{Name: "timeout", Value: "20s"},
	}, operationCibNameValues(action))
}

func TestDataInterfaceHandlersStopOnJSONDecodeError(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"FetchResourceUtilizations":  FetchResourceUtilizations,
		"SubmitResourceUtilizations": SubmitResourceUtilizations,
		"FetchNodeAttributes":        FetchNodeAttributes,
		"SubmitNodeAttributes":       SubmitNodeAttributes,
		"FetchNodeUtilizations":      FetchNodeUtilizations,
		"SubmitNodeUtilizations":     SubmitNodeUtilizations,
	}

	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			callLog := filepath.Join(t.TempDir(), "calls")
			installFakeCibadmin(t, `#!/bin/sh
echo cibadmin >> `+callLog+`
printf '%s\n' '<cib><configuration><nodes><node id="1" uname="node1"><instance_attributes id="nodes-1"><nvpair id="nodes-1-keep" name="keep" value="1"/></instance_attributes><utilization id="nodes-1-u"><nvpair id="nodes-1-u-keep" name="keep" value="1"/></utilization></node></nodes><resources><primitive id="rsc1" class="ocf" provider="heartbeat" type="Dummy"><utilization id="rsc1-u"><nvpair id="rsc1-u-keep" name="keep" value="1"/></utilization></primitive></resources></configuration></cib>'
`)
			installFakeCrm(t, `#!/bin/sh
echo "crm $*" >> `+callLog+`
`)

			// "value" must be a string, so decoding fails after CibObject
			// and the first nvpair name have already been filled in.
			body := `{"CibObject":"node1","nvpair":[{"name":"weight","value":100}]}`
			if strings.HasPrefix(name, "Fetch") || strings.Contains(name, "Resource") {
				body = `{"CibObject":1}`
			}
			rec := httptest.NewRecorder()
			handler(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "Invalid request\n", rec.Body.String())
			_, err := os.Stat(callLog)
			assert.True(t, os.IsNotExist(err), "no cibadmin/crm command should run after a decode error")
		})
	}
}
