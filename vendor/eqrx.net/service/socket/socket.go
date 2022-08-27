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

// Package socket handles the socket activation of systemd.
package socket

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	// listenerPIDEnvName is the env name that indicates which PID gets systemd sockets.
	listenerPIDEnvName = "LISTEN_PID"
	// listenerPIDEnvName is the env name that indicates how many sockets were passed by systemd.
	listenerCountEnvName = "LISTEN_FDS"
	// listenerPIDEnvName is the env name that indicates the named of sockets passed by systemd.
	listenerNamesEnvName = "LISTEN_FDNAMES"
	// listenFdsStart indicates which fd index is the first socket passed by systemd.
	listenFdsStart = 3
)

// ErrEnvMissing indicates a required environment variable is not set.
var ErrEnvMissing = errors.New("environment variable missing")

// Listeners returnes listeners passed by systemd.
func Listeners() ([]net.Listener, error) {
	files, err := files()
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0, len(files))

	for {
		if len(files) == 0 {
			return listeners, nil
		}

		file := files[0]
		files = files[1:]

		var listener net.Listener
		listener, err = net.FileListener(file)

		if err != nil {
			err = fmt.Errorf("convert file to listener: %w", err)

			break
		}

		listeners = append(listeners, listener)

		if err = file.Close(); err != nil {
			err = fmt.Errorf("close listener file: %w", err)

			break
		}
	}

	for _, f := range files {
		_ = f.Close()
	}

	for _, l := range listeners {
		_ = l.Close()
	}

	return nil, err
}

// files returns the files passed by systemd.
func files() ([]*os.File, error) {
	fileCount, err := fileCount()
	if err != nil {
		return nil, err
	}

	if fileCount == 0 {
		return []*os.File{}, nil
	}

	if err := os.Unsetenv(listenerPIDEnvName); err != nil {
		return nil, fmt.Errorf("unset listen pid: %w", err)
	}

	if err := os.Unsetenv(listenerCountEnvName); err != nil {
		return nil, fmt.Errorf("unset listener pid: %w", err)
	}

	listenerNamesStr, listenerNamesSet := os.LookupEnv(listenerNamesEnvName)
	if !listenerNamesSet {
		return []*os.File{}, fmt.Errorf("%w: %s", ErrEnvMissing, listenerNamesEnvName)
	}

	if err := os.Unsetenv(listenerNamesEnvName); err != nil {
		return nil, fmt.Errorf("unset listener names: %w", err)
	}

	listenerNames := strings.Split(listenerNamesStr, ":")

	listeners := make([]*os.File, 0, fileCount)

	for fd := listenFdsStart; fd < listenFdsStart+fileCount; fd++ {
		unix.CloseOnExec(fd)
		listeners = append(listeners, os.NewFile(uintptr(fd), listenerNames[fd-listenFdsStart]))
	}

	return listeners, nil
}

// fileCount returns the number of files passed by systemd.
func fileCount() (int, error) {
	pidStr, pidSet := os.LookupEnv(listenerPIDEnvName)
	if !pidSet {
		return 0, nil
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid listen pid: %w", err)
	}

	if pid != os.Getpid() {
		return 0, nil
	}

	listenerCountStr, listenerCountSet := os.LookupEnv(listenerCountEnvName)
	if !listenerCountSet {
		return 0, fmt.Errorf("%w: %s", ErrEnvMissing, listenerCountEnvName)
	}

	listenerCount, err := strconv.Atoi(listenerCountStr)
	if err != nil {
		return 0, fmt.Errorf("invalid listen count: %w", err)
	}

	return listenerCount, nil
}
