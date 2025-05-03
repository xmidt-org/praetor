// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"iter"
)

// Registrars is an aggregate of multiple Registrar instances. An application can register itself
// as implementing several services with consul, and a Registrars holds the state of
// each registered service.
type Registrars interface {
	// State returns a snapshot of the current states of all contained Registrar
	// instances.
	//
	// If this Registrars is empty, the returned map will be empty.
	State() (current map[ServiceID]State)

	// SetState updates the state for all contained Registrar instances.
	// The returned map holds the previous states for each Registrar.
	//
	// If this Registrars is empty, the returned map will be empty and no
	// State change will occur.
	SetState(State) (previous map[ServiceID]State)

	// Len returns the count of contained Registrar instances.
	Len() int

	// Registrars provides iteration over the contained Registrar instances.
	Registrars() iter.Seq2[ServiceID, Registrar]
}

// NewRegistrars creates an aggregate Registrars from a definitions bundle. The
// opts will be applied to each created Registrar.
//
// The Definitions bundle can be nil or empty, in which case a non-nil, empty
// Registrars is returned.
func NewRegistrars(definitions *Definitions, opts ...RegistrarOption) (Registrars, error) {
	r := &registrars{
		all: make([]Registrar, 0, definitions.len()),
	}

	if definitions != nil {
		for def := range definitions.all() {
			if registrar, err := newRegistrar(def, opts...); err != nil {
				return nil, err
			} else {
				r.all = append(r.all, registrar)
			}
		}
	}

	return r, nil
}

type registrars struct {
	all []Registrar
}

func (rs *registrars) State() (current map[ServiceID]State) {
	current = make(map[ServiceID]State, len(rs.all))
	for _, r := range rs.all {
		current[r.ServiceID()] = r.State()
	}

	return
}

func (rs *registrars) SetState(new State) (previous map[ServiceID]State) {
	previous = make(map[ServiceID]State, len(rs.all))
	for _, r := range rs.all {
		previous[r.ServiceID()] = r.SetState(new)
	}

	return
}

func (rs *registrars) Len() int {
	return len(rs.all)
}

func (rs *registrars) Registrars() iter.Seq2[ServiceID, Registrar] {
	return func(f func(ServiceID, Registrar) bool) {
		for _, r := range rs.all {
			if !f(r.ServiceID(), r) {
				return
			}
		}
	}
}
