package internal

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"
)

// Config holds runtime command-line settings and internal route definitions.
type Config struct {
	SessionDirectory string        `json:"session_directory"`
	Listen           string        `json:"listen"`
	Port             int           `json:"port"`
	Key              string        `json:"key"`
	Cert             string        `json:"cert"`
	LogLevel         string        `json:"loglevel"`
	Route            []ConfigRoute `json:"route"`
}

// ConfigRoute is used in the configuration to map routes to handlers.
//
// Possible handlers (this list may be outdated)a:
//
//   - `api/v1` - Exposes a CIB API endpoint.
//   - `monitor` - Typically mapped to `/monitor` to handle
//     long-polling for CIB updates.
//   - `file` - A static file serving route mapped to a directory.
//   - `proxy` - Proxies requests to another server.
type ConfigRoute struct {
	Handler string  `json:"handler"`
	Path    string  `json:"path"`
	Target  *string `json:"target"`
}

// InitConfig initializes runtime settings from command-line flags.
func InitConfig(version string) Config {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
		DisableSorting:   true,
	})

	config := Config{
		SessionDirectory: "/var/lib/hawk/go-sessions",
		Listen:           "0.0.0.0",
		Port:             17630,
		Key:              "/etc/hawk/hawk.key",
		Cert:             "/etc/hawk/hawk.pem",
		LogLevel:         "info",
		Route:            []ConfigRoute{},
	}

	flag.StringVar(&config.SessionDirectory, "session-dir", config.SessionDirectory, "Directory for Go authentication sessions")
	flag.StringVar(&config.Listen, "listen", config.Listen, "Address to listen to")
	flag.IntVar(&config.Port, "port", config.Port, "Port to listen to")
	flag.StringVar(&config.Key, "key", config.Key, "TLS key file")
	flag.StringVar(&config.Cert, "cert", config.Cert, "TLS cert file")
	flag.StringVar(&config.LogLevel, "loglevel", config.LogLevel, "Log level (debug|info|warning|error|fatal|panic)")
	versionFlag := flag.Bool("version", false, "show version of hawk-apiserver")
	flag.Parse()

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
