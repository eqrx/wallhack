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
	"crypto/x509"
	"errors"
	"fmt"

	"eqrx.net/rungroup"
	"eqrx.net/service"
	"eqrx.net/wallhack/internal/server/listener"
	"github.com/go-logr/logr"
)

var errCaMissing = errors.New("no CA configured")

func tlsConf(service *service.Service) (*tls.Config, error) {
	certData, err := service.LoadCred("cert")
	if err != nil {
		return nil, fmt.Errorf("tls conf: %w", err)
	}

	keyData, err := service.LoadCred("key")
	if err != nil {
		return nil, fmt.Errorf("tls conf: %w", err)
	}

	caData, err := service.LoadCred("ca")
	if err != nil {
		return nil, fmt.Errorf("tls conf: %w", err)
	}

	cert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return nil, fmt.Errorf("tls conf: load certs: %w", err)
	}

	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(caData) {
		return nil, errCaMissing
	}

	config := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                clientCAs,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS13,
		NextProtos:               []string{"wallhack"},
		ClientAuth:               tls.RequireAndVerifyClientCert,
	}

	return config, nil
}

// Run wallhack in server mode.
func Run(ctx context.Context, log logr.Logger, service *service.Service) error {
	tlsConfig, err := tlsConf(service)
	if err != nil {
		return fmt.Errorf("server: %w", err)
	}

	listeners := service.Listeners()

	plugin, err := loadPlugin()
	if err != nil {
		return err
	}

	var pluginTLSConfig *tls.Config
	if plugin != nil {
		pluginTLSConfig = plugin.TLSConfig()
		pluginTLSConfig.Certificates = []tls.Certificate{tlsConfig.Certificates[0]}
	}

	comboListener := listener.New(listeners, tlsConfig, pluginTLSConfig)

	group := rungroup.New(ctx)
	group.Go(func(ctx context.Context) error {
		if err := comboListener.Listen(ctx, log); err != nil {
			return fmt.Errorf("combo listener: %w", err)
		}

		return nil
	})
	group.Go(func(ctx context.Context) error { return accept(ctx, log, service, comboListener.WallhackListener()) })

	if plugin != nil {
		group.Go(func(ctx context.Context) error {
			if err := plugin.Listen(ctx, comboListener.PluginListener()); err != nil {
				return fmt.Errorf("plugin listen: %w", err)
			}

			return nil
		})
	}

	group.Go(service.RunNotify)

	if err := group.Wait(); err != nil {
		return fmt.Errorf("server: %w", err)
	}

	return nil
}
