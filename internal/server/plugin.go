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
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"plugin"
)

const (
	// HTTPPluginPathEnvName is the environment name that contains the path to a go plugin that is loaded by wallhack
	// for serving extra HTTP stuff.
	HTTPPluginPathEnvName = "WALLHACK_HTTP_PLUGIN"

	// HTTPPluginServerSetupSymbolName is the name of the symbol within the HTTP plugin that is responsible for
	// setting up a given http server for the plugins purpose. Needs to have the signature func(*http.Server) error.
	HTTPPluginServerSetupSymbolName = "SetupHTTPServer"
	// HTTPPluginTLSConfigSetupSymbolName is the name of the symbol within the HTTP plugin that is responsible for
	// setting up a given tls config for the plugins purpose. Needs to have the signature func(*tls.Config) error.
	HTTPPluginTLSConfigSetupSymbolName = "SetupTLSConfig"
)

var errSymbolType = errors.New("plugin symbol has unexpected symbol type")

func loadHTTPPlugin() (setupTLS func(*tls.Config), setupHTTP func(*http.Server) error, err error) {
	path, pluginSet := os.LookupEnv(HTTPPluginPathEnvName)
	if !pluginSet {
		return func(*tls.Config) {}, func(*http.Server) error { return nil }, nil
	}

	plugin, err := plugin.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open http plugin: %w", err)
	}

	setupServerSymbol, err := plugin.Lookup(HTTPPluginServerSetupSymbolName)
	if err != nil {
		return nil, nil, fmt.Errorf("lookup http plugin server setup symbol: %w", err)
	}

	var symbolSet bool

	setupHTTP, symbolSet = setupServerSymbol.(func(*http.Server) error)
	if !symbolSet {
		return nil, nil, fmt.Errorf("%w: need %T, have %T", errSymbolType, setupHTTP, setupServerSymbol)
	}

	setupTLSSymbol, err := plugin.Lookup(HTTPPluginTLSConfigSetupSymbolName)
	if err != nil {
		return nil, nil, fmt.Errorf("lookup http plugin server setup symbol: %w", err)
	}

	setupTLS, symbolSet = setupTLSSymbol.(func(*tls.Config))
	if !symbolSet {
		return nil, nil, fmt.Errorf("%w: need %T, have %T", errSymbolType, setupTLS, setupTLSSymbol)
	}

	return setupTLS, setupHTTP, nil
}
