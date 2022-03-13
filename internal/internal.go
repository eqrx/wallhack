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

	"eqrx.net/wallhack/internal/client"
	"eqrx.net/wallhack/internal/credentials"
	"eqrx.net/wallhack/internal/server"
	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
)

// Run wallhack.
func Run(ctx context.Context, log logr.Logger) error {
	isServer := flag.Bool("server", false, "run in server mode")
	flag.Parse()

	credentialBytes, err := credentials.LoadBytes()
	if err != nil {
		return fmt.Errorf("could not load credentials: %w", err)
	}

	if *isServer {
		var credentials credentials.Server
		if err := yaml.Unmarshal(credentialBytes, &credentials); err != nil {
			return fmt.Errorf("could not unmarshal credentials: %w", err)
		}

		if err := server.Run(ctx, log.WithName("server"), credentials); err != nil {
			return fmt.Errorf("server run failed: %w", err)
		}

		return nil
	}

	var credentials credentials.Client
	if err := yaml.Unmarshal(credentialBytes, &credentials); err != nil {
		return fmt.Errorf("could not unmarshal credentials: %w", err)
	}

	if err := client.Run(ctx, log.WithName("client"), credentials); err != nil {
		return fmt.Errorf("client  failed: %w", err)
	}

	return nil
}
