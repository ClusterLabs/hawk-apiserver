package internal

import (
	"os"
	"strings"
)

// OSRelease contains the distribution ID and VERSION_ID from /etc/os-release.
type OSRelease struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// ApplicationConfig holds shared application settings exposed to Go-rendered pages.P
type ApplicationConfig struct {
	OS OSRelease `json:"os"`
}

// AppConfig is initialized once at startup.
var AppConfig = ApplicationConfig{
	OS: readOSRelease(),
}

func readOSRelease() OSRelease {
	contents, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return OSRelease{}
	}
	return parseOSRelease(string(contents))
}

func parseOSRelease(contents string) OSRelease {
	var release OSRelease
	for _, line := range strings.Split(contents, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), "\"'")
		switch key {
		case "ID":
			release.ID = value
		case "VERSION_ID":
			release.Version = value
		}
	}
	return release
}
