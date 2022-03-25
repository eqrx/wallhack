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

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"eqrx.net/service"
	"eqrx.net/wallhack/internal"
	"golang.org/x/sys/unix"
)

func main() {
	service, err := service.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "systemd: %v", err)
		os.Exit(1)
	}

	log := service.Journal()

	ctx, cancel := signal.NotifyContext(context.Background(), unix.SIGTERM, unix.SIGINT)

	err = internal.Run(ctx, log, service)

	cancel()

	if err != nil {
		log.Error(err, "main")

		os.Exit(1)
	}

	os.Exit(0)
}
