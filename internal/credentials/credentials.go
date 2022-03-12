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

// Package credentials handles to loading and unmarshalling of wallhack credentials using systemd.
package credentials

import (
	"errors"
	"fmt"
	"os"
	"path"
)

const (
	credDirEnvName = "CREDENTIALS_DIRECTORY"
	credName       = "wallhack"
	// ALPNWallhack is the protocol name of wallhack for ALPN. This string is set as the next protocol of
	// a wallhack TLS connection to distinguish it from HTTP traffic.
	ALPNWallhack = "wallhack"
)

var errCredDirectoryNotSet = errors.New("env CREDENTIALS_DIRECTORY not set")

// LoadBytes returns the content of the wallhack credential file offered by systemd.
func LoadBytes() ([]byte, error) {
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
