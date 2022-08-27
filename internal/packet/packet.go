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

// Package packet provides access to ipv6 packet handling.
package packet

import (
	"errors"
	"fmt"

	"golang.org/x/net/ipv6"
)

var (
	errVersion = errors.New("unsupported ip version")
	errJumbo   = errors.New("unsupported jumbo packet")
)

// Packet is a [ipv6.Header] and a slice of the whole marshalled packet.
type Packet struct {
	Header     *ipv6.Header
	Marshalled []byte
}

func asHeader(data []byte) (*ipv6.Header, error) {
	header, err := ipv6.ParseHeader(data)
	if err != nil {
		return nil, fmt.Errorf("as packet header: %w", err)
	}

	if version := header.Version; version != ipv6.Version {
		return nil, fmt.Errorf("as packet header: %w: %d", errVersion, version)
	}

	if header.PayloadLen == 0 {
		return nil, fmt.Errorf("as packet header: %w", errJumbo)
	}

	return header, nil
}
