package main

import (
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestRubyAuthPreservesBatchContext(t *testing.T) {
	for _, path := range []string{"/cib/hacluster", "/cib/hacluster/", "/api/batch/hacluster/state", "/api/cib/resource/params/fetch?cib_id=hacluster"} {
		require.Equal(t, "http://localhost/internal/auth?cib_id=hacluster", rubyAuthURL(httptest.NewRequest("GET", path, nil)))
	}
	for _, path := range []string{"/cib/live", "/api/cib/resource/stop?cib_id=hacluster", "/api/batch/other/state", "/api/cib/resource/params/fetch?cib_id=other"} {
		require.Equal(t, "http://localhost/internal/auth", rubyAuthURL(httptest.NewRequest("GET", path, nil)))
	}
}
