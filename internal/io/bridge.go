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

package io

import (
	"fmt"
	"net"
)

// maxBridgeFrameSize defines the maximum len in bytes of IP frames that are passed to the bridge.
const maxBridgeFrameSize = int(^uint16(0))

// bridge wraps a net.Conn and allows to stream ip frames over it.
type bridge struct {
	conn net.Conn
}

// writeIPFrame writes a complete IP frame as bytes and returns any io error encountered. This is done by first
// sending the size of the IP frame as a uint16 and then the actual frame.
//nolint:gomnd // Byte shifting.
func (t *bridge) writeIPFrame(f []byte) error {
	fLen := len(f)
	if fLen > maxBridgeFrameSize {
		panic(fmt.Sprintf("did not expect frame length to be greater than %d but was %d", maxBridgeFrameSize, fLen))
	}

	n, err := t.conn.Write([]byte{byte(fLen >> 8), byte(fLen)})
	if err != nil {
		return fmt.Errorf("write frame header: %w", err)
	}

	if n != 2 {
		panic("conn did not write header at once")
	}

	n, err = t.conn.Write(f)
	if err != nil {
		return fmt.Errorf("write frame payload: %w", err)
	}

	if n != fLen {
		panic("conn did not write payload at once")
	}

	return nil
}

// readIPFrame returns a complete IP frame as bytes and any io error encountered. It does so by first reading a uint16
// from the conn that indicates how large in bytes the following IP frame will be. It then reads that size and
// returns it.
//nolint:gomnd // Byte shifting.
func (t *bridge) readIPFrame() ([]byte, error) {
	fLenBytes := []byte{0, 0}

	n, err := t.conn.Read(fLenBytes)
	if err != nil {
		return nil, fmt.Errorf("read frame header: %w", err)
	}

	if n != 2 {
		panic("conn did not read header at once")
	}

	fLen := int(uint16(fLenBytes[0])<<8 | uint16(fLenBytes[1]))

	f := make([]byte, fLen)

	n, err = t.conn.Read(f)
	if err != nil {
		return nil, fmt.Errorf("read frame: %w", err)
	}

	if n != fLen {
		panic("conn did not read frame at once")
	}

	return f, nil
}
