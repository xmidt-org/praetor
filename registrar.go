package praetor

import (
	"context"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/xmidt-org/retry"
	"go.uber.org/fx"
	"go.uber.org/multierr"
)

// AgentRegisterer is the strategy for registering a service with a consul Agent.
// The *api.Agent type implements this interface.
//
// A component of this type is created by Provide, and can be decorated via fx.Decorate.
type AgentRegisterer interface {
	ServiceRegisterOpts(*api.AgentServiceRegistration, api.ServiceRegisterOpts) error
	ServiceDeregisterOpts(string, *api.QueryOptions) error
	UpdateTTLOpts(checkID, output, status string, q *api.QueryOptions) error
}

// Registrar is implemented by components responsible for registering
// one or more consul services and maintaining a central spot for service
// check health state to report to consul.
type Registrar interface {
	// Register handles service registration for all services known to this
	// instance.  This method blocks until all registrations are complete
	// or there is an error.  If this method returns an error, Deregister should
	// be called to clean up any services that successfully registered.
	//
	// If any services had TTL checks, this method will start goroutines to update
	// those checks.
	Register() error

	// Deregister handles deregistering all services known to this instance.
	// This method always deregisters all services, regardless of errors.  The returned
	// error will be an aggregate of any errors that occurred.
	//
	// Any background goroutines started by Register will be shutdown by this method.
	Deregister() error
}

type nopRegistrar struct{}

func (nr nopRegistrar) Register() error   { return nil }
func (nr nopRegistrar) Deregister() error { return nil }

type agentRegistrar struct {
	registerer AgentRegisterer
	rcfg       retry.Config
	regs       ServiceRegistrations

	lock sync.RWMutex
}

func (ar *agentRegistrar) registerTask(reg ServiceRegistration) retry.Task[bool] {
	return func(ctx context.Context) (bool, error) {
		return true, ar.registerer.ServiceRegisterOpts(
			reg.asAgentServiceRegistration(),
			reg.RegisterOptions.WithContext(ctx),
		)
	}
}

func (ar *agentRegistrar) Register() (err error) {
	defer ar.lock.Unlock()
	ar.lock.Lock()

	var runner retry.Runner[bool]
	runner, err = retry.NewRunner(
		retry.WithPolicyFactory[bool](ar.rcfg),
	)

	ar.regs.Each(func(_ ServiceID, reg ServiceRegistration) {
		_, taskErr := runner.Run(context.Background(), ar.registerTask(reg))
		err = multierr.Append(err, taskErr)
	})

	if err == nil {
		// TODO
	}

	return
}

func (ar *agentRegistrar) Deregister() (err error) {
	ar.regs.Each(func(serviceID ServiceID, reg ServiceRegistration) {
		// clone the options, to avoid unintended modification
		opts := reg.DeregisterOptions
		err = multierr.Append(err,
			ar.registerer.ServiceDeregisterOpts(string(serviceID), &opts),
		)
	})

	return
}

// NewAgentRegistrar creates a Registrar that uses the consul agent to register
// services.  The given retry configuration is used to continue retrying
// registration according to a policy.
func NewAgentRegistrar(ar AgentRegisterer, rcfg retry.Config, regs ServiceRegistrations) Registrar {
	if ar == nil || regs.Len() == 0 {
		return nopRegistrar{}
	}

	return &agentRegistrar{
		registerer: ar,
		rcfg:       rcfg,
		regs:       regs,
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
