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

// Package journalr allows sending log messages to systemd journald via logr.Logger.
package journalr

import (
	"fmt"
	"net"

	"github.com/go-logr/logr"
)

// journalPath indicates where to find the journald unix socket.
const journalPath = "/run/systemd/journal/socket"

// Sink is a logr.LogSink that sends structured log messages to systemd journald.
type Sink struct {
	conn      *net.UnixConn
	level     int
	callDepth int
	name      string
	values    []interface{}
}

// NewSink creates a new Sink for sending structured log messages to systemd journald.
func NewSink() (*Sink, error) {
	conn, err := net.Dial("unixgram", journalPath)
	if err != nil {
		return nil, fmt.Errorf("connect to journal: %w", err)
	}

	unixConn := conn.(*net.UnixConn)

	return &Sink{unixConn, 0, 1, "", []interface{}{}}, nil
}

// WithValues returns a new LogSink with additional key/value pairs.  See
// Logger.WithValues for more details.
func (s Sink) WithValues(newValues ...interface{}) logr.LogSink {
	s.values = mergeValues(s.values, newValues)

	return &s
}

// WithCallDepth returns a LogSink that will offset the call
// stack by the specified number of frames when logging call
// site information.
//
// If depth is 0, the LogSink should skip exactly the number
// of call frames defined in RuntimeInfo.CallDepth when Info
// or Error are called, i.e. the attribution should be to the
// direct caller of Logger.Info or Logger.Error.
//
// If depth is 1 the attribution should skip 1 call frame, and so on.
// Successive calls to this are additive.
func (s Sink) WithCallDepth(depth int) logr.LogSink {
	s.values = append(make([]interface{}, 0, len(s.values)), s.values...)
	s.callDepth += depth

	return &s
}

// WithName returns a new LogSink with the specified name appended.  See
// Logger.WithName for more details.
func (s Sink) WithName(name string) logr.LogSink {
	s.values = append(make([]interface{}, 0, len(s.values)), s.values...)

	if s.name == "" {
		s.name = name
	} else {
		s.name = s.name + "/" + name
	}

	return &s
}

// Enabled tests whether this LogSink is enabled at the specified V-level.
// For example, commandline flags might be used to set the logging
// verbosity and disable some info logs.
func (s Sink) Enabled(level int) bool { return true }

// Init receives optional information about the logr library for LogSink
// implementations that need it.
func (s *Sink) Init(info logr.RuntimeInfo) { s.callDepth += info.CallDepth }
