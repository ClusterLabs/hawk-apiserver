package main

import (
	"github.com/ClusterLabs/hawk-apiserver/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigParse(t *testing.T) {
	config := util.Config{}
	util.ParseConfigFile("./config.json.example", &config)
	assert.Equal(t, config.Port, 7630, "Port should be 7630")
}

func TestGetStdout(t *testing.T) {
	want := "hello"
	got := util.GetStdout("echo", "-n", "hello")
	assert.Equal(t, got, want, "Unexpected output")
}
