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

//go:build mage
// +build mage

package main

import (
	"fmt"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func BuildWallhack() error {
	cmd := "wallhack"
	env := map[string]string{"CGO_ENABLED": "1"}

	cmdline := []string{
		"build",
		"--ldflags", "-w -s --extldflags '-O1'",
		"--trimpath", "--mod=readonly",
		"-v", "-o", fmt.Sprintf("./bin/%s/%s", runtime.GOARCH, cmd), "./cmd/" + cmd,
	}

	if err := sh.RunWithV(env, "go", cmdline...); err != nil {
		return fmt.Errorf("compile ./cmd/%s: %w", cmd, err)
	}

	return nil
}

func Lint() error {
	if err := sh.RunV("golangci-lint", "run", "./..."); err != nil {
		return fmt.Errorf("lint: %w", err)
	}

	return nil
}

func TestUnit() error {
	env := map[string]string{"CGO_ENABLED": "1"}
	cmdline := []string{"test", "-cover", "-race", "./..."}
	if err := sh.RunWithV(env, "go", cmdline...); err != nil {
		return fmt.Errorf("test unit: %w", err)
	}

	return nil
}

func Test() { mg.Deps(TestUnit) }
func Build() {
	mg.Deps(BuildWallhack)
}
