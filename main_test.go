package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigParse(t *testing.T) {
	config := Config{}
	parseConfigFile("./config.json.example", &config)
	assert.Equal(t, config.Port, 7630, "Port should be 7630")
}

func TestRouteHandler(t *testing.T) {
	config := Config{}
	parseConfigFile("./config.json.example", &config)
	routeHandler := newRouteHandler(&config)
	assert.NotNil(t, routeHandler)

	for route := range config.Route {
		if config.Route[route].Handler == "proxy" {
			p1 := routeHandler.proxyForRoute(&config.Route[route])
			p2 := routeHandler.proxyForRoute(&config.Route[route])
			assert.Equal(t, p1, p2, "route cache returns inconsistent results")
		}
	}
}
