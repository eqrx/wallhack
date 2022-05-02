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

package bridge

import (
	"fmt"
	"io"
)

// maxFrameSize defines the maximum len in bytes of IP frames that are passed to the bridge.
const maxFrameSize = int(^uint16(0))

// writeIPFrame writes a complete IP frame given via data and returns any io error encountered. This is done by first
// sending the size of the IP frame as a uint16 and then the actual frame.
func writeIPFrame(dst io.Writer, src []byte) error {
	fLen := len(src)
	if fLen > maxFrameSize {
		panic(fmt.Sprintf("did not expect frame length to be greater than %d but was %d", maxFrameSize, fLen))
	}

	bytesWritten, err := dst.Write([]byte{byte(fLen >> 8), byte(fLen)})
	if err != nil {
		return fmt.Errorf("write frame header: %w", err)
	}

	if bytesWritten != 2 {
		panic("conn did not write header at once")
	}

	bytesWritten, err = dst.Write(src)
	if err != nil {
		return fmt.Errorf("write frame payload: %w", err)
	}

	if bytesWritten != fLen {
		panic("conn did not write payload at once")
	}

	return nil
}

// readIPFrame returns a complete IP frame as bytes and any io error encountered. It does so by first reading a uint16
// from the conn that indicates how large in bytes the following IP frame will be. It then reads that size and
// returns it.
func readIPFrame(src io.Reader) ([]byte, error) {
	fLenBytes := []byte{0, 0}

	bytesRead, err := src.Read(fLenBytes)
	if err != nil {
		return nil, fmt.Errorf("read frame header: %w", err)
	}

	if bytesRead != len(fLenBytes) {
		panic("conn did not read header at once")
	}

	fLen := int(uint16(fLenBytes[0])<<8 | uint16(fLenBytes[1]))

	bytes := make([]byte, fLen)

	bytesRead, err = src.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("read frame: %w", err)
	}

	if bytesRead != fLen {
		panic("conn did not read frame at once")
	}

	return bytes, nil
}
