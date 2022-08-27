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

// Package client provides wallhack client mode. This means it attaches to a tun interface and dials to a wallhack
// server. On success all packages from the tun and written to the connection and vice versa.
package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"eqrx.net/service"
	"eqrx.net/wallhack/internal/bridge"
	"eqrx.net/wallhack/internal/packet"
	"eqrx.net/wallhack/internal/tun"
	"github.com/go-logr/logr"
)

const (
	// backOffDelay is the delay between connection attempts.
	backOffDelay = 10 * time.Second
	// tunIfaceName is the name of the tun interface to use for wallhack.
	tunIfaceName = "wallhack"
	// ServerEnvName is the name of the environment file containing the wallhack server address to connect to.
	ServerEnvName = "WALLHACK_SERVER"
)

// Client represents the credentials for running in client mode.
type Client struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

// TLSConf generates the TLS configuration for read credentials. It can
// be used to connect to a wallhack server.
func (c Client) tlsConf() (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(c.Cert), []byte(c.Key))
	if err != nil {
		return nil, fmt.Errorf("tls config: parse keys: %w", err)
	}

	config := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS13,
		NextProtos:               []string{"wallhack"},
	}

	return config, nil
}

// Run this instance in client mode.
func (c Client) Run(ctx context.Context, log logr.Logger) error {
	serverAddr, _ := os.LookupEnv(ServerEnvName)

	if _, _, err := net.SplitHostPort(serverAddr); err != nil {
		return fmt.Errorf("client: %w", err)
	}

	tlsConfig, err := c.tlsConf()
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}

	dialer := &tls.Dialer{Config: tlsConfig}

	_ = service.Instance().MarkReady()
	defer func() { _ = service.Instance().MarkStopping() }()

	return dial(ctx, log, dialer, serverAddr)
}

// dial attempts to dial with dialer to the server behind serverName until canceled.
// On success a local tun is opened and all packets arriving on it will be streamed over conn
// and vice versa. Returns any unexpected errors.
func dial(ctx context.Context, log logr.Logger, dialer *tls.Dialer, serverName string) error {
	for {
		_ = service.Instance().MarkStatus("dialing to " + serverName)
		conn, err := dialer.DialContext(ctx, "tcp4", serverName)

		switch {
		case err == nil:
		case errors.Is(err, ctx.Err()):
			return fmt.Errorf("dial: %w", err)
		case err != nil:
			log.Error(err, "could not open tunnel, backing off")
			_ = service.Instance().MarkStatus("backing off from " + serverName + ": " + err.Error())

			delay := time.NewTimer(backOffDelay)
			select {
			case <-ctx.Done():
				return fmt.Errorf("dial: backoff: %w", ctx.Err())
			case <-delay.C:
			}

			continue
		}

		tun, err := tun.New(tunIfaceName)
		if err != nil {
			return fmt.Errorf("dial: %w", err)
		}

		_ = service.Instance().MarkStatus("streaming to " + serverName)

		c := packet.NewReadWriteCloser(conn, packet.NewStreamReader(conn))
		t := packet.NewReadWriteCloser(tun, packet.NewMTUReader(tun))

		if err := bridge.Bridge(ctx, c, t); err != nil {
			log.Error(err, "transport")
		}
	}
}
