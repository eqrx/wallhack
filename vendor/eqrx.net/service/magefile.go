//go:build mage
// +build mage

package main

import (
	"fmt"

	"github.com/magefile/mage/sh"
)

func Lint() error {
	if err := sh.RunV("golangci-lint", "run", "./..."); err != nil {
		return fmt.Errorf("lint: %w", err)
	}

	return nil
}
