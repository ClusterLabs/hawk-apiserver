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
	config := internal.InitConfig(version)
	routehandler := internal.NewRouteHandler(&config)
	routehandler.Cib.Start()

	log.Infof("Listening to https://%s:%d\n", config.Listen, config.Port)
	//TODO: this function should return errors
	server.ListenAndServeWithRedirect(fmt.Sprintf("%s:%d", config.Listen, config.Port), routehandler, config.Cert, config.Key)
}
