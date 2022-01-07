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

// Package internal is responsible for bootstrapping wallhack and run it.
package internal

import (
	"context"
	"errors"
	"fmt"

	"dev.eqrx.net/wallhack/internal/client"
	"dev.eqrx.net/wallhack/internal/env"
	"dev.eqrx.net/wallhack/internal/server"
	"github.com/go-logr/logr"
)

// errUnknownMode indicates that wallhack was started with an unknown server mode.
var errUnknownMode = errors.New("unknown operating mode")

// Run wallhack.
func Run(ctx context.Context, log logr.Logger) error {
	mode, err := env.Lookup(env.Mode)
	if err != nil {
		return fmt.Errorf("%w: operating mode of this instance. May be client or server", err)
	}

	switch mode {
	case "server":
		if err := server.Run(ctx, log.WithName("server")); err != nil {
			return fmt.Errorf("server run failed: %w", err)
		}
	case "client":
		if err := client.Run(ctx, log.WithName("client")); err != nil {
			return fmt.Errorf("client  failed: %w", err)
		}
	default:
		return fmt.Errorf("%w set by %s: %s", errUnknownMode, env.Mode, mode)
	}

	return nil
}
