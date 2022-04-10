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

package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"eqrx.net/rungroup"
	"eqrx.net/wallhack/internal/bridge"
	"github.com/go-logr/logr"
)

func serveListener(log logr.Logger, listener net.Listener) func(context.Context) error {
	return func(ctx context.Context) error {
		group := rungroup.New(ctx)

		group.Go(func(ctx context.Context) error {
			<-ctx.Done()
			if err := listener.Close(); err != nil {
				return fmt.Errorf("close listener: %w", err)
			}

			return nil
		})

		group.Go(func(ctx context.Context) error {
			for {
				conn, err := listener.Accept()
				switch {
				case err == nil:
					group.Go(serveConn(log, conn.(*tls.Conn))) //nolint:forcetypeassert
				case errors.Is(err, net.ErrClosed):
					return nil
				default:
					return fmt.Errorf("accept: %w", err)
				}
			}
		})

		if err := group.Wait(); err != nil {
			return err
		}

		return nil
	}
}

func serveConn(log logr.Logger, conn *tls.Conn) func(context.Context) error {
	return func(ctx context.Context) error {
		group := rungroup.New(ctx)
		group.Go(func(ctx context.Context) error {
			<-ctx.Done()
			if err := conn.Close(); err != nil {
				return fmt.Errorf("closing conn: %w", err)
			}

			return nil
		})

		group.Go(func(ctx context.Context) error {
			if err := conn.HandshakeContext(ctx); err != nil {
				log.Error(err, "tls handshake failed for tunnel")

				return nil
			}

			tlsState := conn.ConnectionState()
			if len(tlsState.PeerCertificates) != 1 {
				log.Info("client did not send exactly one cert")

				return nil
			}

			commonName := tlsState.PeerCertificates[0].Subject.CommonName

			log.Info("start bridging", "cn", commonName)
			err := bridge.Connect(ctx, log, conn, commonName)
			log.Info("stop bridging", "cn", commonName)

			if err != nil {
				return fmt.Errorf("connect tun and bridge: %w", err)
			}

			return nil
		})

		if err := group.Wait(); err != nil {
			panic(fmt.Sprintf("no errors expected, got %v", err))
		}

		return nil
	}
}
