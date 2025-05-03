// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"errors"
	"fmt"
	"iter"
	"time"

	"github.com/hashicorp/consul/api"
	"go.uber.org/multierr"
)

const (
	// DefaultReplaceExistingChecks is the default value for the ReplaceExistingChecks flag
	// used when registering services. This flag controls whether checks are replaced when
	// reregistering the same service.
	//
	// See: https://developer.hashicorp.com/consul/api-docs/agent/service#replace-existing-checks
	DefaultReplaceExistingChecks bool = true
)

// ServiceID is a unique identifier for registered consul services.
type ServiceID string

func serviceIDOf(reg api.AgentServiceRegistration) (sid ServiceID) {
	sid = ServiceID(reg.ID)
	if len(sid) == 0 {
		sid = ServiceID(reg.Name)
	}

	return
}

// CheckID is a unique identifier for registered consul checks, either as part of a service
// registration or independent checks associated with a ServiceID.
type CheckID string

func checkIDOf(check api.AgentServiceCheck) (cid CheckID) {
	cid = CheckID(check.CheckID)
	if len(cid) == 0 {
		cid = CheckID(check.Name)
	}

	return
}

// checkIDSet tracks check identifiers for uniqueness.
type checkIDSet map[CheckID]bool

// add adds an identifier to this set. if the given id
// is a duplicate, this method returns an error.
func (cis *checkIDSet) add(id CheckID) (err error) {
	if cis == nil {
		*cis = make(checkIDSet)
	}

	if (*cis)[id] {
		err = fmt.Errorf("duplicate check [%s]", id)
	} else {
		(*cis)[id] = true
	}

	return
}

// merge inserts another checkIDSet into this one. if there
// are any duplicates, this method returns an error.
func (cis *checkIDSet) merge(more checkIDSet) (err error) {
	for id := range more {
		err = multierr.Append(err, cis.add(id))
	}

	return
}

// parseCheckTTL parses the check's TTL field and returns the result. If the check
// does not represent a TTL, this function returns a zero (0) duration and a nil error.
func parseCheckTTL(c api.AgentServiceCheck) (d time.Duration, err error) {
	if len(c.TTL) > 0 {
		d, err = time.ParseDuration(c.TTL)
	}

	return
}

// ttlDefinition holds information about a single TTL check that is part
// of a service's embedded checks.
type ttlDefinition struct {
	// id is the unique check identifier for this TTL check.
	id CheckID

	// interval is the time interval at which this TTL is updated.
	interval time.Duration

	// updateOptions are the set of options used when updating the TTL.
	updateOptions api.QueryOptions
}

// serviceDefinition holds everything praetor knows about a service that can
// be registered and deregistered with consul.
type serviceDefinition struct {
	// id is the unique service identifier for this service. This field
	// is required.
	id ServiceID

	// Registration is the consul registration for this service. This field
	// is required.
	registration api.AgentServiceRegistration

	// registerOptions are the options used when registering this service.
	// This field is optional.
	registerOptions api.ServiceRegisterOpts

	// checkIDs holds all the check identifiers that were explicitly set
	// within the registration.
	checkIDs checkIDSet

	// TTLS hold information about the checks that are ttls, contained within
	// the Registration field.
	ttls []ttlDefinition
}

// serviceDefinitionSet holds a set of definitions with unique service identifiers.
type serviceDefinitionSet map[ServiceID]serviceDefinition

// add inserts the given serviceDefinition. if the service id is a duplicate,
// this method returns an error.
func (sds *serviceDefinitionSet) add(sd serviceDefinition) (err error) {
	if sds == nil {
		*sds = make(serviceDefinitionSet)
	}

	if _, exists := (*sds)[sd.id]; exists {
		err = fmt.Errorf("duplicate service [%s]", sd.id)
	} else {
		(*sds)[sd.id] = sd
	}

	return
}

// checksLen returns the total number of checks. useful for preallocating things.
func (sd serviceDefinition) checksLen() (n int) {
	n += len(sd.registration.Checks)
	if sd.registration.Check != nil {
		n++
	}

	return
}

// checks provides iteration over the set of checks in this definition.
func (sd serviceDefinition) checks() iter.Seq2[CheckID, api.AgentServiceCheck] {
	return func(f func(CheckID, api.AgentServiceCheck) bool) {
		if sd.registration.Check != nil {
			cid := checkIDOf(*sd.registration.Check)
			if !f(cid, *sd.registration.Check) {
				return
			}
		}

		for _, c := range sd.registration.Checks {
			cid := checkIDOf(*c)
			if !f(cid, *c) {
				return
			}
		}
	}
}

// ServiceDefinitionOption is a configurable option for defining a registerable service.
type ServiceDefinitionOption interface {
	apply(*serviceDefinition) error
}

type serviceDefinitionOptionFunc func(*serviceDefinition) error

func (f serviceDefinitionOptionFunc) apply(sd *serviceDefinition) error { return f(sd) }

// WithRegisterOptions sets the options used to register this definition's service.
//
// By default, ReplaceExistingChecks is set to true. This option can be used to change that.
func WithRegisterOptions(opts api.ServiceRegisterOpts) ServiceDefinitionOption {
	return serviceDefinitionOptionFunc(func(sd *serviceDefinition) error {
		sd.registerOptions = opts
		return nil
	})
}

// newServiceDefinition builds the internal representation of what praetor needs to manage
// a single service registration.
func newServiceRegistration(reg api.AgentServiceRegistration, opts ...ServiceDefinitionOption) (sd serviceDefinition, err error) {
	sd.registration = reg
	sd.registerOptions.ReplaceExistingChecks = DefaultReplaceExistingChecks
	sd.checkIDs = make(checkIDSet, sd.checksLen())

	sd.id = serviceIDOf(sd.registration)
	if len(sd.id) == 0 {
		err = multierr.Append(err, errors.New("service registrations must have an id or name"))
	}

	for cid, c := range sd.checks() {
		interval, ttlErr := parseCheckTTL(c)
		switch {
		case ttlErr != nil:
			err = multierr.Append(err, ttlErr)

		case interval < 0:
			err = multierr.Append(err, errors.New("negative ttl intervals are not allowed"))

		case len(cid) == 0 && interval == 0:
			// checks that have no id and are not TTLs can be skipped.
			// consul will generate identifiers for these checks.

		case len(cid) == 0 && interval > 0:
			// we don't support ttl checks with no identifiers
			err = multierr.Append(err, errors.New("ttl checks must have an id or name"))

		default:
			err = multierr.Append(err, sd.checkIDs.add(cid))
			if interval > 0 {
				sd.ttls = append(sd.ttls,
					ttlDefinition{
						id:       cid,
						interval: interval,
					},
				)
			}
		}
	}

	for _, o := range opts {
		err = multierr.Append(err, o.apply(&sd))
	}

	return
}
