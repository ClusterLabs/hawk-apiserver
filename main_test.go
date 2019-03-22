package main

import (
	"github.com/ClusterLabs/hawk-apiserver/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRouteHandler(t *testing.T) {
	config := util.Config{}
	util.ParseConfigFile("./config.json.example", &config)
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
