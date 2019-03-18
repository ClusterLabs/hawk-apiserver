package main

import (
	"testing"
)

func TestGetStdout(t *testing.T) {
	want := "hello"
	got := GetStdout("echo", "-n", "hello")
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
