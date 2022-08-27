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

// Package service handles interfacing with systemd.
package service

import (
	"fmt"
	"net"
	"os"

	"eqrx.net/journalr"
	"eqrx.net/service/socket"
	"github.com/go-logr/logr"
)

const (
	credDirEnvName    = "CREDENTIALS_DIRECTORY"
	stateDirEnvName   = "STATE_DIRECTORY"
	runtimeDirEnvName = "RUNTIME_DIRECTORY"
)

// Service allows interfacing with systemd.
type Service struct {
	notify     *net.UnixConn
	listeners  []net.Listener
	journal    *journalr.Sink
	stateDir   string
	credsDir   string
	runtimeDir string
}

// Journal returns a [logr.Logger] that writes structured logs to the systemd journal.
func (s Service) Journal() logr.Logger { return logr.New(s.journal) }

// Listeners returns listeners passed by systemd via socket activation.
func (s Service) Listeners() []net.Listener {
	if len(s.listeners) == 0 {
		panic("no listeners passed by systemd")
	}

	return s.listeners
}

// StateDirectory is the state directory set by systemd.
// Panics if not set by systemd.
func (s Service) StateDirectory() string {
	if s.stateDir == "" {
		panic("state dir not set")
	}

	return s.stateDir
}

// RuntimeDirectory is the runtime directory set by systemd.
// Panics if not set by systemd.
func (s Service) RuntimeDirectory() string {
	if s.runtimeDir == "" {
		panic("runtime dir not set")
	}

	return s.runtimeDir
}

// New creates a new Service instance to interface with systemd.
func New() (*Service, error) {
	notify, err := newNotifySocket()
	if err != nil {
		return nil, fmt.Errorf("setup service: %w", err)
	}

	journalSink, err := journalr.NewSink()
	if err != nil {
		return nil, fmt.Errorf("setup service: %w", err)
	}

	listeners, err := socket.Listeners()
	if err != nil {
		return nil, fmt.Errorf("setup service: %w", err)
	}

	stateDir := os.Getenv(stateDirEnvName)
	credsDir := os.Getenv(credDirEnvName)
	runtimeDir := os.Getenv(runtimeDirEnvName)

	return &Service{notify, listeners, journalSink, stateDir, credsDir, runtimeDir}, nil
}
