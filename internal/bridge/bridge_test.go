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

package bridge_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"testing"

	"eqrx.net/rungroup"
	"eqrx.net/wallhack/internal/bridge"
	"eqrx.net/wallhack/internal/packet"
)

type (
	buf struct {
		write    []*packet.Packet
		read     []*packet.Packet
		closed   int
		closeErr error
		emptyErr error
		writeErr error
	}
)

func (b *buf) Close() error {
	b.closed++

	return b.closeErr
}

func (b *buf) ReadPacket() (*packet.Packet, error) {
	if len(b.read) == 0 {
		return nil, b.emptyErr
	}

	d := b.read[0]
	b.read = b.read[1:]

	return d, nil
}

func (b *buf) WritePacket(packet *packet.Packet) error {
	b.write = append(b.write, packet)

	return b.writeErr
}

func TestEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bufA := &buf{[]*packet.Packet{}, []*packet.Packet{}, 0, nil, io.EOF, nil}
	bufB := &buf{[]*packet.Packet{}, []*packet.Packet{}, 0, nil, io.EOF, nil}
	err := bridge.Bridge(ctx, bufA, bufB)

	if err == nil {
		t.Fatal()
	}

	var gErr *rungroup.Error

	if !errors.As(err, &gErr) {
		t.Fatal()
	}

	if len(gErr.Errs) != 2 {
		t.Fatal(len(gErr.Errs))
	}

	if !errors.Is(gErr.Errs[0], io.EOF) {
		t.Fatal()
	}

	if !errors.Is(gErr.Errs[1], io.EOF) {
		t.Fatal()
	}

	if bufA.closed != 1 {
		t.Fatal()
	}

	if bufB.closed != 1 {
		t.Fatal()
	}
}

func TestDuplex(t *testing.T) {
	t.Parallel()

	payloadA := []byte{1, 2, 3}
	payloadB := []byte{4, 5, 6}
	payloadC := []byte{7, 8, 9}

	ctx := context.Background()
	bufA := &buf{[]*packet.Packet{}, []*packet.Packet{{Marshalled: payloadA}, {Marshalled: payloadB}}, 0, nil, io.EOF, nil}
	bufB := &buf{[]*packet.Packet{}, []*packet.Packet{{Marshalled: payloadC}}, 0, nil, io.EOF, nil}
	_ = bridge.Bridge(ctx, bufA, bufB)

	if len(bufA.read) != 0 {
		t.Fatal(len(bufA.read))
	}

	if len(bufB.read) != 0 {
		t.Fatal(len(bufB.read))
	}

	if len(bufA.write) != 1 {
		t.Fatal(len(bufA.write))
	}

	if len(bufB.write) != 2 {
		t.Fatal(len(bufB.write))
	}
}

func TestWriteErr(t *testing.T) {
	t.Parallel()

	payloadA := []byte{1, 2, 3}
	payloadB := []byte{4, 5, 6}
	payloadC := []byte{7, 8, 9}

	errA := fs.ErrExist
	errB := fs.ErrInvalid
	errC := fs.ErrPermission
	errD := fs.ErrNotExist

	ctx := context.Background()
	bufA := &buf{[]*packet.Packet{}, []*packet.Packet{{Marshalled: payloadA}, {Marshalled: payloadB}}, 0, errC, nil, errA}
	bufB := &buf{[]*packet.Packet{}, []*packet.Packet{{Marshalled: payloadC}}, 0, errD, nil, errB}
	err := bridge.Bridge(ctx, bufA, bufB)

	if err == nil {
		t.Fatal()
	}

	var gErr *rungroup.Error

	if !errors.As(err, &gErr) {
		t.Fatal()
	}

	errsACount := 0
	errsBCount := 0
	errsCCount := 0
	errsDCount := 0

	for _, err = range gErr.Errs {
		switch {
		case errors.Is(err, errA):
			errsACount++
		case errors.Is(err, errB):
			errsBCount++
		case errors.Is(err, errC):
			errsCCount++
		case errors.Is(err, errD):
			errsDCount++
		default:
			t.Fatal(err)
		}
	}

	if errsACount != 1 || errsBCount != 1 || errsCCount != 1 || errsDCount != 1 {
		t.Fatalf("%d %d %d %d", errsACount, errsBCount, errsCCount, errsDCount)
	}
}
