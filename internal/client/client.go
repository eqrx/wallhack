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

// Package client provides wallhack client mode. This means it attaches to a tun interface and dials to a wallhack
// server. On success all packages from the tun and written to the connection and vice versa.
package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"dev.eqrx.net/rungroup"
	"dev.eqrx.net/wallhack/internal/env"
	"dev.eqrx.net/wallhack/internal/io"
	"github.com/go-logr/logr"
)

const (
	// backOffDelay is the delay between connection attempts.
	backOffDelay = 10 * time.Second
	// tunIfaceName is the name of the tun interface to use for wallhack.
	tunIfaceName = "wallhack"
)

// Run this instance in client mode.
func Run(ctx context.Context, log logr.Logger) error {
	serverName, err := env.Lookup(env.ServerAddr)
	if err != nil {
		return fmt.Errorf("%w: net.Dial compatible address of the server", err)
	}

	tlsConfig, err := env.CreateTLSConfig()
	if err != nil {
		return fmt.Errorf("create tls dialer: %w", err)
	}

	dialer := &tls.Dialer{Config: tlsConfig}

	return dial(ctx, log, dialer, serverName)
}

// dial attempts to dial with dialer to the server behind serverName until canceled.
// On success a local tun is opened and all packets arriving on it will be streamed over conn
// and vice versa. Returns any unexpected errors.
func dial(ctx context.Context, log logr.Logger, dialer *tls.Dialer, serverName string) error {
	for {
		conn, err := dialer.DialContext(ctx, "tcp4", serverName)

		switch {
		case ctx.Err() != nil:
			return nil //nolint:nilerr // net package throws unexported net.errCanceled instead of wrapping context errs.
		case err != nil:
			log.Error(err, "could not open tunnel, backing off")

			delay := time.NewTimer(backOffDelay)
			select {
			case <-ctx.Done():
				return nil
			case <-delay.C:
			}

			continue
		}

		group := rungroup.New(ctx)

		group.Go(func(ctx context.Context) error {
			<-ctx.Done()

			if err := conn.Close(); err != nil {
				log.Error(err, "could not close connection")
			}

			return nil
		})

		group.Go(func(ctx context.Context) error {
			if err := io.Connect(ctx, log, conn, tunIfaceName); err != nil {
				return fmt.Errorf("connect tun and bridge: %w", err)
			}

			return nil
		})

		if err := group.Wait(); err != nil {
			return fmt.Errorf("conn group: %w", err)
		}
	}
}
