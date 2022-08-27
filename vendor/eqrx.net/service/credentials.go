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

package service

import (
	"fmt"
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

// CredPath returns the path credentials that contains systemd credentials.
func (s Service) CredPath(name string) string {
	if s.credsDir == "" {
		panic("credentials directory not set by systemd")
	}

	return s.credsDir + "/" + name
}

// UnmarshalYAMLCreds unmarshals the YAML credential file called name into dst.
func (s Service) UnmarshalYAMLCreds(name string, dst interface{}) error {
	if s.credsDir == "" {
		panic("credentials directory not set by systemd")
	}

	return UnmarshalYAMLCreds(s.credsDir, name, dst)
}

// UnmarshalYAMLCreds unmarshals the YAML credential file called name into dst.
func UnmarshalYAMLCreds(dir, name string, dst interface{}) error {
	credFile, err := os.Open(path.Join(dir, name))
	if err != nil {
		return fmt.Errorf("open cred file: %w", err)
	}

	err = yaml.NewDecoder(credFile).Decode(dst)

	closeErr := credFile.Close()

	switch {
	case err != nil && closeErr != nil:
		return fmt.Errorf("decode cred: %w; close cred file: %v", err, closeErr)
	case err != nil:
		return fmt.Errorf("decode cred: %w", err)
	case closeErr != nil:
		return fmt.Errorf("close cred file: %w", closeErr)
	default:
		return nil
	}
}
