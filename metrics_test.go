package main

import (
	"encoding/xml"
	"github.com/stretchr/testify/assert"
	"testing"
	"io/ioutil"
)

func TestParseMetrics(t *testing.T) {
	monxml, err := ioutil.ReadFile("./test/crm-mon-2.xml")
	assert.Nil(t, err, "Failed to read test data")
	
	var status crmMon
	err = xml.Unmarshal(monxml, &status)
	assert.Nil(t, err, "Failed to unmarshal test data")

	metrics := parseMetrics(&status)
	assert.Equal(t, metrics.Node.Total, 1, "Should have one node")
}
