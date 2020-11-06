package internal

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

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
	if !checkBasicAuth(user, pass) {
		return false
	}
	return true
}

// checkBasicAuth does HTTP Basic Auth checking against
// a system user/pass with some help from
// /usr/sbin/hawk_chkpwd.
func checkBasicAuth(user, pass string) bool {
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
