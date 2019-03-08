package main

import (
	"encoding/xml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func TestParseMetrics(t *testing.T) {
	monxml, err := ioutil.ReadFile("./test/crm-mon-2.xml")
	assert.Nil(t, err, "Failed to read test data")

	var status crmMon
	err = xml.Unmarshal(monxml, &status)
	assert.Nil(t, err, "Failed to unmarshal test data")

	metrics := parseMetrics(&status)
	assert.Equal(t, metrics.Node.Configured, 1, "Should have one node")
}

type mockWriter struct {
	mock.Mock
}

func (m *mockWriter) Header() http.Header {
	return http.Header{}
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
	testObj.On("Write", mock.Anything).Return(0, nil).Times(29)
	ret := handleMetrics(testObj)
	assert.Equal(t, true, ret, "expected success")
	testObj.AssertExpectations(t)
}
