// Copyright (C) 2021 Alexander Sowitzki
//
// This program is free software: you can redistribute it and/or modify it under the terms of the
// GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied
// warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License for more
// details.
//
// You should have received a copy of the GNU Affero General Public License along with this program.
// If not, see <https://www.gnu.org/licenses/>.

// Package env handles environment variable.
package env

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// errEnvNotFound indicates that a looked up env variable is not set.
var errEnvNotFound = errors.New("required env variable not set")

const (
	// Mode indicates if this wallhack instance shall be run in "server" or "client" mode.
	Mode = "WALLHACK_MODE"
	// Certificate wallhack shall use to authenticate against its peer. This is not a file name but the certificate itself.
	Certificate = "WALLHACK_CERTIFICATE"
	// Key wallhack shall use to encrypt traffic to its peer. This is not a file name but the key itself.
	Key = "WALLHACK_KEY"
	// CA wallhack shall use to authenticate its peer peer. This is not a file name but the certificate itself.
	CA = "WALLHACK_CA"
	// ServerAddr is the address a client shall connect to.
	ServerAddr = "WALLHACK_SERVER"
)

// Lookup the environment variable name and return an error if not found.
// Just a wrapper for os.Lookup.
func Lookup(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return value, fmt.Errorf("%w: %s", errEnvNotFound, name)
	}

	return value, nil
}

// CreateTLSConfig using environment variable names defined above.
func CreateTLSConfig() (*tls.Config, error) {
	certStr, err := Lookup(Certificate)
	if err != nil {
		return nil, fmt.Errorf("%w: certificate for authenticating with peer", err)
	}

	keyStr, err := Lookup(Key)
	if err != nil {
		return nil, fmt.Errorf("%w: key for authenticating with peer", err)
	}

	caStr, err := Lookup(CA)
	if err != nil {
		return nil, fmt.Errorf("%w: certificate to validate peer", err)
	}

	cert, err := tls.X509KeyPair([]byte(certStr), []byte(keyStr))
	if err != nil {
		return nil, fmt.Errorf("parse tls certificate pair: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM([]byte(caStr)) {
		panic("empty CA was given?")
	}

	return &tls.Config{
		ClientCAs:                caCertPool,
		RootCAs:                  caCertPool,
		Certificates:             []tls.Certificate{cert},
		CurvePreferences:         []tls.CurveID{tls.CurveP521},
		MinVersion:               tls.VersionTLS13,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		PreferServerCipherSuites: true,
	}, nil
}
