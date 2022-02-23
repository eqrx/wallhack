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

// Package bridge is responsible for bridging linux tuns over network connections.
package bridge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"dev.eqrx.net/rungroup"
	"github.com/go-logr/logr"
)

// Connect attaches to the linux tun named as in tunIfaceName and exchanges its packets over
// the given conn with another Connect instance. Any errors are returned. Packets transmitted
// over the conn are prepended with a uint16 indicating the full length of the following packet.
func Connect(ctx context.Context, log logr.Logger, conn net.Conn, tunIfaceName string) error {
	tun, err := newTun(tunIfaceName)
	if err != nil {
		return fmt.Errorf("setup tun: %w", err)
	}

	group := rungroup.New(ctx)

	group.Go(func(ctx context.Context) error {
		<-ctx.Done()

		if err := tun.close(); err != nil {
			return fmt.Errorf("close tun: %w", err)
		}

		return nil
	})

	group.Go(func(context.Context) error {
		reader := func() ([]byte, error) { return readIPFrame(conn) }
		transportFrames(log.WithName("tun<-bridge"), tun.writeIPFrame, reader)

		return nil
	})
	group.Go(func(context.Context) error {
		writer := func(packet []byte) error { return writeIPFrame(conn, packet) }
		transportFrames(log.WithName("bridge<-tun"), writer, tun.readIPFrame)

		return nil
	})

	if err := group.Wait(); err != nil {
		return fmt.Errorf("bridge conn and tun group: %w", err)
	}

	return nil
}

// transportFrames reads frames from src and writes them to dst. Returns on any error.
func transportFrames(log logr.Logger, writer func([]byte) error, reader func() ([]byte, error)) {
	for {
		frame, err := reader()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrClosed) {
				log.Error(err, "could not read frame")
			}

			return
		}

		if err := writer(frame); err != nil {
			if !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrClosed) {
				log.Error(err, "could not write frame")
			}

			return
		}
	}
}
