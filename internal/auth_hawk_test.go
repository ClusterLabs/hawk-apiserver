package internal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPasswordHelperUsesArgvAndStdin(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "chkpwd")
	// Check the helper contract without invoking PAM or touching system accounts.
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\n[ \"$#\" = 2 ] || exit 2\n[ \"$1\" = passwd ] || exit 3\n[ \"$2\" = 'user;echo unsafe' ] || exit 4\nvalue=$(cat)\n[ \"$value\" = 'a$pass word' ] || exit 5\n"), 0700))
	require.NoError(t, authenticatePasswordUsing(context.Background(), script, "user;echo unsafe", "a$pass word"))
	require.Error(t, authenticatePasswordUsing(context.Background(), script, "other", "a$pass word"))
	require.Error(t, authenticatePasswordUsing(context.Background(), filepath.Join(dir, "missing"), "alice", "test"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, authenticatePasswordUsing(ctx, script, "user;echo unsafe", "a$pass word"))
}

func TestPasswordHelperTimeout(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "chkpwd")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nexec sleep 30\n"), 0700))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := authenticatePasswordUsing(ctx, script, "alice", "test")
	require.Error(t, err)
	require.Less(t, time.Since(start), 2*time.Second)
	require.False(t, strings.Contains(err.Error(), "test"))
}
