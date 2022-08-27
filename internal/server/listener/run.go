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

func (l *Listener) acceptBackend(ctx context.Context, backend net.Listener, log logr.Logger) error {
	for {
		conn, err := backend.Accept()

		switch {
		case err == nil:
		case errors.Is(err, net.ErrClosed):
			return nil
		default:
			return fmt.Errorf("accept backend: %w", err)
		}

		tlsConn := conn.(*tls.Conn)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			log.Error(err, "tls handshake")

			continue
		}

		sink := l.pickSink(tlsConn.ConnectionState())

		select {
		case <-ctx.Done():
			if err := conn.Close(); err != nil {
				return fmt.Errorf("accept backend: %w", err)
			}

			return nil
		case sink <- conn:
		}
	}
}

func (l *Listener) acceptBackends(ctx context.Context, log logr.Logger) error {
	backendGroup := rungroup.New(ctx)

	for i := range l.backends {
		backend := l.backends[i]

		backendGroup.Go(func(ctx context.Context) error {
			<-ctx.Done()

			if err := backend.Close(); err != nil {
				return fmt.Errorf("close: %w", err)
			}

			return nil
		})

		backendGroup.Go(func(ctx context.Context) error { return l.acceptBackend(ctx, backend, log) })
	}

	if err := backendGroup.Wait(); err != nil {
		return fmt.Errorf("accept backends: %w", err)
	}

	return nil
}

// Listen returns a rungroup compatible method that listens on the
// configured backends an shoves connections into wallhack and plugin.
func (l *Listener) Listen(ctx context.Context, log logr.Logger) error {
	rootGroup := rungroup.New(ctx)
	rootGroup.Go(func(ctx context.Context) error {
		<-ctx.Done()

		close(l.wallhackFrontend.conns)
		close(l.pluginFrontend.conns)

		return nil
	})

	rootGroup.Go(func(ctx context.Context) error { return l.acceptBackends(ctx, log) })

	if err := rootGroup.Wait(); err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	return nil
}
