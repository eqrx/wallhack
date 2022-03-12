// Copyright (C) 2022 Alexander Sowitzki
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

package credentials

import (
	"crypto/tls"
	"fmt"
)

// Client represents the credentials for running in client mode.
type Client struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

// TLSConf generates the TLS configuration for read credentials. It can
// be used to connect to a wallhack server.
func (c Client) TLSConf() (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(c.Cert), []byte(c.Key))
	if err != nil {
		return nil, fmt.Errorf("parse cert: %w", err)
	}

	config := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS13,
		NextProtos:               []string{ALPNWallhack},
	}

	return config, nil
}
