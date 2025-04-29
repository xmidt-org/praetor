// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"errors"
	"fmt"
	"iter"

	"github.com/hashicorp/consul/api"
	"go.uber.org/multierr"
)

// ServiceID is a unique identifier for registered consul services.
type ServiceID string

// getServiceRegistrationID returns the ServiceID for a given service registration, if one exists.
// This function checks the ID field first, falling back to the Name field is ID is not set.
func getServiceRegistrationID(s api.AgentServiceRegistration) (sid ServiceID) {
	if sid = ServiceID(s.ID); len(sid) == 0 {
		sid = ServiceID(s.Name)
	}

	return
}

// CheckID is a unique identifier for registered consul checks, either as part of a service
// registration or independent checks associated with a ServiceID.
type CheckID string

// CheckKey holds the tuple of identifiers that uniquely specify a check in a sequence.
type CheckKey struct {
	// ServiceID is the unique identifier for the service containing this check.
	ServiceID ServiceID

	// CheckID is the unique identifier for the check. Note that this can be empty,
	// as in the case where client code expects consul to generate the check ID.
	CheckID CheckID
}

// getServiceCheckID returns the CheckID for an check embedded within a service registration.
// This function checks the CheckID field first, falling back to the Name field is CheckID is not set.
func getServiceCheckID(c api.AgentServiceCheck) (cid CheckID) {
	if cid = CheckID(c.CheckID); len(cid) == 0 {
		cid = CheckID(c.Name)
	}

	return
}

// getCheckKey is a helper that creates a CheckKey for an embedded service check.
func getCheckKey(sid ServiceID, c api.AgentServiceCheck) CheckKey {
	return CheckKey{
		ServiceID: sid,
		CheckID:   getServiceCheckID(c),
	}
}

// ServiceRegistrationChecksLen returns the count of registered checks associated with the given
// service registrations.
func ServiceRegistrationChecksLen(services ...api.AgentServiceRegistration) (n int) {
	for _, s := range services {
		n += len(s.Checks)
		if s.Check != nil {
			n++
		}
	}

	return
}

// ServiceRegistrationChecks allows easy iteration over checks embedded in service registrations.
// Note that embedded checks may not have a CheckID, as would be the case when client code wants
// consul to generate unique check id's.
func ServiceRegistrationChecks(services ...api.AgentServiceRegistration) iter.Seq2[CheckKey, api.AgentServiceCheck] {
	return func(f func(CheckKey, api.AgentServiceCheck) bool) {
		for _, s := range services {
			sid := getServiceRegistrationID(s)
			if s.Check != nil {
				if !f(getCheckKey(sid, *s.Check), *s.Check) {
					return
				}
			}

			for _, check := range s.Checks {
				if !f(getCheckKey(sid, *check), *check) {
					return
				}
			}
		}
	}
}

// RegistrationsBuilder is a Fluent Builder for creating Registrations bundles.
//
// The zero value is a ready to use builder. This builder is not safe for
// concurrent usage.
type RegistrationsBuilder struct {
	services    map[ServiceID]api.AgentServiceRegistration
	allCheckIDs map[CheckID]bool

	err error
}

// appendErr adds the given error to our accumulator.
func (rb *RegistrationsBuilder) appendErr(err error) {
	rb.err = multierr.Append(rb.err, err)
}

// appendServiceRegistration safely inserts the given service, ensuring that any
// data structures are created.
func (rb *RegistrationsBuilder) appendServiceRegistration(sid ServiceID, s api.AgentServiceRegistration) {
	if rb.services == nil {
		rb.services = make(map[ServiceID]api.AgentServiceRegistration)
	}

	rb.services[sid] = s
}

// recordCheckID notes that a CheckID has been encountered. This could be either an embedded
// AgentServiceCheck or a standalone AgentCheckRegistration.
//
// This method ensures that any needed data structures are created.
func (rb *RegistrationsBuilder) recordCheckID(cid CheckID) {
	if rb.allCheckIDs == nil {
		rb.allCheckIDs = make(map[CheckID]bool)
	}

	rb.allCheckIDs[cid] = true
}

// AddServiceRegistrations appends service registrations to this builder. The registrations are
// required to have service names or id's that are unique. Any embedded checks may have no id or name,
// but if any embedded check provides an id it must be unique.
//
// Any errors that occur are accumulated and made available when Build is called.
func (rb *RegistrationsBuilder) AddServiceRegistrations(services ...api.AgentServiceRegistration) *RegistrationsBuilder {
	for _, s := range services {
		sid := getServiceRegistrationID(s)
		_, serviceExists := rb.services[sid]

		switch {
		case len(sid) == 0:
			rb.appendErr(errors.New("a service id or name is required"))

		case serviceExists:
			rb.appendErr(fmt.Errorf("duplicate service id [%s]", sid))

		default:
			rb.appendServiceRegistration(sid, s)

			// verify that any embedded checks with ids (including names) are unique
			for key := range ServiceRegistrationChecks(s) {
				switch {
				case len(key.CheckID) == 0:
					// skip embedded checks with no id, as consul will generate those

				case rb.allCheckIDs[key.CheckID]:
					rb.appendErr(fmt.Errorf("duplicate check id [%s]", key.CheckID))

				default:
					rb.recordCheckID(key.CheckID)
				}
			}
		}
	}

	return rb
}

// Err returns any accumulated error thus far.
func (rb *RegistrationsBuilder) Err() error {
	return rb.err
}

// Reset clears this builder's internal state. When Build is called,
// this builder's state is always reset.
func (rb *RegistrationsBuilder) Reset() *RegistrationsBuilder {
	*rb = RegistrationsBuilder{}
	return rb
}

// Build creates a new Registrations if possible. If any errors occurred during building, a nil
// Registrations is returned along with an aggregate error.
//
// This method always resets the state of this builder.
func (rb *RegistrationsBuilder) Build() (r *Registrations, err error) {
	if err = rb.err; err == nil {
		r = &Registrations{
			services: rb.services,
		}
	}

	rb.Reset()
	return
}

// Registrations is an immutable bundle of consul registrations. Both service registrations
// and check registrations (separate from a service object) are supported. A Registrations should be
// created via a RegistrationsBuilder.
//
// The zero value of this type is an empty bundle and is usable.  However, no additional registrations
// may be added.  Use a RegistrationsBuilder rather than creating instances of this type directly.
type Registrations struct {
	services map[ServiceID]api.AgentServiceRegistration
}

// ServiceRegistrationsLen returns the count of consul service registrations in this bundle.
func (r *Registrations) ServiceRegistrationsLen() int {
	return len(r.services)
}

// ServiceRegistrations provides read-only iteration over the set of consul
// AgentServiceRegistrations.  The sequence of registrations is guaranteed to have
// unique service identifiers, and any embedded checks will have unique check identifiers
// if supplied.
func (r *Registrations) ServiceRegistrations() iter.Seq2[ServiceID, api.AgentServiceRegistration] {
	return func(yield func(ServiceID, api.AgentServiceRegistration) bool) {
		for sid, s := range r.services {
			if !yield(sid, s) {
				return
			}
		}
	}
}
