package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigParse(t *testing.T) {
	config := Config{}
	parseConfigFile("../config.json.example", &config)
	assert.Equal(t, config.Port, 7630, "Port should be 7630")
}
