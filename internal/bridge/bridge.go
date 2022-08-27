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

// Package bridge is responsible for bridging linux tuns over network connections.
package bridge

import (
	"context"
	"fmt"
	"io"

	"eqrx.net/rungroup"
	"eqrx.net/wallhack/internal/packet"
)

type (
	// Reader allows reading IP packets.
	Reader interface {
		ReadPacket() (*packet.Packet, error)
	}

	// Writer allows writing IP packets.
	Writer interface {
		WritePacket(*packet.Packet) error
	}

	// ReadWriteCloser allows reading and writing IP packets and implements [io.Closer].
	ReadWriteCloser interface {
		io.Closer
		Reader
		Writer
	}
)

// Bridge given streams left and right together by reading IPpackets from both and writing
// them to the other.
func Bridge(ctx context.Context, left, right ReadWriteCloser) error {
	group := rungroup.New(ctx)

	group.Go(func(ctx context.Context) error { return closer(ctx, left) })
	group.Go(func(ctx context.Context) error { return closer(ctx, right) })
	group.Go(func(_ context.Context) error { return simplex(left, right) })
	group.Go(func(_ context.Context) error { return simplex(right, left) })

	return fmt.Errorf("bridge: %w", group.Wait())
}

func closer(ctx context.Context, c io.Closer) error {
	<-ctx.Done()

	if err := c.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return nil
}

func simplex(dst Writer, src Reader) error {
	for {
		packet, err := src.ReadPacket()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		if err = dst.WritePacket(packet); err != nil {
			return fmt.Errorf("write: %w", err)
		}
	}
}
