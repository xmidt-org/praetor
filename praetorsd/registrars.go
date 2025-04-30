// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"iter"
)

// Registrars is an aggregate of multiple Registrar. An application can register itself
// as implementing several services with consul, and a Registrars holds the state of
// each registered service.
type Registrars interface {
	Registrars() iter.Seq2[ServiceID, Registrar]
}

func NewRegistrars(services *Registrations, opts ...RegistrarOption) (Registrars, error) {
	r := &registrars{
		all: make([]Registrar, 0, services.ServiceRegistrationsLen()),
	}

	if services != nil {
		for _, reg := range services.ServiceRegistrations() {
			if registrar, err := NewRegistrar(reg, opts...); err != nil {
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

func (rs *registrars) Registrars() iter.Seq2[ServiceID, Registrar] {
	return func(f func(ServiceID, Registrar) bool) {
		for _, r := range rs.all {
			if !f(r.ServiceID(), r) {
				return
			}
		}
	}
}
