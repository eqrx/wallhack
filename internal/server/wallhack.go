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
	"fmt"

	"dev.eqrx.net/rungroup"
	"dev.eqrx.net/wallhack/internal/bridge"
	"github.com/go-logr/logr"
)

// handleWallhackConn that was accepted by the listener.
func handleWallhackConn(ctx context.Context, log logr.Logger, conn *tls.Conn) {
	tlsState := conn.ConnectionState()
	if len(tlsState.PeerCertificates) != 1 {
		log.Info("client did not send exactly one cert")

		return
	}

	commonName := tlsState.PeerCertificates[0].Subject.CommonName

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
}
