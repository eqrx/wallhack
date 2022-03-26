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

// Package listener handles TLS ALPN routing magic.
package listener

import (
	"crypto/tls"
	"net"
)

// Listener that sources connections from all given backends,
// and routes TLS connection to wallhack and the plugin according to the ALPN
// field of the client.
type Listener struct {
	backends         []net.Listener
	wallhackFrontend frontend
	pluginFrontend   frontend
	hasPlugin        bool
}

// WallhackListener returns the frontend listener for wallhack.
func (l *Listener) WallhackListener() net.Listener { return l.wallhackFrontend }

// PluginListener returns the frontend listener for the configured plugin.
func (l *Listener) PluginListener() net.Listener { return l.pluginFrontend }

// New creates a new listener that sources connections from all given backends,
// and routes TLS connection to wallhack and the plugin according to the ALPN
// field of the client.
func New(backends []net.Listener, wallhackCfg, pluginCfg *tls.Config) *Listener {
	listener := &Listener{
		make([]net.Listener, 0, len(backends)),
		frontend{make(chan net.Conn), frontendAddr{"frontend for wallhack alpns"}},
		frontend{make(chan net.Conn), frontendAddr{"frontend for plugin"}},
		pluginCfg != nil,
	}

	if listener.hasPlugin {
		wallhackCfg.GetConfigForClient = func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
			if chi.SupportedProtos != nil {
				for _, proto := range chi.SupportedProtos {
					if proto == "wallhack" {
						return nil, nil //nolint: nilnil
					}
				}
			}

			return pluginCfg, nil
		}
	}

	for _, l := range backends {
		listener.backends = append(listener.backends, tls.NewListener(l, wallhackCfg))
	}

	return listener
}
