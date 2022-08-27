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

// Package rungroup allows goroutines to share the same lifecycle.
//
// Like https://pkg.go.dev/golang.org/x/sync/errgroup, rungroup allows to create a handler that allows to run given
// functions in parallel with goroutines. It tries to be as close to the interface of errgroup but handles the
// cancelation of its routines and treatment of errors differently.
//
// # Differences to errgroup
//
// Goroutine function gets a context as first argument, a new group does not get handled to the caller with the groups
// context. By default all routines get canceled as soon as one routine return, regardless if the error is nil or not.
// This can be overridden per routine. errgroup only cancels on the first non nil error. All non nil errors are
// returned to the creator of the group. errgroup only returns the first error and drops the rest.
package rungroup

import (
	"context"
	"sync"
)

// Group represents a set of goroutines which lifecycles are bound to each other.
//
//nolint:containedctx // Group is a context manager and needs to have access to said context.
type Group struct {
	ctx    context.Context // Passed to spawned goroutines.
	cancel func()          // Cancels ctx.
	wg     sync.WaitGroup  // WaitGroup that completes when all routines spawned from Group have returned.
	mtx    sync.Mutex      // Lock for access to errs.
	errs   []error         // Slice of errors that were returned by spawned goroutines.
}

type (
	// optionSet contains settings regarding spawned go routines. Only to be used with Option.
	optionSet struct {
		noCancelOnSuccess bool
		noCancelOnError   bool
	}
	// Option modifies the settings of a spawned routine with Group.Go.
	Option func(o *optionSet)

	// Function specifies the signature of functions that can be run via the group.
	Function func(context.Context) error
)

// NoCancelOnSuccess prevents goroutines spawned with Group.Go to cancel the group context when they return
// a non nil error. Default is to cancel the group context on return regardless of the returned error.
func NoCancelOnSuccess(o *optionSet) { o.noCancelOnSuccess = true }

// NeverCancel prevents goroutines spawned with Group.Go to cancel the group context  in any case.
// Default is to cancel the group context on return regardless of the returned error.
func NeverCancel(o *optionSet) {
	o.noCancelOnError = true
	o.noCancelOnSuccess = true
}

// New creates group for goroutine management. The context passed as parameter ctx with be taken as parent for the
// group context. Canceling it will cancel all spawned goroutines. ctx must not be nil.
func New(ctx context.Context) *Group {
	ctx, cancel := context.WithCancel(ctx)

	return &Group{ctx, cancel, sync.WaitGroup{}, sync.Mutex{}, []error{}}
}

// Wait block until all goroutines of the group have returned to it and returns *Error if any error was returned
// returned by the routines. This method must not be called by multiple goroutines at the same time. After this
// call returnes, the group may not be reused.
func (g *Group) Wait() error {
	g.wg.Wait()
	g.mtx.Lock()
	defer g.mtx.Unlock()

	errs := g.errs
	g.errs = nil

	if len(errs) != 0 {
		return &Error{errs}
	}

	return nil
}

// Go spawns a goroutine and calls the function fnc with it. The context of the group is passed as the first
// argument to it.
//
// When any routine spawned by Go return, the following things happen: Panics are not recovered. If the returned error
// value of fnc is non nil it is stored for retrieval by Wait. Depending on the given options the group context is
// on returned depending of the error. Default options cause the context always to be canceled.
//
// As long as no call from Wait has returned, Go may be called by any goroutines at the same time. Passing nil as fnc or
// part of opts is not allowed.
func (g *Group) Go(fnc Function, opts ...Option) {
	g.wg.Add(1)

	options := &optionSet{false, false}
	for _, option := range opts {
		option(options)
	}

	go func() {
		defer g.wg.Done()

		if err := fnc(g.ctx); err != nil {
			if !options.noCancelOnError {
				g.cancel()
			}

			g.mtx.Lock()
			g.errs = append(g.errs, err)
			g.mtx.Unlock()
		} else if !options.noCancelOnSuccess {
			g.cancel()
		}
	}()
}
