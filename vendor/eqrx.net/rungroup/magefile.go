//go:build mage
// +build mage

package main

import (
	"fmt"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

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
