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
	ErrRegistrarRegistered   = errors.New("that registrar has already been registered")
	ErrRegistrarDeregistered = errors.New("that registrar has already been deregistered")
)

type RegistrarOption interface {
	apply(*registrar) error
}

type registrarOptionFunc func(*registrar) error

func (f registrarOptionFunc) apply(r *registrar) error { return f(r) }

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

		if !used {
			err = fmt.Errorf("%T is not an agent", a)
		}

		return
	})
}

func WithAgentRegisterer(ar AgentRegisterer) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		r.ar = ar
		return nil
	})
}

func WithAgentDeregisterer(ad AgentDeregisterer) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		r.ad = ad
		return nil
	})
}

func WithRegisterRetry(d time.Duration) RegistrarOption {
	return registrarOptionFunc(func(r *registrar) error {
		r.retry = d
		return nil
	})
}

type Registrar interface {
	ServiceID() ServiceID
	Register(context.Context) error
	Deregister(context.Context) error
}

func NewRegistrar(reg api.AgentServiceRegistration, opts ...RegistrarOption) (Registrar, error) {
	r := &registrar{
		sid:      getServiceRegistrationID(reg),
		reg:      reg,
		newTimer: defaultNewTimer,
	}

	for _, o := range opts {
		if err := o.apply(r); err != nil {
			return nil, err
		}
	}

	if r.ar == nil || r.ad == nil {
		return nil, errors.New("no agent supplied")
	}

	if r.retry < 1 {
		r.retry = DefaultRegisterRetry
	}

	return r, nil
}

type registrar struct {
	ar       AgentRegisterer
	ad       AgentDeregisterer
	newTimer newTimer

	sid   ServiceID
	reg   api.AgentServiceRegistration
	retry time.Duration

	lock       sync.Mutex
	registered bool
}

func (r *registrar) ServiceID() ServiceID {
	return r.sid
}

func (r *registrar) Register(ctx context.Context) error {
	defer r.lock.Unlock()
	r.lock.Lock()

	if r.registered {
		return ErrRegistrarRegistered
	}

	opts := api.ServiceRegisterOpts{
		ReplaceExistingChecks: true,
	}.WithContext(ctx)

	for {
		err := r.ar.ServiceRegisterOpts(&r.reg, opts)
		if err == nil {
			r.registered = true
			return nil
		}

		ch, stop := r.newTimer(r.retry)
		select {
		case <-ctx.Done():
			stop()
			return err

		case <-ch:
			// continue retrying
		}
	}
}

func (r *registrar) Deregister(ctx context.Context) error {
	defer r.lock.Unlock()
	r.lock.Lock()

	if !r.registered {
		return ErrRegistrarDeregistered
	}

	err := r.ad.ServiceDeregisterOpts(string(r.sid), nil)
	r.registered = false
	return err
}
