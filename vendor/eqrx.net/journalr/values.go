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

import "strings"

// mergeValues of newValues into oldValues. Keys existing in oldValues will be overwritten.
// Keys in newValues will be converted to upper case.
func mergeValues(oldValues, newValues []interface{}) []interface{} {
	if len(newValues)%2 != 0 {
		panic("uneven number of new values")
	}

	oldValues = append(make([]interface{}, 0, len(oldValues)), oldValues...)

	for newIndex := 0; newIndex < len(newValues); newIndex += 2 {
		newKey := asSanitizedKey(newValues[newIndex])

		replaced := false

		for oldIndex := 0; oldIndex < len(oldValues); oldIndex += 2 {
			oldKey := oldValues[oldIndex].(string)

			if newKey == oldKey {
				replaced = true
				oldValues[oldIndex+1] = newValues[newIndex+1]

				break
			}
		}

		if !replaced {
			oldValues = append(oldValues, newKey, newValues[newIndex+1])
		}
	}

	return oldValues
}

// asSanitizedKey ensures that the given value is a valid key and returns it in the sanitized form.
func asSanitizedKey(key interface{}) string {
	keyStr := key.(string)
	keyStr = strings.ToUpper(keyStr)
	keyStr = strings.ReplaceAll(keyStr, " ", "_")

	switch key {
	case "":
		panic("empty key")
	case "MESSAGE":
		panic("key named MESSAGE")
	case "PRIORITY":
		panic("key named PRIORITY")
	}

	if keyStr[0] == '_' {
		panic("key with starting underscore")
	}

	for _, r := range keyStr {
		ensureKeyRuneValid(r)
	}

	return keyStr
}

// ensureKeyRuneValid panics if the given rune may not be part of a syslog key.
func ensureKeyRuneValid(r rune) {
	if !(('A' <= r && r <= 'Z') || ('0' <= r && r <= '9') || r == '_') {
		panic("key with invalid runes")
	}
}
