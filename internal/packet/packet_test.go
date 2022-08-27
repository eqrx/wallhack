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

package packet_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"eqrx.net/wallhack/internal/packet"
	"golang.org/x/net/ipv6"
)

type (
	packetReader interface {
		ReadPacket() (*packet.Packet, error)
	}

	buffer interface {
		AddChunk(data []byte)
	}

	unit struct {
		name   string
		reader packetReader
		buffer buffer
	}

	chunkBuffer struct {
		chunks [][]byte
		mtu    int
	}

	streamBuffer struct {
		buf *bytes.Buffer
	}
)

func (s *streamBuffer) AddChunk(data []byte)          { _, _ = s.buf.Write(data) }
func (s *streamBuffer) Read(data []byte) (int, error) { return s.buf.Read(data) } //nolint:wrapcheck

func (c *chunkBuffer) AddChunk(data []byte) { c.chunks = append(c.chunks, data) }
func (c *chunkBuffer) MTU() int             { return c.mtu }
func (c *chunkBuffer) Read(data []byte) (int, error) {
	if len(c.chunks) == 0 {
		return 0, io.EOF
	}

	d := c.chunks[0]
	c.chunks = c.chunks[1:]

	return copy(data, d), nil
}

const mtu = ipv6.HeaderLen + 5

func mtuReader() unit {
	chunkBuf := &chunkBuffer{[][]byte{}, mtu}

	return unit{"mtuReader", packet.NewMTUReader(chunkBuf), chunkBuf}
}

func streamReader() unit {
	streamBuf := &streamBuffer{&bytes.Buffer{}}

	return unit{"streamReader", packet.NewStreamReader(streamBuf), streamBuf}
}

func readers() []unit { return []unit{mtuReader(), streamReader()} }

func dummyPacket(payloadLen uint8) []byte {
	b := make([]byte, ipv6.HeaderLen+payloadLen)
	b[0] = 0x60
	b[5] = payloadLen

	return b
}

func TestReader(t *testing.T) {
	t.Parallel()

	for _, iterUnit := range readers() {
		unit := iterUnit
		t.Run(unit.name, func(t *testing.T) {
			t.Parallel()

			payloadLen := mtu - ipv6.HeaderLen
			unit.buffer.AddChunk(dummyPacket(uint8(payloadLen)))

			packet, err := unit.reader.ReadPacket()
			if err != nil {
				t.Fatalf("read ip: %v", err)
			}

			if len(packet.Marshalled) != ipv6.HeaderLen+payloadLen {
				t.Fatalf("wrong ip len: want %d, have %d", ipv6.HeaderLen+payloadLen, len(packet.Marshalled))
			}

			if packet.Header.PayloadLen != payloadLen {
				t.Fatalf("wrong payload len: want %d, have %d", payloadLen, packet.Header.PayloadLen)
			}

			if packet.Header.Version != ipv6.Version {
				t.Fatalf("wrong ip version: want %d, have %d", ipv6.Version, packet.Header.Version)
			}
		})
	}
}

func TestCleanEOF(t *testing.T) {
	t.Parallel()

	for _, iterUnit := range readers() {
		unit := iterUnit
		t.Run(unit.name, func(t *testing.T) {
			t.Parallel()

			_, err := unit.reader.ReadPacket()
			if !errors.Is(err, io.EOF) {
				t.Fatalf("read ip: %v", err)
			}
		})
	}
}

func FuzzPortionMissing(f *testing.F) {
	payloadLen := mtu - ipv6.HeaderLen
	for i := 0; i < (payloadLen + ipv6.HeaderLen); i++ {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, chunkLen int) {
		for _, iterUnit := range readers() {
			unit := iterUnit
			t.Run(unit.name, func(t *testing.T) {
				t.Parallel()

				data := dummyPacket(uint8(payloadLen))
				unit.buffer.AddChunk(data[:chunkLen])

				p, err := unit.reader.ReadPacket()
				if err == nil {
					t.Fatalf("this did not fail: %d, %d", chunkLen, len(p.Marshalled))
				}
			})
		}
	})
}

func FuzzOverMTU(f *testing.F) {
	for i := mtu - ipv6.HeaderLen + 1; i < ((mtu - ipv6.HeaderLen) + 5); i++ {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, payloadLen int) {
		unit := mtuReader()
		t.Run(unit.name, func(t *testing.T) {
			t.Parallel()

			data := dummyPacket(uint8(payloadLen))
			unit.buffer.AddChunk(data)

			_, err := unit.reader.ReadPacket()
			if err == nil || !errors.Is(err, io.EOF) {
				t.Fatalf("%v: %v", payloadLen, err)
			}
		})
	})
}

func FuzzJumbo(f *testing.F) {
	payloadLen := mtu - ipv6.HeaderLen
	for i := 0; i < (payloadLen + ipv6.HeaderLen); i++ {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, chunkLen int) {
		for _, iterUnit := range readers() {
			unit := iterUnit
			t.Run(unit.name, func(t *testing.T) {
				t.Parallel()

				data := dummyPacket(uint8(payloadLen))
				data[5] = 0
				unit.buffer.AddChunk(data[:chunkLen])

				p, err := unit.reader.ReadPacket()
				if err == nil {
					t.Fatalf("this did not fail: %d, %d", chunkLen, len(p.Marshalled))
				}
			})
		}
	})
}

func TestInvalidHeader(t *testing.T) {
	t.Parallel()

	for _, iterUnit := range readers() {
		unit := iterUnit
		t.Run(unit.name, func(t *testing.T) {
			t.Parallel()

			payloadLen := mtu - ipv6.HeaderLen
			data := dummyPacket(uint8(payloadLen))
			data[0] = 0x40
			unit.buffer.AddChunk(data)

			_, err := unit.reader.ReadPacket()
			if err == nil {
				t.Fatal(err)
			}
		})
	}
}
