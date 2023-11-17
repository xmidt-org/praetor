// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"errors"
	"sync"

	"github.com/hashicorp/consul/api"
)

var (
	ErrNoSuchServiceID = errors.New("That service id is not registered")
	ErrNoSuchCheckID   = errors.New("That check id is not registered")
)

// HealthStatus enumerates the allowed health statuses for consul checks.
type HealthStatus int

const (
	HealthAny HealthStatus = iota - 1
	HealthPassing
	HealthWarning
	HealthCritical
	HealthMaint
)

// StatusText returns the consul health status that should be passed
// to checks.
func (hs HealthStatus) StatusText() string {
	switch hs {
	case HealthPassing:
		return api.HealthPassing

	case HealthWarning:
		return api.HealthWarning

	case HealthCritical:
		return api.HealthCritical

	case HealthMaint:
		return api.HealthMaint

	default:
		return api.HealthCritical
	}
}

// FromHealthStatusText converts consul health status texts into praetor health statuses.
// Any unrecognized text results in HealthCritical.  The health Any status is honored,
// but many consul APIs do not accepts Any.  Callers should check the return of this function
// to make sure the status makes sense.
//
// Legacy consul health statuses are supported by this function.  For example, passing "pass"
// will result in HealthPassing.
func FromHealthStatusText(text string) HealthStatus {
	switch text {
	case api.HealthAny:
		return HealthAny

	case "pass", api.HealthPassing:
		return HealthPassing

	case "warn", api.HealthWarning:
		return HealthWarning

	case "fail", api.HealthCritical:
		return HealthCritical

	default:
		return HealthCritical
	}
}

// HealthEvent carries information about a health update.  One event
// will be sent for each check that was affected.  That means that when
// a service's overall health is updated, multiple events will be dispatched
// with the same service identifier but different check identifiers.
type HealthEvent struct {
	ServiceID ServiceID
	CheckID   CheckID
	State     HealthState
}

// HealthListener represents a sink for health events.
type HealthListener interface {
	OnHealthEvent(HealthEvent)
}

// HealthState is the full state associated with a consul check.
type HealthState struct {
	// Status reflects the healthiness of the check.  The default value
	// for this field is HealthPassing.
	Status HealthStatus

	// Notes contains optional, human-readable text associated with the
	// check.  This field is reflected in the consul check API.
	Notes string
}

// healthCheck holds state information about a single check.
type healthCheck struct {
	serviceID ServiceID
	checkID   CheckID
	state     HealthState
	listeners []HealthListener
}

func (hc *healthCheck) update(state HealthState) {
	hc.state = state
	for _, l := range hc.listeners {
		l.OnHealthEvent(HealthEvent{
			ServiceID: hc.serviceID,
			CheckID:   hc.checkID,
			State:     hc.state,
		})
	}
}

func (hc *healthCheck) addListener(l HealthListener) {
	hc.listeners = append(hc.listeners, l)
}

func (hc *healthCheck) removeListener(l HealthListener) {
	last := len(hc.listeners) - 1
	for i := 0; i <= last; i++ {
		if hc.listeners[i] == l {
			hc.listeners[i] = hc.listeners[last]
			hc.listeners[last] = nil
			hc.listeners = hc.listeners[:last]
			break
		}
	}
}

// healthChecks is a collection of healthCheck trackers.
type healthChecks []*healthCheck

// Health holds health information for registered services.  Implementations
// are safe for concurrent access.
//
// No overall or aggregate health state is kept.  Each check's state is kept
// separately.  Aggregating health into a single application or service state
// is left to clients.
type Health struct {
	lock     sync.RWMutex
	all      healthChecks
	checks   map[CheckID]*healthCheck
	services map[ServiceID]healthChecks
}

// GetCheck returns the current health state for a check.  If checkID is
// not registered, this method returns a critical HealthState along with
// ErrNoSuchCheckID.
func (h *Health) GetCheck(checkID CheckID) (HealthState, error) {
	defer h.lock.RUnlock()
	h.lock.RLock()

	check, exists := h.checks[checkID]
	if !exists {
		return HealthState{Status: HealthCritical}, ErrNoSuchCheckID
	}

	return check.state, nil
}

// Each applies a visitor function to each check's HealthState.  The check's
// associated service identifier is passed, which means the same service identifier
// may get passed more than once since consul services may have multiple checks
// per service.
//
// The visitor function is executed under a read lock.  Callers must take care
// not to block, otherwise health updates may get delayed.
func (h *Health) Each(f func(ServiceID, CheckID, HealthState)) {
	defer h.lock.RUnlock()
	h.lock.RLock()

	for _, hc := range h.all {
		f(hc.serviceID, hc.checkID, hc.state)
	}
}

// Set causes all checks for all services to be set to the given state.
func (h *Health) Set(state HealthState) {
	defer h.lock.Unlock()
	h.lock.Lock()

	for _, hc := range h.all {
		hc.state = state
	}
}

// SetService updates the health state for all checks associated with a given
// service identifier.  This method returns ErrNoSuchServiceID if serviceID
// was not registered.
func (h *Health) SetService(serviceID ServiceID, state HealthState) error {
	defer h.lock.Unlock()
	h.lock.Lock()

	checks, exists := h.services[serviceID]
	if !exists {
		return ErrNoSuchServiceID
	}

	for _, hc := range checks {
		hc.state = state
	}

	return nil
}

// SetCheck updates a single check's state.  This method returns ErrNoSuchCheckID
// if checkID was not registered.
func (h *Health) SetCheck(checkID CheckID, state HealthState) (err error) {
	defer h.lock.Unlock()
	h.lock.Lock()

	if check, exists := h.checks[checkID]; exists {
		check.state = state
	} else {
		err = ErrNoSuchCheckID
	}

	return
}

func (h *Health) AddListener(l HealthListener, checkIDs ...CheckID) (err error) {
	defer h.lock.Unlock()
	h.lock.Lock()

	switch {
	case len(checkIDs) == 0:
		for _, check := range h.all {
			check.addListener(l)
		}

	default:
		// check that all ids exist before adding anything
		checks := make(healthChecks, 0, len(checkIDs))
		for _, checkID := range checkIDs {
			check, exists := h.checks[checkID]
			if !exists {
				err = ErrNoSuchCheckID
				break
			}

			checks = append(checks, check)
		}

		if err == nil {
			for _, check := range checks {
				check.addListener(l)
			}
		}
	}

	return
}

// NewHealth constructs an initial Health from a set of registrations.  The returned
// Health will contain one (1) initial HealthState per check.  Services without checks
// will not be accessible.
func NewHealth(sr ServiceRegistrations) *Health {
	h := &Health{
		all:      make(healthChecks, sr.Len()),
		checks:   make(map[CheckID]*healthCheck, sr.Len()), // just an estimate
		services: make(map[ServiceID]healthChecks, sr.Len()),
	}

	sr.Each(func(serviceID ServiceID, reg ServiceRegistration) {
		for _, registeredCheck := range reg.Checks {
			check := &healthCheck{
				serviceID: serviceID,
				checkID:   CheckID(registeredCheck.CheckID),

				// the initial state of this check
				state: HealthState{
					Notes: registeredCheck.Notes,
				},
			}

			if len(registeredCheck.Status) > 0 {
				check.state.Status = FromHealthStatusText(registeredCheck.Status)
				if check.state.Status == HealthAny {
					check.state.Status = HealthPassing
				}
			}

			h.all = append(h.all, check)
			h.checks[check.checkID] = check
			h.services[check.serviceID] = append(h.services[serviceID], check)
		}
	})

	return h
}
