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

package server

import (
	"fmt"
	"os"
	"plugin"

	"eqrx.net/wallhack"
)

func loadPlugin() (wallhack.Plugin, error) {
	path, pluginSet := os.LookupEnv(wallhack.PluginPathEnvName)
	if !pluginSet {
		return nil, nil
	}

	plugin, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open http plugin: %w", err)
	}

	newPluginSymbol, err := plugin.Lookup(wallhack.PluginNewSymbolName)
	if err != nil {
		return nil, fmt.Errorf("lookup http plugin server setup symbol: %w", err)
	}

	return newPluginSymbol.(func() interface{})().(wallhack.Plugin), nil
}
