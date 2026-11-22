package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteHandler(t *testing.T) {
	target := "unix:///tmp/hawk-route-handler-test.sock"
	config := Config{Route: []ConfigRoute{
		{Handler: "monitor", Path: "/monitor"},
		{Handler: "proxy", Path: "/", Target: &target},
	}}
	routeHandler := NewRouteHandler(&config)
	assert.NotNil(t, routeHandler)

	for route := range config.Route {
		if config.Route[route].Handler == "proxy" {
			p1 := routeHandler.proxyForRoute(&config.Route[route])
			p2 := routeHandler.proxyForRoute(&config.Route[route])
			assert.Equal(t, p1, p2, "route cache returns inconsistent results")
		}
	}
}
