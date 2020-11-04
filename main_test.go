package main

import (
	"testing"

	"github.com/ClusterLabs/hawk-apiserver/internal"
	"github.com/stretchr/testify/assert"
)

func TestRouteHandler(t *testing.T) {
	config := internal.Config{}
	internal.ParseConfigFile("./config.json.example", &config)
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

func TestInitProcedure(t *testing.T) {
	config := initConfig()
	assert.Equal(t, config.Port, 17630, "Expected default port")
}
