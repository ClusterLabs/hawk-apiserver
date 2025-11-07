package api

import (
	"regexp"
)

// "\x1b[31mERROR\x1b[0m:..." --> "ERROR:..."
func stripANSI(s string) string {
	ansiRE := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	return ansiRE.ReplaceAllString(s, "")
}
