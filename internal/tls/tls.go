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

// Package tls manages TLS configurations required by client and server.
package tls

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path"
)

const (
	credDirEnvName = "CREDENTIALS_DIRECTORY"
	credName       = "wallhack"
)

var (
	errCredDirectoryNotSet = errors.New("env CREDENTIALS_DIRECTORY not set")
	errPEM                 = errors.New("PEMs in credential file invalid")
)

func getCredentials() ([]byte, error) {
	credsDirectory, ok := os.LookupEnv(credDirEnvName)
	if !ok {
		return nil, errCredDirectoryNotSet
	}

	credBytes, err := os.ReadFile(path.Join(credsDirectory, credName))
	if err != nil {
		return nil, fmt.Errorf("read credential file: %w", err)
	}

	return credBytes, nil
}

func getCertsFromCredentials(data []byte) ([]crypto.PrivateKey, []*x509.Certificate, *x509.CertPool, error) {
	keys := make([]crypto.PrivateKey, 0, 1)
	certs := make([]*x509.Certificate, 0, 1)
	caPool := x509.NewCertPool()

	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)

		switch {
		case block == nil:
			return nil, nil, nil, errPEM
		case block.Type == "CERTIFICATE":
			currentCert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, nil, errPEM
			}

			if currentCert.IsCA {
				caPool.AddCert(currentCert)
			} else {
				certs = append(certs, currentCert)
			}
		case block.Type == "PRIVATE KEY":
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("read key from credential file: %w", errPEM)
			}

			keys = append(keys, key)
		default:
			return nil, nil, nil, fmt.Errorf("%w: unknown PEM type %s", errPEM, block.Type)
		}
	}

	return keys, certs, caPool, nil
}

// Config creates a tls config from the credentials file supplied by systemd.
func Config() (*tls.Config, error) {
	credBytes, err := getCredentials()
	if err != nil {
		return nil, err
	}

	keys, certs, caPool, err := getCertsFromCredentials(credBytes)
	if err != nil {
		return nil, err
	}

	switch {
	case len(caPool.Subjects()) != 1:
		return nil, fmt.Errorf("%w: expected one CA, got %d", errPEM, len(caPool.Subjects()))
	case len(keys) != 1:
		return nil, fmt.Errorf("%w: expected one key, got %d", errPEM, len(keys))
	case len(certs) != 1:
		return nil, fmt.Errorf("%w: expected one cert, got %d", errPEM, len(certs))
	}

	tlsCert := tls.Certificate{Certificate: [][]byte{certs[0].Raw}, PrivateKey: keys[0]}
	config := &tls.Config{
		Certificates:             []tls.Certificate{tlsCert},
		RootCAs:                  caPool,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                caPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS13,
	}

	return config, nil
}
