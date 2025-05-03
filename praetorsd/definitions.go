// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"iter"
	"maps"
	"slices"

	"github.com/hashicorp/consul/api"
	"go.uber.org/multierr"
)

// DefinitionsBuilder is a Fluent Builder for creating Definitions bundles.
//
// The zero value is a ready to use builder. This builder is not safe for
// concurrent usage.
type DefinitionsBuilder struct {
	services    serviceDefinitionSet
	allCheckIDs checkIDSet

	err error
}

// appendErrs adds the given errors, if any, to our accumulator.
func (rb *DefinitionsBuilder) appendErrs(errs ...error) {
	rb.err = multierr.Append(
		rb.err,
		multierr.Combine(errs...),
	)
}

// DefineService defines a single service for registration. Any errors that occur can
// be accessed with Err() or as the result of Build().
//
// Services must have an identifier, either by setting the Name or ID field of api.AgentServiceRegistration.
//
// Checks defined on the api.AgentServiceRegistration do not have to have identifiers, as in that
// case consul will generate them. However, if a check has an identifier, is must be unique within
// the entire Definitions bundle being built.
//
// IMPORTANT: TTL Checks MUST have an identifier.
func (rb *DefinitionsBuilder) DefineService(reg api.AgentServiceRegistration, opts ...ServiceDefinitionOption) *DefinitionsBuilder {
	sd, err := newServiceRegistration(reg, opts...)
	rb.appendErrs(err)

	if err == nil {
		rb.appendErrs(
			rb.allCheckIDs.merge(sd.checkIDs),
			rb.services.add(sd),
		)
	}

	return rb
}

// DefineServices defines multiple services for registration. All the same caveats apply
// as with DefineService(). The set of options is applied to each definition that is created.
func (rb *DefinitionsBuilder) DefineServices(regs iter.Seq[api.AgentServiceRegistration], opts ...ServiceDefinitionOption) *DefinitionsBuilder {
	for reg := range regs {
		rb = rb.DefineService(reg, opts...)
	}

	return rb
}

// Err returns any accumulated error thus far.
func (rb *DefinitionsBuilder) Err() error {
	return rb.err
}

// Reset clears this builder's internal state. When Build is called,
// this builder's state is always reset.
func (rb *DefinitionsBuilder) Reset() *DefinitionsBuilder {
	*rb = DefinitionsBuilder{}
	return rb
}

// Build creates a new Definitions bundle if possible. If any errors occurred during building, a nil
// Definitions is returned along with an aggregate error.
//
// This method always resets the state of this builder.
func (rb *DefinitionsBuilder) Build() (r *Definitions, err error) {
	if err = rb.err; err == nil {
		r = &Definitions{
			services: slices.Collect(
				maps.Values(rb.services),
			),
		}
	}

	rb.Reset()
	return
}

// Definitions is an immutable bundle of consul service registrations. A Definitions should be
// created via a DefinitionsBuilder.
//
// The zero value of this type is an empty bundle and is usable.  However, no additional registrations
// may be added.  Use a DefinitionsBuilder rather than creating instances of this type directly.
type Definitions struct {
	services []serviceDefinition
}

// len returns the total number of service definitions in this bundle.
func (r *Definitions) len() int {
	return len(r.services)
}

// all provides iteration over the service definitions in this bundle.
func (r *Definitions) all() iter.Seq[serviceDefinition] {
	return func(f func(serviceDefinition) bool) {
		for _, sd := range r.services {
			if !f(sd) {
				return
			}
		}
	}
}
