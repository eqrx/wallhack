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
	"errors"
	"fmt"
	"io"

	"golang.org/x/net/ipv6"
)

var errPortionMissing = errors.New("missing parts of packet")

// MTUByteReader extends [io.Reader] by allowing to query the MTU size.
type MTUByteReader interface {
	io.Reader
	MTU() int
}

// MTUReader reads IP packets over a non streamed (one IP packets is put into one read)
// MTU restricted connection. Consecutive packet reads use the same buffer.
type MTUReader struct {
	reader MTUByteReader
	buffer []byte
}

// NewMTUReader creates a mew [MTUReader] with the given [MTUByteReader] as its source.
func NewMTUReader(reader MTUByteReader) *MTUReader {
	return &MTUReader{reader, make([]byte, reader.MTU()+1)}
}

// ReadPacket reads an IP packet from the stream. When receiving a packet that is
// larger then the last queried MTU size it is dropped, the buffer is resized to the current
// MTU size and a new read is performed.
func (n *MTUReader) ReadPacket() (*Packet, error) {
mturetry:
	bytesRead, err := n.reader.Read(n.buffer)

	if err != nil {
		return nil, fmt.Errorf("read packet: %w", err)
	}

	if bytesRead == len(n.buffer) {
		n.buffer = make([]byte, n.reader.MTU()+1)

		goto mturetry
	}

	data := n.buffer[:bytesRead]

	header, err := asHeader(data)
	if err != nil {
		return nil, fmt.Errorf("read packet: %w", err)
	}

	if header.PayloadLen != bytesRead-ipv6.HeaderLen {
		return nil, fmt.Errorf("%w: expected %d, got %d", errPortionMissing, header.PayloadLen+ipv6.HeaderLen, bytesRead)
	}

	return &Packet{header, data}, nil
}
