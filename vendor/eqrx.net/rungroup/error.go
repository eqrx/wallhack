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

package rungroup

import (
	"bytes"
)

// Error wraps multiple errors into one error instance. It does not support unwrapping since the current interface
// design of Go allows only for one child of an error to be unwrapped. If you need to know the concrete types
// please go over Errs manually.
type Error struct {
	Errs []error
}

func (e Error) Error() string {
	errs := e.Errs
	switch len(errs) {
	case 0:
		panic("empty errs")
	case 1:
		return e.Errs[0].Error()
	default:
		buf := bytes.Buffer{}
		buf.WriteString("[ ")

		for {
			if len(errs) == 1 {
				buf.WriteString(errs[0].Error() + " ")

				break
			}

			buf.WriteString(errs[0].Error() + ", ")
			errs = errs[1:]
		}

		buf.WriteString("]")

		return buf.String()
	}
}
