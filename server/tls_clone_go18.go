// +build go1.8

package server

import "crypto/tls"

// cloneTLSConfig
//
// The Clone() func was added in go 1.8.
func cloneTLSConfig(cfg *tls.Config) *tls.Config {
	if cfg != nil {
		return cfg.Clone()
	}
	return &tls.Config{}
}
