package praetor

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

// HealthState is the full state associated with a consul check.
type HealthState struct {
	// Status reflects the healthiness of the check.  The default value
	// for this field is HealthPassing.
	Status HealthStatus

	// Notes contains optional, human-readable text associated with the
	// check.  This field is reflected in the consul check API.
	Notes string
}

// Health holds health information for registered services.  Implementations
// are safe for concurrent access.
//
// No overall or aggregate health state is kept.  Each check's state is kept
// separately.  Aggregating health into a single application or service state
// is left to clients.
type Health struct {
	lock     sync.RWMutex
	checks   map[CheckID]HealthState
	services map[ServiceID][]CheckID
}

// GetCheck returns the current health state for a check.  If checkID is
// not registered, this method returns a critical HealthState along with
// ErrNoSuchCheckID.
func (h *Health) GetCheck(checkID CheckID) (HealthState, error) {
	defer h.lock.RUnlock()
	h.lock.RLock()

	state, exists := h.checks[checkID]
	if !exists {
		return HealthState{Status: HealthCritical}, ErrNoSuchCheckID
	}

	return state, nil
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

	for serviceID, checkIDs := range h.services {
		for _, checkID := range checkIDs {
			f(serviceID, checkID, h.checks[checkID])
		}
	}
}

// Set causes all checks for all services to be set to the given state.
func (h *Health) Set(hs HealthState) {
	defer h.lock.Unlock()
	h.lock.Lock()

	for checkID := range h.checks {
		h.checks[checkID] = hs
	}
}

// SetService updates the health state for all checks associated with a given
// service identifier.  This method returns ErrNoSuchServiceID if serviceID
// was not registered.
func (h *Health) SetService(serviceID ServiceID, hs HealthState) error {
	defer h.lock.Unlock()
	h.lock.Lock()

	checkIDs, exists := h.services[serviceID]
	if !exists {
		return ErrNoSuchServiceID
	}

	for _, checkID := range checkIDs {
		h.checks[checkID] = hs
	}

	return nil
}

// SetCheck updates a single check's state.  This method returns ErrNoSuchCheckID
// if checkID was not registered.
func (h *Health) SetCheck(checkID CheckID, hs HealthState) error {
	defer h.lock.Unlock()
	h.lock.Lock()

	if _, exists := h.checks[checkID]; !exists {
		return ErrNoSuchCheckID
	}

	h.checks[checkID] = hs
	return nil
}

// NewHealth constructs an initial Health from a set of registrations.  The returned
// Health will contain one (1) initial HealthState per check.  Services without checks
// will not be accessible.
func NewHealth(sr ServiceRegistrations) *Health {
	h := &Health{
		checks:   make(map[CheckID]HealthState, sr.Len()), // just an estimate
		services: make(map[ServiceID][]CheckID, sr.Len()),
	}

	sr.Each(func(serviceID ServiceID, reg ServiceRegistration) {
		for _, check := range reg.Checks {
			checkID := CheckID(check.CheckID)
			initial := HealthState{
				Notes: check.Notes,
			}

			if len(check.Status) > 0 {
				initial.Status = FromHealthStatusText(check.Status)
				if initial.Status == HealthAny {
					initial.Status = HealthPassing
				}
			}

			h.checks[checkID] = initial
			h.services[serviceID] = append(h.services[serviceID], checkID)
		}
	})

	return h
}
