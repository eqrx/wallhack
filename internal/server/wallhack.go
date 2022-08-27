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
	"sync"

	"eqrx.net/rungroup"
	"eqrx.net/service"
	"eqrx.net/wallhack/internal/bridge"
	"eqrx.net/wallhack/internal/packet"
	"eqrx.net/wallhack/internal/tun"
	"github.com/go-logr/logr"
)

func accept(ctx context.Context, log logr.Logger, listener net.Listener) error {
	locker := sync.Mutex{}
	bridges := map[string]context.CancelFunc{}

	group := rungroup.New(ctx)

	group.Go(func(ctx context.Context) error {
		<-ctx.Done()
		if err := listener.Close(); err != nil {
			return fmt.Errorf("close: %w", err)
		}

		return nil
	})

	group.Go(func(ctx context.Context) error {
		_ = service.Instance().MarkStatus("listening")
		for {
			conn, err := listener.Accept()
			switch {
			case err == nil:
				group.Go(func(ctx context.Context) error { return newConn(ctx, log, conn.(*tls.Conn), &locker, bridges) }, rungroup.NoCancelOnSuccess)
			case errors.Is(err, net.ErrClosed):
				return nil
			default:
				return fmt.Errorf("accept: %w", err)
			}
		}
	})

	if err := group.Wait(); err != nil {
		return fmt.Errorf("bridger: %w", err)
	}

	return nil
}

func newConn(ctx context.Context, log logr.Logger, conn *tls.Conn, locker sync.Locker, bridges map[string]context.CancelFunc) error {
	log = log.WithName(conn.RemoteAddr().String())
	if err := conn.HandshakeContext(ctx); err != nil {
		log.Error(err, "tls handshake")

		return nil
	}

	tlsState := conn.ConnectionState()
	if len(tlsState.PeerCertificates) != 1 {
		log.Info("client did not send exactly one cert")

		return nil
	}

	commonName := tlsState.PeerCertificates[0].Subject.CommonName
	log = log.WithName(commonName)

	tun, err := tun.New(commonName)
	if err != nil {
		return fmt.Errorf("new bridge: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)

	locker.Lock()

	oldCancel, ok := bridges[commonName]
	if ok {
		oldCancel()
	}

	bridges[commonName] = cancel

	locker.Unlock()

	log.Info("start bridging")

	c := packet.NewReadWriteCloser(conn, packet.NewStreamReader(conn))
	t := packet.NewReadWriteCloser(tun, packet.NewMTUReader(tun))

	if err := bridge.Bridge(ctx, c, t); err != nil {
		log.Error(err, "serving conn")
	}

	log.Info("stop bridging")

	return nil
}
