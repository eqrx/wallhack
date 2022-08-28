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

// LoadCred loads a credential file.
func (s Service) LoadCred(name string) ([]byte, error) {
	if s.credsDir == "" {
		panic("credentials directory not set by systemd")
	}

	return LoadCred(s.credsDir, name)
}

// UnmarshalYAMLCred unmarshals the YAML credential file called name into dst.
func (s Service) UnmarshalYAMLCred(name string, dst interface{}) error {
	if s.credsDir == "" {
		panic("credentials directory not set by systemd")
	}

	return UnmarshalYAMLCred(s.credsDir, name, dst)
}

// LoadCred loads a credential file.
func LoadCred(dir, name string) ([]byte, error) {
	data, err := os.ReadFile(path.Join(dir, name))
	if err != nil {
		return nil, fmt.Errorf("load cred: %w", err)
	}

	return data, err
}

// UnmarshalYAMLCred unmarshals the YAML credential file called name into dst.
func UnmarshalYAMLCred(dir, name string, dst interface{}) error {
	data, err := LoadCred(dir, name)
	if err != nil {
		return fmt.Errorf("unmarshal YAML creds: %w", err)
	}

	if err := yaml.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("unmarshal yaml cred: %w", err)
	}

	return nil
}
