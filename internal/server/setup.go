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

package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"github.com/coreos/go-systemd/v22/activation"
	"github.com/eqrx/wallhack/internal/env"
)

// errSystemd indicates that interfacing with systemd did not work out quite well.
var errSystemd = errors.New("systemd interfacing failed")

// getListener returns the listener passed by systemd.
func getListener(context.Context) (net.Listener, error) {
	listeners, err := activation.Listeners()
	if err != nil {
		return nil, fmt.Errorf("could not get listeners from systemd: %w", err)
	}

	if len(listeners) < 1 {
		return nil, fmt.Errorf("%w: listeners too small", errSystemd)
	}

	if listeners[0] == nil {
		return nil, fmt.Errorf("%w: first file is not listener", errSystemd)
	}

	listener := listeners[0]

	tlsConfig, err := env.CreateTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("could not setup tls: %w", err)
	}

	return tls.NewListener(listener, tlsConfig), nil
}
