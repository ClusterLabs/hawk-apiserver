package api

import (
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
