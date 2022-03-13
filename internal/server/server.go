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
	"net/http"
	"time"

	"eqrx.net/rungroup"
	"eqrx.net/wallhack/internal/credentials"
	"github.com/coreos/go-systemd/v22/activation"
	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/go-logr/logr"
)

// errSystemd indicates that interfacing with systemd did not work out quite well.
var errSystemd = errors.New("systemd interfacing failed")

const shutdownTimeout = 3 * time.Second

// Run wallhack in server mode.
func Run(ctx context.Context, log logr.Logger, credentials credentials.Server) error {
	tlsConfig, err := credentials.TLSConf()
	if err != nil {
		return fmt.Errorf("could not setup tls: %w", err)
	}

	group := rungroup.New(ctx)
	if err := startServers(log, group, tlsConfig); err != nil { //nolint:contextcheck
		return err
	}

	if _, err := daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
		return fmt.Errorf("systemd notify: %w", err)
	}

	_, _ = daemon.SdNotify(false, "STATUS=listening")

	defer func() { _, _ = daemon.SdNotify(false, daemon.SdNotifyStopping) }()

	if err := group.Wait(); err != nil {
		return fmt.Errorf("listening group failed: %w", err)
	}

	return nil
}

func startServers(log logr.Logger, group *rungroup.Group, tlsConfig *tls.Config) error {
	setupTLS, setupServer, err := loadHTTPPlugin()
	if err != nil {
		return err
	}

	setupTLS(tlsConfig)

	listeners, err := activation.Listeners()
	if err != nil {
		return fmt.Errorf("could not get listeners from systemd: %w", err)
	}

	tlsListeners := []net.Listener{}
	servers := []*http.Server{}

	for _, l := range listeners {
		if l == nil {
			return fmt.Errorf("%w: file passed is not listener", errSystemd)
		}

		tlsListeners = append(tlsListeners, tls.NewListener(l, tlsConfig))

		server := &http.Server{
			TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){
				credentials.ALPNWallhack: func(server *http.Server, conn *tls.Conn, _ http.Handler) {
					handleWallhackConn(server.BaseContext(nil), log, conn)
				},
			},
			Handler: http.NewServeMux(),
		}

		if err := setupServer(server); err != nil {
			return fmt.Errorf("setup http server by plugin: %w", err)
		}

		servers = append(servers, server)
	}

	for i := range servers {
		startServer(servers[i], tlsListeners[i], group)
	}

	return nil
}

func startServer(server *http.Server, listener net.Listener, group *rungroup.Group) {
	group.Go(func(ctx context.Context) error {
		server.BaseContext = func(l net.Listener) context.Context { return ctx }

		group.Go(func(ctx context.Context) error {
			<-ctx.Done()

			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)

			err := server.Shutdown(shutdownCtx) //nolint:contextcheck
			shutdownCancel()

			if err == nil || errors.Is(err, http.ErrServerClosed) {
				return nil
			}

			return fmt.Errorf("close listener: %w", err)
		})
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve http: %w", err)
		}

		return nil
	})
}
