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

package packet

import (
	"fmt"
	"io"
)

type (
	// Reader reads IP packets.
	Reader interface {
		ReadPacket() (*Packet, error)
	}

	// ReadWriteCloser provides the functionality of [Reader], allows writing
	// ip packets and forwards functionality of [io.ReadWriteCloser].
	ReadWriteCloser struct {
		io.ReadWriteCloser
		Reader
	}
)

// NewReadWriteCloser creates a new [ReadWriteCloser] with sub as the underlying [io.ReadWriteCloser]
// and reader as [Reader] implementation.
func NewReadWriteCloser(sub io.ReadWriteCloser, reader Reader) *ReadWriteCloser {
	return &ReadWriteCloser{sub, reader}
}

// WritePacket writes the marshalled form of [Packet] over the stream.
func (r *ReadWriteCloser) WritePacket(p *Packet) error {
	_, err := r.Write(p.Marshalled)
	if err != nil {
		return fmt.Errorf("write packet: %w", err)
	}

	return nil
}
