package main

import (
	"testing"
)

func TestConfigParse(t *testing.T) {
	config := Config{}
	parseConfigFile("./config.json.example", &config)
	if config.Port != 7630 {
		t.Fatal("expected 7630, got ", config.Port)
	}
}
