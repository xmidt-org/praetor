// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"sync"
	"sync/atomic"
)

//go:generate stringer -type=Status -linecomment

// Status represents the Consul service status. The String() value
// for this type is the correct value to use with Consul's service check API.
type Status int

const (
	// Any is a wildcard status. It's not intended to be used as an actual service status.
	Any Status = iota // any

	// Passing indicates that a service is fully healthy.
	Passing // passing

	// Warning indicates that a service can still take some traffic, but
	// that something is wrong.
	Warning // warning

	// Critical means that a service cannot take traffic and is down.
	Critical // critical

	// Maintenance indicates a service that is temporarily unavailable,
	// most often due to server maintenance.
	Maintenance // maintenance
)

// State is a service's overall state. The zero value of this type represents
// a healthy service with no output.
type State struct {
	// Output is the additional detail text associated with this state. The content
	// of this type is not used by Consul, and it can be anything. Most often, it is
	// either (1) simple, human readable text, or (2) a JSON object.
	Output string

	// Status is the Consul service status.
	Status Status
}

// StateAccessor defines the behavior of anything that can atomically access
// the State of a registered consul service.
type StateAccessor interface {
	// State is the current health state for this instance. Different registered
	// services are allowed to have different states.
	//
	// This is the value sent in any TTL updates associated with this instance. It should
	// also by the value sent by any HTTP health endpoints the application implements.
	State() State

	// SetState updates the current state. This method may be called at any time.
	//
	// Updating or obtaining State is always atomic and safe for concurrent access.
	SetState(State) (previous State)
}

// stateAccessor is a concurrent-safe access point for a State object.
type stateAccessor struct {
	lock  sync.Mutex
	value atomic.Value
}

// newStateAccessor creates a stateHolder access point with the given initial state.
func newStateAccessor(initial State) *stateAccessor {
	sh := new(stateAccessor)
	sh.value.Store(initial)
	return sh
}

func (sh *stateAccessor) State() State {
	return sh.value.Load().(State)
}

func (sh *stateAccessor) SetState(s State) (previous State) {
	sh.lock.Lock()
	previous, _ = sh.value.Load().(State) // allow Store not to have been called yet
	sh.value.Store(s)
	sh.lock.Unlock()

	return
}
