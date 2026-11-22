package internal

import (
	"flag"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestConfigCommandLine(t *testing.T) {
	defaults := Config{
		SessionDirectory: "/var/lib/hawk/go-sessions",
		Listen:           "0.0.0.0",
		Port:             17630,
		Key:              "/etc/hawk/hawk.key",
		Cert:             "/etc/hawk/hawk.pem",
		LogLevel:         "info",
		Route:            []ConfigRoute{},
	}
	for _, tt := range []struct {
		name string
		args []string
		want Config
	}{
		{"defaults", nil, defaults},
		{"overrides", []string{"-listen", "127.0.0.1", "-port", "7630", "-key", "/tmp/test.key", "-cert", "/tmp/test.pem", "-session-dir", "/tmp/test-sessions", "-loglevel", "debug"}, Config{
			SessionDirectory: "/tmp/test-sessions", Listen: "127.0.0.1", Port: 7630, Key: "/tmp/test.key", Cert: "/tmp/test.pem", LogLevel: "debug", Route: []ConfigRoute{},
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			originalFlags, originalArgs := flag.CommandLine, os.Args
			originalLevel, originalFormatter := log.GetLevel(), log.StandardLogger().Formatter
			t.Cleanup(func() {
				flag.CommandLine, os.Args = originalFlags, originalArgs
				log.SetLevel(originalLevel)
				log.SetFormatter(originalFormatter)
			})
			flag.CommandLine = flag.NewFlagSet("hawk-apiserver", flag.ContinueOnError)
			os.Args = append([]string{"hawk-apiserver"}, tt.args...)
			require.Equal(t, tt.want, InitConfig("test"))
			require.Nil(t, flag.Lookup("config"), "The obsolete JSON configuration flag must not be accepted")
		})
	}
}
