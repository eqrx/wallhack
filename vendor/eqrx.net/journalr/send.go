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

package journalr

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	// priorityError is the syslog priority level used for Error calls.
	priorityError = "3"
	// priorityInfo is the syslog priority level used for Info calls.
	priorityInfo = "6"
)

// send a log message to the journal.
func (s *Sink) send(values []interface{}) {
	buf := &bytes.Buffer{}

	for idx := 0; idx < len(values); idx += 2 {
		key := values[idx].(string)

		value := fmt.Sprint(values[idx+1])
		if strings.ContainsRune(value, '\n') {
			buf.WriteString(key)
			buf.WriteRune('\n')

			if err := binary.Write(buf, binary.LittleEndian, uint64(len(value))); err != nil {
				panic(err)
			}

			buf.WriteString(value)
			buf.WriteRune('\n')
		} else {
			buf.WriteString(key)
			buf.WriteRune('=')
			buf.WriteString(value)
			buf.WriteRune('\n')
		}
	}

	if _, err := s.conn.Write(buf.Bytes()); err != nil {
		panic(fmt.Sprintf("write to journal: %v", err))
	}
}

// format given values into a string.
func formatValues(values []interface{}) string {
	buf := &bytes.Buffer{}
	for idx := 0; idx < len(values); idx += 2 {
		buf.WriteString(values[idx].(string))
		buf.WriteString(" = ")
		buf.WriteString(fmt.Sprint(values[idx+1]))
		buf.WriteRune('\n')
	}

	return buf.String()
}

// Error logs an error, with the given message and key/value pairs as
// context.  See Logger.Error for more details.
func (s *Sink) Error(err error, msg string, kvList ...interface{}) {
	values := mergeValues(s.values, kvList)

	if len(values) == 0 {
		msg += ": " + err.Error()
	} else {
		msg += ":\n" + formatValues(values)
	}

	s.send(append(values, "MESSAGE", msg, "PRIORITY", priorityError, "ERROR", err.Error()))
}

// Info logs a non-error message with the given key/value pairs as context.
// The level argument is provided for optional logging.  This method will
// only be called when Enabled(level) is true. See Logger.Info for more
// details.
func (s *Sink) Info(level int, msg string, kvList ...interface{}) {
	values := mergeValues(s.values, kvList)

	if len(values) != 0 {
		msg += ":\n" + formatValues(values)
	}

	s.send(append(values, "MESSAGE", msg, "PRIORITY", priorityInfo))
}
