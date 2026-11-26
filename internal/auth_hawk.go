package internal

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// AuthenticatePassword uses the existing privileged PAM helper directly. The
// password travels through stdin, never argv, a shell, or the Rails process.
func AuthenticatePassword(ctx context.Context, username, password string) error {
	return authenticatePasswordUsing(ctx, "/usr/sbin/hawk-apiserver_chkpwd", username, password)
}

func authenticatePasswordUsing(ctx context.Context, helper, username, password string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, helper, "passwd", username)
	cmd.Stdin = strings.NewReader(password)
	return cmd.Run()
}
