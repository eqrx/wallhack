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

package listener

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"eqrx.net/rungroup"
	"github.com/go-logr/logr"
)

func (l *Listener) pickSink(state tls.ConnectionState) chan<- net.Conn {
	switch {
	case state.NegotiatedProtocol == "wallhack":
		if state.Version != tls.VersionTLS13 {
			panic("version drop")
		}

		if len(state.PeerCertificates) < 1 {
			panic("no client auth")
		}

		return l.wallhackFrontend.conns
	case l.hasPlugin:
		return l.pluginFrontend.conns
	default:
		panic("non wallhack protocol and no plugin")
	}
}

func (l *Listener) acceptBackend(backend net.Listener, log logr.Logger) func(context.Context) error {
	return func(ctx context.Context) error {
		for {
			conn, err := backend.Accept()

			switch {
			case err == nil:
			case errors.Is(err, net.ErrClosed):
				return nil
			default:
				return fmt.Errorf("backend accept: %w", err)
			}

			tlsConn := conn.(*tls.Conn) //nolint:forcetypeassert
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				log.Error(err, "tls handshake")
			}

			sink := l.pickSink(tlsConn.ConnectionState())

			select {
			case <-ctx.Done():
				if err := conn.Close(); err != nil {
					return fmt.Errorf("close conn: %w", err)
				}

				return nil
			case sink <- conn:
			}
		}
	}
}

func (l *Listener) acceptBackends(log logr.Logger) func(context.Context) error {
	return func(ctx context.Context) error {
		backendGroup := rungroup.New(ctx)

		for i := range l.backends {
			backend := l.backends[i]

			backendGroup.Go(func(ctx context.Context) error {
				<-ctx.Done()

				if err := backend.Close(); err != nil {
					return fmt.Errorf("close backend: %w", err)
				}

				return nil
			})

			backendGroup.Go(l.acceptBackend(backend, log))
		}

		if err := backendGroup.Wait(); err != nil {
			return err
		}

		return nil
	}
}

// Listen returns a rungroup compatible method that listens on the
// configured backends an shoves connections into wallhack and plugin.
func (l *Listener) Listen(log logr.Logger) func(context.Context) error {
	return func(ctx context.Context) error {
		rootGroup := rungroup.New(ctx)
		rootGroup.Go(func(ctx context.Context) error {
			<-ctx.Done()

			close(l.wallhackFrontend.conns)
			close(l.pluginFrontend.conns)

			return nil
		})

		rootGroup.Go(l.acceptBackends(log))

		if err := rootGroup.Wait(); err != nil {
			return err
		}

		return nil
	}
}
