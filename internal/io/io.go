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

// Package io is responsible for handling linux tuns and serialize packets over network connections.
package io

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/eqrx/rungroup"
	"github.com/go-logr/logr"
)

// ipFrameReader allows to read full IP frames.
type ipFrameReader interface {
	// readIPFrame returns a complete IP frame as bytes and any io error encountered.
	readIPFrame() ([]byte, error)
}

// ipFrameWriter allows to write IP frames.
type ipFrameWriter interface {
	// writeIPFrame writes a complete IP frame as bytes and returns any io error encountered.
	writeIPFrame([]byte) error
}

// Connect attaches to the linux tun named as in tunIfaceName and exchanges its packets over
// the given conn with another Connect instance. Any errors are returned. Packets transmitted
// overthe conn are prepended with a uint16 indicating the full length of the following packet.
func Connect(ctx context.Context, log logr.Logger, conn net.Conn, tunIfaceName string) error {
	tun, err := newTun(tunIfaceName)
	if err != nil {
		return fmt.Errorf("could not setup tun: %w", err)
	}

	bridge := &bridge{conn}

	group := rungroup.New(ctx)

	group.Go(func(ctx context.Context) error {
		<-ctx.Done()

		if err := tun.Close(); err != nil {
			return fmt.Errorf("could not close tun: %w", err)
		}

		return nil
	})

	group.Go(func(context.Context) error {
		transportFrames(log.WithName("tun<-bridge"), tun, bridge)

		return nil
	})
	group.Go(func(context.Context) error {
		transportFrames(log.WithName("bridge<-tun"), bridge, tun)

		return nil
	})

	if err := group.Wait(); err != nil {
		return fmt.Errorf("bridge conn and tun group failed: %w", err)
	}

	return nil
}

// transportFrames reads frames from src and writes them to dst. Returns on any error.
func transportFrames(log logr.Logger, dst ipFrameWriter, src ipFrameReader) {
	for {
		frame, err := src.readIPFrame()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrClosed) {
				log.Error(err, "could not read frame")
			}

			return
		}

		if err := dst.writeIPFrame(frame); err != nil {
			if !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrClosed) {
				log.Error(err, "could not write frame")
			}

			return
		}
	}
}
