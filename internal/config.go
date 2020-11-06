package internal

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

type offsetContext struct {
	start int
	end   int
	line  int
	pos   int
}

func contextAtOffset(str string, offset int64) offsetContext {
	start, end := strings.LastIndex(str[:offset], "\n")+1, len(str)
	if idx := strings.Index(str[start:], "\n"); idx >= 0 {
		end = start + idx
	}
	line, pos := strings.Count(str[:start], "\n"), int(offset)-start-1
	return offsetContext{
		start: start,
		end:   end,
		line:  line,
		pos:   pos,
	}
}

func fatalSyntaxError(js string, err error) {
	syntax, ok := err.(*json.SyntaxError)
	if !ok {
		log.Fatal(err)
		return
	}
	ctx := contextAtOffset(js, syntax.Offset)
	log.Printf("Error in line %d: %s", ctx.line, err)
	log.Printf("%s", js[ctx.start:ctx.end])
	log.Fatalf("%s^", strings.Repeat(" ", ctx.pos))
}

// Config is the internal representation of the configuration file.
type Config struct {
	Listen   string        `json:"listen"`
	Port     int           `json:"port"`
	Key      string        `json:"key"`
	Cert     string        `json:"cert"`
	LogLevel string        `json:"loglevel"`
	Route    []ConfigRoute `json:"route"`
}

// ConfigRoute is used in the configuration to map routes to handlers.
//
// Possible handlers (this list may be outdated)a:
//
//   * `api/v1` - Exposes a CIB API endpoint.
//   * `monitor` - Typically mapped to `/monitor` to handle
//     long-polling for CIB updates.
//   * `file` - A static file serving route mapped to a directory.
//   * `proxy` - Proxies requests to another server.
type ConfigRoute struct {
	Handler string  `json:"handler"`
	Path    string  `json:"path"`
	Target  *string `json:"target"`
}

// parseConfigFile is a configuration file parser.
//
// The configuration file format is described in
// config.json.example and README.md.
func parseConfigFile(cfgfile string, target *Config) {
	log.Printf("Reading %v...", cfgfile)
	raw, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		log.Fatal(err)
		return
	}
	err = json.Unmarshal(raw, target)
	if err != nil {
		fatalSyntaxError(string(raw), err)
	}
}

// InitConfig initialize the configuration
func InitConfig(version string) Config {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
		DisableSorting:   true,
	})

	config := Config{
		Listen:   "0.0.0.0",
		Port:     17630,
		Key:      "/etc/hawk/hawk.key",
		Cert:     "/etc/hawk/hawk.pem",
		LogLevel: "info",
		Route:    []ConfigRoute{},
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
		parseConfigFile(*cfgfile, &config)
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
