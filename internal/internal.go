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

// Package internal is responsible for bootstrapping wallhack and run it.
package internal

import (
	"context"
	"flag"
	"fmt"

	"eqrx.net/service"
	"eqrx.net/wallhack/internal/client"
	"eqrx.net/wallhack/internal/server"
	"github.com/go-logr/logr"
)

// Run wallhack.
func Run(ctx context.Context, log logr.Logger, service *service.Service) error {
	isServer := flag.Bool("server", false, "run in server mode")
	flag.Parse()

	if *isServer {
		if err := server.Run(ctx, log, service); err != nil {
			return fmt.Errorf("wallhack: %w", err)
		}

		return nil
	}

	if err := client.Run(ctx, log, service); err != nil {
		return fmt.Errorf("wallhack:: %w", err)
	}

	return nil
}
