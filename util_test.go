package main

import (
	"testing"

	"github.com/ClusterLabs/hawk-apiserver/internal"
	"github.com/stretchr/testify/assert"
)

func TestConfigParse(t *testing.T) {
	config := internal.Config{}
	internal.ParseConfigFile("./config.json.example", &config)
	assert.Equal(t, config.Port, 7630, "Port should be 7630")
}
