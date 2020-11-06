package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ClusterLabs/hawk-apiserver/internal"
	"github.com/ClusterLabs/hawk-apiserver/server"
	log "github.com/sirupsen/logrus"
)

// the released version and buildDate. They are set by makefile or RPM
var version = "was not built correctly"

func initConfig() internal.Config {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
		DisableSorting:   true,
	})

	config := internal.Config{
		Listen:   "0.0.0.0",
		Port:     17630,
		Key:      "/etc/hawk/hawk.key",
		Cert:     "/etc/hawk/hawk.pem",
		LogLevel: "info",
		Route:    []internal.ConfigRoute{},
	}

	listen := flag.String("listen", config.Listen, "Address to listen to")
	port := flag.Int("port", config.Port, "Port to listen to")
	key := flag.String("key", config.Key, "TLS key file")
	cert := flag.String("cert", config.Cert, "TLS cert file")
	loglevel := flag.String("loglevel", config.LogLevel, "Log level (debug|info|warning|error|fatal|panic)")
	cfgfile := flag.String("config", "", "Configuration file")
	versionFlag := flag.Bool("version", false, "show version of hawk-apiserver")
	flag.Parse()

	if *cfgfile != "" {
		internal.ParseConfigFile(*cfgfile, &config)
	}

	if *listen != "0.0.0.0" {
		config.Listen = *listen
	}
	if *port != 17630 {
		config.Port = *port
	}
	if *key != "/etc/hawk/hawk.key" {
		config.Key = *key
	}
	if *cert != "/etc/hawk/hawk.pem" {
		config.Cert = *cert
	}
	if *loglevel != "info" {
		config.LogLevel = *loglevel
	}
	if *versionFlag == true {
		log.Infof("hawk-apiserver version: %s", version)
		os.Exit(0)
	}

	lvl, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.Errorf("Failed to parse loglevel \"%v\" (must be debug|info|warning|error|fatal|panic)", config.LogLevel)
		lvl = log.InfoLevel
	}
	log.SetLevel(lvl)

	return config
}

func main() {
	config := initConfig()
	routehandler := internal.NewRouteHandler(&config)
	routehandler.Cib.Start()

	log.Infof("Listening to https://%s:%d\n", config.Listen, config.Port)
	//TODO: this function should return errors
	server.ListenAndServeWithRedirect(fmt.Sprintf("%s:%d", config.Listen, config.Port), routehandler, config.Cert, config.Key)
}
