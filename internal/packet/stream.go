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
	"bytes"
	"fmt"
	"io"

	"golang.org/x/net/ipv6"
)

// StreamReader reads ip packets from a stream (not a tun).
type StreamReader struct {
	reader io.Reader
	buffer *bytes.Buffer
}

// NewStreamReader creates a new [NewStreamReader] with the unterlying reader.
func NewStreamReader(reader io.Reader) *StreamReader {
	return &StreamReader{reader, &bytes.Buffer{}}
}

// ReadPacket reads an IP packet from the stream.
func (n *StreamReader) ReadPacket() (*Packet, error) {
	if _, err := io.CopyN(n.buffer, n.reader, ipv6.HeaderLen); err != nil {
		return nil, fmt.Errorf("read packet: %w", err)
	}

	header, err := asHeader(n.buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("read packet: %w", err)
	}

	if _, err := io.CopyN(n.buffer, n.reader, int64(header.PayloadLen)); err != nil {
		return nil, fmt.Errorf("read packet: %w", err)
	}

	marshalled := n.buffer.Bytes()
	n.buffer.Reset()

	return &Packet{header, marshalled}, nil
}
