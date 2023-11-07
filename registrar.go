package praetor

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/consul/api"
	"github.com/xmidt-org/retry"
	"go.uber.org/fx"
	"go.uber.org/multierr"
)

var (
	ErrRegistered   = errors.New("That Registrar has already registered its services")
	ErrUnregistered = errors.New("That Registrar has no services registered")
)

// RegistrarEventType identifies the kind of Registrar event.
type RegistrarEventType int

const (
	// EventRegister is the kind of event that results from a
	// Registrar.Register call.
	EventRegister RegistrarEventType = iota

	// EventDeregister is the kind of event that results from
	// a Registrar.Deregister call.
	EventDeregister
)

// RegistrarEvent holds information about the state of a Registrar.
type RegistrarEvent struct {
	Type RegistrarEventType

	// Registrations is the bundle of service registrations known
	// to the Registrar that sent this event.
	Registrations ServiceRegistrations

	// Registered holds the service identifiers that should be considered
	// registered with consul.
	//
	// If Type is EventRegister, this field holds the service identifiers
	// that were successfully registered.
	//
	// If Type is EventDeregister, this field will be empty.
	Registered []ServiceID

	// Err is any error that occurred that halted the previous operation.
	Err error
}

// RegistrarListener is a sink for RegistrarEvents.
type RegistrarListener interface {
	OnRegistrarEvent(RegistrarEvent)
}

// AgentRegisterer is the strategy for registering a service with a consul Agent.
// The *api.Agent type implements this interface.
//
// A component of this type is created by Provide, and can be decorated via fx.Decorate.
type AgentRegisterer interface {
	ServiceRegisterOpts(*api.AgentServiceRegistration, api.ServiceRegisterOpts) error
	ServiceDeregisterOpts(string, *api.QueryOptions) error
}

// Registrar is implemented by components responsible for registering
// one or more consul services and for dispatching events to allow other
// parts of an application react to registration/deregistration.
type Registrar interface {
	// Register handles service registration for all services known to this
	// instance.  This method blocks until all registrations are complete
	// or there is an error.  If this method returns an error, Deregister should
	// be called to clean up any services that successfully registered.
	//
	// This method is idempotent.  If it is called on a Registrar that has its
	// services currently registered, it returns ErrRegistered.
	Register() error

	// Deregister handles deregistering all services known to this instance.
	// This method always deregisters all services, regardless of errors.  The returned
	// error will be an aggregate of any errors that occurred.
	//
	// This method is idempotent.  If it is called on a Registrar before a corresponding
	// Register call is made, it returns ErrUnregistered.
	Deregister() error

	// AddListener adds the given listener.  The new listener will immediately receive
	// a RegistrarEvent that reflects the current state of this Registrar.
	AddListener(RegistrarListener)

	// RemoveListener removes the given listener.
	RemoveListener(RegistrarListener)
}

const (
	registrarStateUnregistered uint32 = iota
	registrarStateRegistered
)

type agentRegistrar struct {
	registerer AgentRegisterer
	rcfg       retry.Config
	regs       ServiceRegistrations

	lock      sync.Mutex
	state     atomic.Uint32
	lastEvent RegistrarEvent
	listeners []RegistrarListener
}

func (ar *agentRegistrar) registerTask(reg ServiceRegistration) retry.Task[bool] {
	return func(ctx context.Context) (bool, error) {
		return true, ar.registerer.ServiceRegisterOpts(
			reg.asAgentServiceRegistration(),
			reg.RegisterOptions.WithContext(ctx),
		)
	}
}

func (ar *agentRegistrar) Register() error {
	if ar.state.Load() == registrarStateRegistered {
		return ErrRegistered
	}

	defer ar.lock.Unlock()
	ar.lock.Lock()

	if !ar.state.CompareAndSwap(registrarStateRegistered, registrarStateUnregistered) {
		return ErrRegistered
	}

	runner, err := retry.NewRunner(
		retry.WithPolicyFactory[bool](ar.rcfg),
	)

	if err != nil {
		return err
	}

	ar.lastEvent = RegistrarEvent{
		Type:          EventRegister,
		Registrations: ar.regs,
		Registered:    make([]ServiceID, 0, ar.regs.Len()),
	}

	ar.regs.Each(func(serviceID ServiceID, reg ServiceRegistration) {
		if _, taskErr := runner.Run(context.Background(), ar.registerTask(reg)); taskErr == nil {
			ar.lastEvent.Registered = append(ar.lastEvent.Registered, serviceID)
		} else {
			ar.lastEvent.Err = multierr.Append(ar.lastEvent.Err, taskErr)
		}
	})

	for _, l := range ar.listeners {
		l.OnRegistrarEvent(ar.lastEvent)
	}

	return ar.lastEvent.Err
}

func (ar *agentRegistrar) Deregister() error {
	if ar.state.Load() == registrarStateUnregistered {
		return ErrUnregistered
	}

	defer ar.lock.Unlock()
	ar.lock.Lock()

	if !ar.state.CompareAndSwap(registrarStateUnregistered, registrarStateRegistered) {
		return ErrUnregistered
	}

	// only deregister the services that were successfully registered
	registered := ar.lastEvent.Registered
	ar.lastEvent = RegistrarEvent{
		Type:          EventDeregister,
		Registrations: ar.regs,
		Registered:    nil, // when we're done, nothing will be registered
	}

	for _, serviceID := range registered {
		reg, _ := ar.regs.Get(serviceID)

		// clone the options, to avoid unintended modification
		opts := reg.DeregisterOptions

		ar.lastEvent.Err = multierr.Append(
			ar.lastEvent.Err,
			ar.registerer.ServiceDeregisterOpts(string(serviceID), &opts),
		)
	}

	for _, l := range ar.listeners {
		l.OnRegistrarEvent(ar.lastEvent)
	}

	return ar.lastEvent.Err
}

func (ar *agentRegistrar) AddListener(l RegistrarListener) {
	defer ar.lock.Unlock()
	ar.lock.Lock()

	ar.listeners = append(ar.listeners, l)
	l.OnRegistrarEvent(ar.lastEvent)
}

func (ar *agentRegistrar) RemoveListener(l RegistrarListener) {
	defer ar.lock.Unlock()
	ar.lock.Lock()

	last := len(ar.listeners) - 1
	for i := 0; i <= last; i++ {
		if ar.listeners[i] == l {
			ar.listeners[i] = ar.listeners[last]
			ar.listeners[last] = nil
			ar.listeners = ar.listeners[:last]
			return
		}
	}
}

// NewAgentRegistrar creates a Registrar that uses the consul agent to register
// services.  The given retry configuration is used to continue retrying
// registration according to a policy.
func NewAgentRegistrar(ar AgentRegisterer, rcfg retry.Config, regs ServiceRegistrations) Registrar {
	return &agentRegistrar{
		registerer: ar,
		rcfg:       rcfg,
		regs:       regs,
		lastEvent: RegistrarEvent{
			Type:          EventDeregister,
			Registrations: regs,
			Registered:    nil, // nothing is initially registered
		},
	}
}

// BindRegistrar binds the given Registrar to the enclosing application's lifecycle.
// On startup, Register is called.  On shutdown, Deregister is called.  If there
// is an error on startup, Deregister is also invoked for cleanup.
func BindRegistrar(r Registrar, lc fx.Lifecycle) {
	lc.Append(fx.StartStopHook(
		func() error {
			go func() {
				// TODO: How to report the error from Register properly
				if err := r.Register(); err != nil {
					r.Deregister()
				}
			}()

			return nil
		},
		r.Deregister,
	))
}
