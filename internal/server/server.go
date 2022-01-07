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

// Package server is running wallhack in server mode. It listens on a listener given by systemd and attempts to attach
// tun interfaces identified by the certificate of connecting clients. It then streams frames between the connection
// and the tun.
package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"dev.eqrx.net/rungroup"
	"dev.eqrx.net/wallhack/internal/io"
	"github.com/go-logr/logr"
)

// Run wallhack in server mode.
func Run(ctx context.Context, log logr.Logger) error {
	listener, err := getListener(ctx)
	if err != nil {
		return fmt.Errorf("could not get tunnel listener: %w", err)
	}

	group := rungroup.New(ctx)
	group.Go(func(ctx context.Context) error {
		<-ctx.Done()
		if err := listener.Close(); err != nil {
			return fmt.Errorf("could not close listener: %w", err)
		}

		return nil
	})
	group.Go(func(ctx context.Context) error {
		for {
			from, err := listener.Accept()

			switch {
			case errors.Is(err, net.ErrClosed):
				return nil
			case err != nil:
				return fmt.Errorf("failed to accept new connection: %w", err)
			}

			tlsFrom, ok := from.(*tls.Conn)
			if !ok {
				panic("connection received from tls listener is non TLS")
			}

			group.Go(func(ctx context.Context) error {
				handleConn(ctx, log, tlsFrom)

				return nil
			}, rungroup.NoCancelOnSuccess)
		}
	})

	if err := group.Wait(); err != nil {
		return fmt.Errorf("listening group failed: %w", err)
	}

	return nil
}

// handleConn that was accepted by the listener.
func handleConn(ctx context.Context, log logr.Logger, conn *tls.Conn) {
	group := rungroup.New(ctx)
	group.Go(func(ctx context.Context) error {
		<-ctx.Done()
		if err := conn.Close(); err != nil {
			log.Error(err, "tls handshake failed for tunnel")
		}

		return nil
	})

	group.Go(func(ctx context.Context) error {
		if err := conn.HandshakeContext(ctx); err != nil {
			log.Error(err, "tls handshake failed for tunnel")

			return nil
		}

		commonName := conn.ConnectionState().PeerCertificates[0].Subject.CommonName
		if err := io.Connect(ctx, log, conn, commonName); err != nil {
			return fmt.Errorf("could not connection tun and bridge: %w", err)
		}

		return nil
	})

	if err := group.Wait(); err != nil {
		panic(fmt.Sprintf("no errors expected, got %v", err))
	}
}
