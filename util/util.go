package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
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

// ParseConfigFile is a configuration file parser.
//
// The configuration file format is described in
// config.json.example and README.md.
func ParseConfigFile(cfgfile string, target *Config) {
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

// CheckHawkAuthMethods validates a HTTP request,
// returning true if it's good.
//
// Current methods:
//
// * Hawk attrd cookie
// * Basic Auth (user/passwd)
//
// Future methods?
//
// * API key?
func CheckHawkAuthMethods(r *http.Request) bool {
	// Try hawk attrd cookie
	var user string
	var session string
	for _, c := range r.Cookies() {
		if c.Name == "hawk_remember_me_id" {
			user = c.Value
		}
		if c.Name == "hawk_remember_me_key" {
			session = c.Value
		}
	}
	if user != "" && session != "" {
		cmd := exec.Command("/usr/sbin/attrd_updater", "-R", "-Q", "-A", "-n", fmt.Sprintf("hawk_session_%v", user))
		if cmd != nil {
			out, _ := cmd.StdoutPipe()
			cmd.Start()
			// for each line, look for value="..."
			// if ... == sessioncookie, then OK
			scanner := bufio.NewScanner(out)
			tomatch := fmt.Sprintf("value=\"%v\"", session)
			for scanner.Scan() {
				l := scanner.Text()
				if strings.Contains(l, tomatch) {
					log.Printf("Valid session cookie for %v", user)
					return true
				}
			}
			cmd.Wait()
		}
	}
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	if !CheckBasicAuth(user, pass) {
		return false
	}
	return true
}

// CheckBasicAuth does HTTP Basic Auth checking against
// a system user/pass with some help from
// /usr/sbin/hawk_chkpwd.
func CheckBasicAuth(user, pass string) bool {
	// /usr/sbin/hawk_chkpwd passwd <user>
	// write password
	// close
	cmd := exec.Command("/usr/sbin/hawk_chkpwd", "passwd", user)
	if cmd == nil {
		log.Print("Authorization failed: /usr/sbin/hawk_chkpwd not found")
		return false
	}
	cmd.Stdin = strings.NewReader(pass)
	err := cmd.Run()
	if err != nil {
		log.Printf("Authorization failed: %v", err)
		return false
	}
	return true
}

// GetStdout runs the given command with the given
// arguments and returns its output, or exits on
// error.
func GetStdout(cmd string, args ...string) string {
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		log.Fatal(err)
	}
	return string(out)
}
