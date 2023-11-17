// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"fmt"

	"github.com/hashicorp/consul/api"
	"go.uber.org/multierr"
)

// ServiceID is the type alias for a service's unique identifier
// within an Agent instance.
type ServiceID string

// CheckID is the type alias for a service check's unique identifier.
type CheckID string

// ServiceRegistration holds registration information for a single service.
type ServiceRegistration struct {
	ID                string                        `json:"id" yaml:"id"`
	Name              string                        `json:"name" yaml:"name"`
	Tags              []string                      `json:"tags" yaml:"tags"`
	Port              int                           `json:"port" yaml:"port"`
	Address           string                        `json:"address" yaml:"address"`
	SocketPath        string                        `json:"socketPath" yaml:"socketPath"`
	TaggedAddresses   map[string]api.ServiceAddress `json:"taggedAddresses" yaml:"taggedAddresses"`
	EnableTagOverride bool                          `json:"enableTagOverride" yaml:"enableTagOverride"`
	Meta              map[string]string             `json:"meta" yaml:"meta"`
	Checks            []api.AgentServiceCheck       `json:"checks" yaml:"checks"`

	Namespace string        `json:"namespace" yaml"namespace"`
	Partition string        `json:"partition" yaml:"partition"`
	Locality  *api.Locality `json:"locality" yaml:"locality"`

	// RegisterOptions holds the registration options to send to consul.  Note
	// that this package always set ReplaceExistingChecks to true so that any
	// checks will be replaced upon reregistration.
	RegisterOptions   api.ServiceRegisterOpts `json:"registerOptions" yaml:"registerOptions"`
	DeregisterOptions api.QueryOptions        `json:"deregisterOptions" yaml:"deregisterOptions"`
}

func (sr ServiceRegistration) serviceID() ServiceID {
	if len(sr.ID) > 0 {
		return ServiceID(sr.ID)
	}

	return ServiceID(sr.Name)
}

func (sr ServiceRegistration) asAgentServiceRegistration() (asr *api.AgentServiceRegistration) {
	asr = &api.AgentServiceRegistration{
		ID:                sr.ID,
		Name:              sr.Name,
		Tags:              sr.Tags,
		Port:              sr.Port,
		Address:           sr.Address,
		SocketPath:        sr.SocketPath,
		TaggedAddresses:   sr.TaggedAddresses,
		Meta:              sr.Meta,
		EnableTagOverride: sr.EnableTagOverride,
		Namespace:         sr.Namespace,
		Partition:         sr.Partition,
		Locality:          sr.Locality,
	}

	if len(sr.Checks) > 0 {
		asr.Checks = make(api.AgentServiceChecks, len(sr.Checks))
		for i := 0; i < len(asr.Checks); i++ {
			asr.Checks[i] = new(api.AgentServiceCheck)
			*asr.Checks[i] = sr.Checks[i]
		}
	}

	return
}

// ServiceRegistrations is an immutable bundle of ServiceRegistration objects.
// These registrations will be prevalidated and are ensured to have check
// identifiers assigned in a predictable, unique way.
type ServiceRegistrations struct {
	regs map[ServiceID]ServiceRegistration
}

// Len returns the number of registrations contained in this bundle.
func (sr ServiceRegistrations) Len() int {
	return len(sr.regs)
}

// Get retrieves the registration associated with the given id.  This method
// returns false to indicate that the given id was not present in this bundle.
func (sr ServiceRegistrations) Get(id ServiceID) (ServiceRegistration, bool) {
	v, ok := sr.regs[id]
	return v, ok
}

// Each applies a visitor function to each registration.  The visitor must
// not retain or modify the ServiceRegistration.
func (sr ServiceRegistrations) Each(f func(ServiceID, ServiceRegistration)) {
	for serviceID, reg := range sr.regs {
		f(serviceID, reg)
	}
}

// EachCheck applies a visitor function to each service check.  The visitor
// must not retain or modify the AgentServiceCheck.
func (sr ServiceRegistrations) EachCheck(f func(ServiceID, CheckID, api.AgentServiceCheck)) {
	for serviceID, reg := range sr.regs {
		for _, check := range reg.Checks {
			f(serviceID, CheckID(check.CheckID), check)
		}
	}
}

// NewServiceRegistrations produces an immutable bundle of registrations.  Basic validation is
// performed on the registrations, and any checks that are missing identifiers have a predictable,
// unique id assigned.
func NewServiceRegistrations(regs ...ServiceRegistration) (sr ServiceRegistrations, err error) {
	checks := make(map[CheckID]bool, len(regs))
	sr = ServiceRegistrations{
		regs: make(map[ServiceID]ServiceRegistration, len(regs)),
	}

	for i, reg := range regs {
		if len(reg.Name) == 0 {
			err = multierr.Append(err, fmt.Errorf("No service name for registration #%d", i))
			continue
		}

		serviceID := reg.serviceID()
		if _, exists := sr.regs[serviceID]; exists {
			err = multierr.Append(err, fmt.Errorf("Duplicate service ID: %s", serviceID))
			continue
		}

		for i, check := range reg.Checks {
			if len(check.CheckID) == 0 {
				check.CheckID = fmt.Sprintf("%s:check-%d", serviceID, i)
			}

			checkID := CheckID(check.CheckID)
			if checks[checkID] {
				err = multierr.Append(err, fmt.Errorf("Duplicate check ID: %s", checkID))
			} else {
				checks[checkID] = true
			}
		}

		// always turn on ReplaceExistingChecks, especially since we may have
		// generated check identifiers.
		reg.RegisterOptions.ReplaceExistingChecks = true

		sr.regs[serviceID] = reg
	}

	return
}
