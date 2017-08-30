package main_test

import (
	"testing"
	"net/http"
	"github.com/krig/hawk-apiserver"
)

type TestHandler struct {
	val int
}

func (h *TestHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
}

func incAdapter(h http.Handler) http.Handler {
	h.(*TestHandler).val++
	return h
}

func TestAdapt(t *testing.T) {
	th := TestHandler{0}
	main.Adapt(&th, incAdapter, incAdapter)
	if th.val != 2 {
		t.Fatal("expected th=2, got ", th.val)
	}
}
