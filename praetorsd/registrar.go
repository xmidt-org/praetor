// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

const (
	// DefaultRegisterRetry is the default interval between attempts to register a service.
	DefaultRegisterRetry = 10 * time.Second
)

var (
	// ErrRegistrarRegistered is returned by Registrar.Register if that Registrar has already
	// registered its service.
	ErrRegistrarRegistered = errors.New("that registrar has already been registered")

	// ErrRegistrarDeregistered is returned by Registrar.Deregister if that Registrar has already
	// deregistered its service.
	ErrRegistrarDeregistered = errors.New("that registrar has already been deregistered")
)

// RegistrarOption is a configurable option for creating a single Registrar.
type RegistrarOption interface {
	apply(*registrar) error
}

type registrarOptionFunc func(*registrar) error

func (f registrarOptionFunc) apply(r *registrar) error { return f(r) }

// WithAgent sets the consul agent this Registrar will use. The given object
// must implement at least (1) of the agent interfaces in this package.
func WithAgent(a any) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) (err error) {
		used := false
		if ar, ok := a.(AgentRegisterer); ok {
			used = true
			r.ar = ar
		}

		if ad, ok := a.(AgentDeregisterer); ok {
			used = true
			r.ad = ad
		}

		if tu, ok := a.(TTLUpdater); ok {
			used = true
			r.tu = tu
		}

		if !used {
			err = fmt.Errorf("%T is not an agent", a)
		}

		return
	})
}

// WithAgentRegisterer sets the AgentRegisterer component used by
// Registrar.Register.
func WithAgentRegisterer(ar AgentRegisterer) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		r.ar = ar
		return nil
	})
}

// WithAgentRegisterer sets the AgentDeregisterer component used by
// Registrar.Deregister.
func WithAgentDeregisterer(ad AgentDeregisterer) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		r.ad = ad
		return nil
	})
}

// WithTTLUpdater sets the TTLUpdater used by any TTL tasks that are
// spawned during Registrar.Register.
//
// NOTE: A TTLUpdater is required even if the registered service does
// not define any TTL checks.
func WithTTLUpdater(tu TTLUpdater) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		r.tu = tu
		return nil
	})
}

// WithRegisterRetry sets the interval for retrying a service's registration.
// If unset, this value defaults to DefaultRegisterRetry.
func WithRegisterRetry(d time.Duration) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		r.registerRetry = d
		return nil
	})
}

// WithInitialState sets the initial health state when this service is registered.
func WithInitialState(initial State) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		r.stateAccessor = newStateAccessor(initial)
		return nil
	})
}

// Registrar manages the registration lifecycle for a single service registered with consul.
// A Registrar handles registering the service, deregistering it, and spawning background
// tasks to update any TTL with the registrar's state.
type Registrar interface {
	StateAccessor

	// ServiceID is the unique service identifier for the service managed
	// by this Registrar. This value will never be empty.
	ServiceID() ServiceID

	// Register informs consul about the underlying service. If any TTL checks are defined
	// by the underlying api.AgentServiceRegistration, one background task per TTL check is
	// spawned that updates consul with the State() value in this same Registrar.
	//
	// Register is idempotent. It will return ErrRegistrarRegistered if this Registrar
	// is currently managing a registered service.
	//
	// This method is atomic and may be called at any time.
	Register(context.Context) error

	// Deregister informs consul that the underlying service should be removed. Any background
	// TTL check tasks are stopped.
	//
	// Deregister is idempotent. It will return ErrRegistrarDeregistered if this Registrar
	// is not currently managing a registered service.
	//
	// This method is atomic and may be called at any time.
	Deregister(context.Context) error
}

type registrar struct {
	*stateAccessor

	ar       AgentRegisterer
	ad       AgentDeregisterer
	tu       TTLUpdater
	newTimer newTimer

	def           serviceDefinition
	registerRetry time.Duration

	lock      sync.Mutex
	ttlCancel context.CancelFunc
}

// newRegistrar constructs a single registrar that manages the lifecycle of
// one defined service.
func newRegistrar(def serviceDefinition, opts ...RegistrarOption) (*registrar, error) {
	r := &registrar{
		def:      def,
		newTimer: defaultNewTimer,
	}

	for _, o := range opts {
		if err := o.apply(r); err != nil {
			return nil, err
		}
	}

	if r.ar == nil || r.ad == nil || r.tu == nil {
		return nil, errors.New("no agent supplied")
	}

	if r.registerRetry < 1 {
		r.registerRetry = DefaultRegisterRetry
	}

	if r.stateAccessor == nil {
		r.stateAccessor = newStateAccessor(State{Status: Passing})
	}

	return r, nil
}

func (r *registrar) ServiceID() ServiceID {
	return r.def.id
}

func (r *registrar) Register(ctx context.Context) error {
	defer r.lock.Unlock()
	r.lock.Lock()

	if r.ttlCancel != nil {
		return ErrRegistrarRegistered
	}

	opts := api.ServiceRegisterOpts{
		ReplaceExistingChecks: true,
	}.WithContext(ctx)

	for {
		err := r.ar.ServiceRegisterOpts(&r.def.registration, opts)
		if err == nil {
			break
		}

		ch, stop := r.newTimer(r.registerRetry)
		select {
		case <-ctx.Done():
			stop()
			return err

		case <-ch:
			// continue retrying
		}
	}

	var ttlCtx context.Context
	ttlCtx, r.ttlCancel = context.WithCancel(context.Background())
	for _, def := range r.def.ttls {
		t := &ttl{
			updater:  r.tu,
			def:      def,
			newTimer: r.newTimer,
			state:    r.stateAccessor,
		}

		go t.run(ttlCtx)
	}

	return nil
}

func (r *registrar) Deregister(ctx context.Context) error {
	defer r.lock.Unlock()
	r.lock.Lock()

	if r.ttlCancel == nil {
		return ErrRegistrarDeregistered
	}

	r.ttlCancel()
	r.ttlCancel = nil
	return r.ad.ServiceDeregisterOpts(string(r.def.id), nil)
}
