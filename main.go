package main

import (
	"fmt"

	"github.com/ClusterLabs/hawk-apiserver/internal"
	"github.com/ClusterLabs/hawk-apiserver/server"
	log "github.com/sirupsen/logrus"
)

// the released version of the binary. Generated via makefile or buildsystem
var version = "was not built correctly"

func main() {
	// initialize configuration, which handle the different routes
	config := internal.InitConfig(version)
	routehandler := internal.NewRouteHandler(&config)
	// binds to pacemaker-go and serve CIB async info
	routehandler.Cib.Start()

	log.Infof("Listening to https://%s:%d\n", config.Listen, config.Port)
	//TODO: this function should return errors
	// an https server with a reverse proxy. http is redirected to https
	server.ListenAndServeWithRedirect(fmt.Sprintf("%s:%d", config.Listen, config.Port), routehandler, config.Cert, config.Key)
}
