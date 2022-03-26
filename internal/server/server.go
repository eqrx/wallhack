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

var errNoCA = errors.New("no CA given")

// Server represents the credentials for running in server mode.
type Server struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
	CA   string `yaml:"ca"`
}

func (s Server) tlsConf() (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(s.Cert), []byte(s.Key))
	if err != nil {
		return nil, fmt.Errorf("parse cert: %w", err)
	}

	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM([]byte(s.CA)) {
		return nil, errNoCA
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
func (s Server) Run(ctx context.Context, log logr.Logger, service *service.Service) error {
	tlsConfig, err := s.tlsConf()
	if err != nil {
		return fmt.Errorf("could not setup tls: %w", err)
	}

	listeners, err := service.Listeners()
	if err != nil {
		return fmt.Errorf("no listeners: %w", err)
	}

	plugin, err := loadPlugin()
	if err != nil {
		return err
	}

	var pluginTLSConfig *tls.Config
	if plugin != nil {
		pluginTLSConfig = plugin.TLSConfig()
		pluginTLSConfig.Certificates = []tls.Certificate{tlsConfig.Certificates[0]}
	}

	group := rungroup.New(ctx)

	comboListener := listener.New(listeners, tlsConfig, pluginTLSConfig)

	group.Go(comboListener.Listen(log))

	group.Go(serveListener(log, comboListener.WallhackListener()))

	if plugin != nil {
		group.Go(func(ctx context.Context) error {
			if err := plugin.Listen(ctx, comboListener.PluginListener()); err != nil {
				return fmt.Errorf("plugin serve: %w", err)
			}

			return nil
		})
	}

	_ = service.MarkReady()
	_ = service.MarkStatus("listening")

	defer func() { _ = service.MarkStopping() }()

	if err := group.Wait(); err != nil {
		return fmt.Errorf("listening group failed: %w", err)
	}

	return nil
}
