package main

import (
	"github.com/ClusterLabs/hawk-apiserver/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"os"
	"testing"
)

type mockWriter struct {
	mock.Mock
	head http.Header
}

func (m *mockWriter) Header() http.Header {
	if m.head == nil {
		m.head = make(http.Header)
	}
	return m.head
}

func (m *mockWriter) Write(bytes []byte) (int, error) {
	args := m.Called(bytes)
	return args.Int(0), args.Error(1)
}

func (m *mockWriter) WriteHeader(statusCode int) {
}

func TestHandleMetrics(t *testing.T) {
	os.Setenv("CIB_file", "./test/cib4.xml")

	testObj := new(mockWriter)
	testObj. //On("Header").Once().
		On("Write", mock.Anything).Return(0, nil).Times(29)
	ret := metrics.HandleMetrics(testObj)
	assert.True(t, ret, "Should print metrics for the example CIB")
	testObj.AssertExpectations(t)
}
