// +build !go1.8

package main

import "crypto/tls"

// cloneTLSConfig
//
// tls.Config.Clone() was added in go 1.8, so
// use this func in older versions.
func cloneTLSConfig(config *tls.Config) *tls.Config {
	if config != nil {
		return &tls.Config{
			Rand: config.Rand,
			Time: config.Time,
			Certificates: config.Certificates,
			NameToCertificate: config.NameToCertificate,
			GetCertificate: config.GetCertificate,
			RootCAs: config.RootCAs,
			NextProtos: config.NextProtos,
			ServerName: config.ServerName,
			ClientAuth: config.ClientAuth,
			ClientCAs: config.ClientCAs,
			InsecureSkipVerify: config.InsecureSkipVerify,
			CipherSuites: config.CipherSuites,
			PreferServerCipherSuites: config.PreferServerCipherSuites,
			ClientSessionCache: config.ClientSessionCache,
			MinVersion: config.MinVersion,
			MaxVersion: config.MaxVersion,
			CurvePreferences: config.CurvePreferences,
		}
	}
	return &tls.Config{}
}
