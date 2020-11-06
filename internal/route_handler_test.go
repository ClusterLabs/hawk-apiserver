package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteHandler(t *testing.T) {
	config := Config{}
	parseConfigFile("../config.json.example", &config)
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
