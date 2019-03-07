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

func TestRouteHandler(t *testing.T) {
	config := Config{}
	parseConfigFile("./config.json.example", &config)
	routeHandler := newRouteHandler(&config)
	if routeHandler == nil {
		t.Fatal("Failed to create a route handler")
	}

	for route := range config.Route {
		if config.Route[route].Handler == "proxy" {
			p1 := routeHandler.proxyForRoute(&config.Route[route])
			p2 := routeHandler.proxyForRoute(&config.Route[route])
			if p1 != p2 {
				t.Fatalf("route cache returns inconsistent results for route #%v", route)
			}
		}
	}
}
