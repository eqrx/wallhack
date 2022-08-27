[![Go Reference](https://pkg.go.dev/badge/eqrx.net/journalr.svg)](https://pkg.go.dev/eqrx.net/journalr)
# journalr

This project provides a [logr](https://github.com/go-logr/logr) sink that writes log messages to the systemd
journal. While normal log messages get written to the journal as text this sink also writes values of a log 
message as fields so they can be parsed later.

This project is released under GNU Affero General Public License v3.0, see LICENCE file in this repo for more info.
