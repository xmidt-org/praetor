// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/consul/api"
	"go.uber.org/multierr"
)

var (
	// ErrRegistrarNoAgent indicates that no Agent or one of the Agent interface was not configured
	// in the options for a Registrar.
	ErrRegistrarNoAgent = errors.New("an agent was not fully configured for a registrar")

	// ErrRegistrarServiceID indicates that an attempt was made to register a service that either
	// did not have a service ID or the service ID was not unique.
	ErrRegistrarServiceID = errors.New("registered services must have a unique service ID")

	// ErrRegistrarCheckID indicates that one or more services either did not have check IDs or
	// a check ID was not unique.  Check IDs must be unique across an entire Registrar, not just
	// within a single service.
	ErrRegistrarCheckID = errors.New("registered service checks must have unique check IDs")

	// ErrRegistrarNoCheck indicates that no service check existed with the requested id.
	ErrRegistrarNoCheck = errors.New("no check exists with that id")

	// ErrRegistrarNoService indicates that no service existed with the requested id.
	ErrRegistrarNoService = errors.New("no service exists with that id")

	// ErrRegistrarStarted indicates Registrar.Register was called on a Registrar that
	// had already been running.
	ErrRegistrarStarted = errors.New("that registrar has already been started")

	// ErrRegistrarStopped indicates Registrar.Deregister was called on a Registrar that
	// had already been stopped.
	ErrRegistrarStopped = errors.New("that registrar has already been stopped")

	// ErrRegistrarJitter indicates an invalid TTL jitter value.
	ErrRegistrarJitter = errors.New("a TTL jitter must be in the half open range [0.0, 1.0)")
)

// registrarCheck holds information about a single check for a managed service.
type registrarCheck struct {
	check   api.AgentServiceCheck
	service api.AgentServiceRegistration
	opts    api.QueryOptions

	newTimer func(time.Duration) (<-chan time.Time, func() bool)
	interval time.Duration
	jitter   float64
	state    atomic.Value
}

func newRegistrarCheck(service api.AgentServiceRegistration, check api.AgentServiceCheck) (rc *registrarCheck, err error) {
	rc = &registrarCheck{
		service: service,
		check:   check,
		newTimer: func(d time.Duration) (<-chan time.Time, func() bool) {
			timer := time.NewTimer(d)
			return timer.C, timer.Stop
		},
	}

	// initial state
	rc.updateState(State{})

	if len(rc.check.TTL) > 0 {
		rc.interval, err = time.ParseDuration(rc.check.TTL)
	}

	return
}

// updateState updates the health state for this check.
func (rc *registrarCheck) updateState(s State) {
	rc.state.Store(s)
}

// nextTimer returns a closure that chooses new timer intervals based on
// this check's interval and jitter. This method returns nil if this check
// does not represent a TTL check.
func (rc *registrarCheck) nextTimer() func() (<-chan time.Time, func() bool) {
	if rc.interval <= 0 {
		return nil
	}

	return func() (<-chan time.Time, func() bool) {
		interval := rc.interval
		if rc.jitter > 0.0 {
			jitterRange := rand.N(
				time.Duration(rc.jitter * float64(rc.interval)),
			)

			interval -= jitterRange
		}

		return rc.newTimer(interval)
	}
}

// ttlFunc returns a closure that can be used to update the TTL state.
// This method returns nil if this check does not have a TTL interval.
func (rc *registrarCheck) ttlFunc(ctx context.Context, updater TTLUpdater) func() error {
	if rc.interval <= 0 {
		return nil
	}

	opts := rc.opts.WithContext(ctx)
	return func() error {
		cs := rc.state.Load().(State)
		return updater.UpdateTTLOpts(rc.check.CheckID, cs.Output, cs.Status.String(), opts)
	}
}

// startTTL starts a background goroutine that updates any TTL. The goroutine will
// exit when the given context is canceled.
//
// If this check does not require a TTL update, this method does nothing.
func (rc *registrarCheck) startTTL(ctx context.Context, updater TTLUpdater) {
	var (
		ttlFunc   = rc.ttlFunc(ctx, updater)
		nextTimer = rc.nextTimer()
	)

	if ttlFunc != nil && nextTimer != nil {
		go func(ctx context.Context) {
			for ctx.Err() == nil {
				// TODO: additional error handling
				err := ttlFunc()
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}

				ch, stop := nextTimer()
				select {
				case <-ctx.Done():
					stop()
					return

				case <-ch:
					// continue
				}
			}
		}(ctx)
	}
}

// registrarService holds information about services for a Registrar.
// Each service can be associated with their own options.
type registrarService struct {
	service        api.AgentServiceRegistration
	registerOpts   api.ServiceRegisterOpts
	deregisterOpts api.QueryOptions
	checks         map[string]*registrarCheck
}

func newRegistrarService(service api.AgentServiceRegistration) (rs *registrarService, err error) {
	rs = &registrarService{
		service: service,
		registerOpts: api.ServiceRegisterOpts{
			ReplaceExistingChecks: true,
		},
		checks: make(map[string]*registrarCheck, ServiceRegistrationChecksLen(service)),
	}

	for _, check := range ServiceRegistrationChecks(rs.service) {
		var rc *registrarCheck

		if len(check.CheckID) == 0 {
			err = ErrRegistrarCheckID
		} else if _, exists := rs.checks[check.CheckID]; exists {
			err = ErrRegistrarCheckID
		}

		if err == nil {
			rc, err = newRegistrarCheck(service, check)
		}

		if err == nil {
			rs.checks[rc.check.CheckID] = rc
		}
	}

	return
}

// updateState updates the check state for all checks for this service.
func (rs *registrarService) updateState(s State) {
	for _, rc := range rs.checks {
		rc.updateState(s)
	}
}

// RegistrarServiceOption represents a configurable option for each
// service that a Registrar manages.
type RegistrarServiceOption interface {
	applyToService(*registrarService) error
}

type registrarServiceOptionFunc func(*registrarService) error

func (o registrarServiceOptionFunc) applyToService(rs *registrarService) error { return o(rs) }

func WithRegisterOpts(opts api.ServiceRegisterOpts) RegistrarServiceOption {
	return registrarServiceOptionFunc(func(rs *registrarService) error {
		rs.registerOpts = opts
		return nil
	})
}

func WithDeregisterOpts(opts api.QueryOptions) RegistrarServiceOption {
	return registrarServiceOptionFunc(func(rs *registrarService) error {
		rs.deregisterOpts = opts
		return nil
	})
}

func WithTTLOpts(opts api.QueryOptions) RegistrarServiceOption {
	return registrarServiceOptionFunc(func(rs *registrarService) error {
		for _, rc := range rs.checks {
			rc.opts = opts
		}

		return nil
	})
}

func WithTTLJitter(j float64) RegistrarServiceOption {
	return registrarServiceOptionFunc(func(rs *registrarService) error {
		if j < 0.0 || j >= 1.0 {
			return ErrRegistrarJitter
		}

		for _, rc := range rs.checks {
			rc.jitter = j
		}

		return nil
	})
}

// RegistrarOption is a configurable option for a Registrar.
type RegistrarOption interface {
	applyToRegistrar(*registrar) error
}

type registrarOptionFunc func(*registrar) error

func (o registrarOptionFunc) applyToRegistrar(r *registrar) error { return o(r) }

// WithRegisterer sets the AgentRegisterer that is used to perform the
// low-level service registration with consul.
func WithRegisterer(a AgentRegisterer) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		r.registerer = a
		return nil
	})
}

// WithRegisterer sets the AgentRegisterer that is used to perform the
// low-level service deregistration with consul.
func WithDeregisterer(a AgentDeregisterer) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		r.deregisterer = a
		return nil
	})
}

// WithRegisterer sets the AgentRegisterer that is used to perform the
// low-level TTL updates with consul.
func WithTTLUpdater(u TTLUpdater) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		r.ttlUpdater = u
		return nil
	})
}

// WithAgent is a one-stop setup for a Registrar's agent. The given value
// must implement one or more of the agent interfaces in this package, or
// an error is returned. An *api.Agent may be passed to this option, as can
// anything that decorates an agent.
func WithAgent(a any) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		count := 0

		if v, ok := a.(AgentRegisterer); ok {
			if err := WithRegisterer(v).applyToRegistrar(r); err != nil {
				return err
			}

			count++
		}

		if v, ok := a.(AgentDeregisterer); ok {
			if err := WithDeregisterer(v).applyToRegistrar(r); err != nil {
				return err
			}

			count++
		}

		if v, ok := a.(TTLUpdater); ok {
			if err := WithTTLUpdater(v).applyToRegistrar(r); err != nil {
				return err
			}

			count++
		}

		if count == 0 {
			return fmt.Errorf("%T does not implement any agent interfaces", a)
		}

		return nil
	})
}

// WithService adds a service to the Registrar. The AgentServiceRegistration must have a service ID that is
// unique, and each check must have a unique check ID.
func WithService(service api.AgentServiceRegistration, opts ...RegistrarServiceOption) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		if len(service.ID) == 0 {
			return ErrRegistrarServiceID
		} else if _, exists := r.services[service.ID]; exists {
			return ErrRegistrarServiceID
		}

		rs, err := newRegistrarService(service)
		if err != nil {
			return err
		}

		for _, rc := range rs.checks {
			if _, exists := r.checks[rc.check.CheckID]; exists {
				return ErrRegistrarCheckID
			}
		}

		for _, o := range opts {
			if err := o.applyToService(rs); err != nil {
				return err
			}
		}

		r.services[service.ID] = rs
		maps.Copy(r.checks, rs.checks)
		return nil
	})
}

// Registrar is a service that handles service registration and deregistration.
type Registrar interface {
	// Register submits each service registration to consul and starts any background
	// goroutines for updating TTLs. If the supplied context is canceled or expires,
	// registration will be interrupted.
	//
	// If this method is called more than once without Deregister in between calls,
	// this method returns ErrRegistrarStarted.
	Register(context.Context) error

	// Deregister stops any background goroutines and deregisters all services.
	// If the supplied context is canceled or expires, deregistration will be
	// interrupted.
	//
	// If this method is called more than once without Register in between calls,
	// this method returns ErrRegistrarStopped.
	Deregister(context.Context) error

	// SetState updates the state for all checks. If this registrar has no
	// checks, this method does nothing.
	SetState(State)

	// SetServiceState updates the state for a service's checks. If the given service
	// is unknown to this Registrar, ErrRegistrarNoService is returned.
	SetServiceState(serviceID string, s State) error

	// SetCheckState updates the state of a single check. If the given check is unknown
	// to this Registrar, ErrRegistrarNoCheck is returned.
	//
	// A check's state may be set at anytime, regardless of whether Register
	// has been called.
	SetCheckState(checkID string, s State) error
}

// NewRegistrar creates a Registrar using a supplied set of options. The set
// of managed services is fixed and immutable after creation.
//
// If no services are registered, the returned Registrar does nothing.
func NewRegistrar(opts ...RegistrarOption) (Registrar, error) {
	r := &registrar{
		services: make(map[string]*registrarService),
		checks:   make(map[string]*registrarCheck),
	}

	for _, o := range opts {
		if err := o.applyToRegistrar(r); err != nil {
			return nil, err
		}
	}

	// an agent must have been configured
	if r.registerer == nil || r.deregisterer == nil || r.ttlUpdater == nil {
		return nil, ErrRegistrarNoAgent
	}

	return r, nil
}

type registrar struct {
	registerer   AgentRegisterer
	deregisterer AgentDeregisterer
	ttlUpdater   TTLUpdater
	services     map[string]*registrarService
	checks       map[string]*registrarCheck

	registerLock sync.Mutex
	cancel       context.CancelFunc
}

func (r *registrar) Register(ctx context.Context) (err error) {
	defer r.registerLock.Unlock()
	r.registerLock.Lock()

	var taskCtx context.Context
	if r.cancel != nil {
		err = ErrRegistrarStarted
	}

	if err == nil {
		taskCtx, r.cancel = context.WithCancel(context.Background())
		for _, rs := range r.services {
			err = multierr.Append(err, r.registerer.ServiceRegisterOpts(
				&rs.service,
				rs.registerOpts.WithContext(ctx),
			))
		}
	}

	if err == nil {
		for _, rc := range r.checks {
			rc.startTTL(taskCtx, r.ttlUpdater)
		}
	}

	return
}

func (r *registrar) Deregister(ctx context.Context) (err error) {
	defer r.registerLock.Unlock()
	r.registerLock.Lock()

	if r.cancel == nil {
		err = ErrRegistrarStopped
	}

	if err == nil {
		r.cancel()
		r.cancel = nil

		for _, rs := range r.services {
			err = multierr.Append(err, r.deregisterer.ServiceDeregisterOpts(
				rs.service.ID,
				rs.deregisterOpts.WithContext(ctx),
			))
		}
	}

	return
}

func (r *registrar) SetState(s State) {
	for _, rc := range r.checks {
		rc.updateState(s)
	}
}

func (r *registrar) SetServiceState(serviceID string, s State) (err error) {
	if rs, exists := r.services[serviceID]; exists {
		rs.updateState(s)
	} else {
		err = ErrRegistrarNoService
	}

	return
}

func (r *registrar) SetCheckState(checkID string, s State) (err error) {
	if rc, exists := r.checks[checkID]; exists {
		rc.updateState(s)
	} else {
		err = ErrRegistrarNoCheck
	}

	return
}
