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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"plugin"
)

// Plugin defines what methods a wallhack plugin needs to implement.
type Plugin interface {
	TLSConfig() *tls.Config
	Listen(context.Context, net.Listener) error
}

const (
	// PluginPathEnvName is the environment name that contains the path to a go plugin that is loaded by wallhack
	// for serving extra stuff.
	PluginPathEnvName = "WALLHACK_PLUGIN_PATH"

	// PluginNewSymbolName is the name of the symbol within the plugin that is responsible for
	// returning the Plugin interface.
	PluginNewSymbolName = "New"
)

func loadPlugin() (Plugin, error) { //nolint:ireturn
	path, pluginSet := os.LookupEnv(PluginPathEnvName)
	if !pluginSet {
		return nil, nil //nolint:nilnil
	}

	plugin, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("load server plugin: %w", err)
	}

	newPluginSymbol, err := plugin.Lookup(PluginNewSymbolName)
	if err != nil {
		return nil, fmt.Errorf("load server plugin: %w", err)
	}

	return newPluginSymbol.(func() interface{})().(Plugin), nil
}
