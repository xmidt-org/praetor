package praetor

import (
	"context"
	"fmt"

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
}

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

	RegisterOptions   api.ServiceRegisterOpts `json:"registerOptions" yaml:"registerOptions"`
	DeregisterOptions api.QueryOptions        `json:"deregisterOptions" yaml:"deregisterOptions"`
}

// AsAgentServiceRegistration produces an *api.AgentServiceRegistration that corresponds to
// this registration.
func (sr ServiceRegistration) AsAgentServiceRegistration() (asr *api.AgentServiceRegistration) {
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

// Registrar is implemented by components responsible for registering
// one or more consul services.
type Registrar interface {
	// Register handles service registration for all services known to this
	// instance.  This method blocks until all registrations are complete
	// or there is an error.  If this method returns an error, Deregister should
	// be called to clean up any services that successfully registered.
	Register() error

	// Deregister handles deregistering all services known to this instance.
	// This method always deregisters all services, regardless of errors.  The returned
	// error will be an aggregate of any errors that occurred.
	Deregister() error
}

type nopRegistrar struct{}

func (nr nopRegistrar) Register() error   { return nil }
func (nr nopRegistrar) Deregister() error { return nil }

type agentRegistrar struct {
	registerer AgentRegisterer
	rcfg       retry.Config
	regs       map[string]ServiceRegistration
}

func (ar *agentRegistrar) registerTask(serviceID string, reg ServiceRegistration) retry.Task[bool] {
	return func(ctx context.Context) (bool, error) {
		var (
			opts = reg.RegisterOptions.WithContext(ctx)
			asr  = reg.AsAgentServiceRegistration()
		)

		err := ar.registerer.ServiceRegisterOpts(asr, opts)
		return true, err
	}
}

func (ar *agentRegistrar) Register() (err error) {
	var runner retry.Runner[bool]
	runner, err = retry.NewRunner(
		retry.WithPolicyFactory[bool](ar.rcfg),
	)

	for serviceID, reg := range ar.regs {
		_, taskErr := runner.Run(context.Background(), ar.registerTask(serviceID, reg))
		err = multierr.Append(err, taskErr)
	}

	return
}

func (ar *agentRegistrar) Deregister() (err error) {
	for serviceID, reg := range ar.regs {
		// clone the options, to avoid unintended modification
		opts := reg.DeregisterOptions
		err = multierr.Append(err, ar.registerer.ServiceDeregisterOpts(serviceID, &opts))
	}

	return
}

// NewAgentRegistrar creates a Registrar that uses the consul agent to register
// services.  The given retry configuration is used to continue retrying
// registration according to a policy.
func NewAgentRegistrar(ar AgentRegisterer, rcfg retry.Config, regs ...ServiceRegistration) (r Registrar, err error) {
	switch {
	case ar == nil:
		fallthrough

	case len(regs) == 0:
		r = nopRegistrar{}

	default:
		registrar := &agentRegistrar{
			registerer: ar,
			rcfg:       rcfg,
			regs:       make(map[string]ServiceRegistration, len(regs)),
		}

		for i, reg := range regs {
			if len(reg.Name) == 0 {
				err = multierr.Append(err, fmt.Errorf("No service name supplied for service registration #%d", i))
				continue
			}

			serviceID := reg.ID
			if len(serviceID) == 0 {
				serviceID = reg.Name
			}

			if _, exists := registrar.regs[serviceID]; exists {
				err = multierr.Append(err, fmt.Errorf("Duplicate service id: %s", serviceID))
				continue
			}

			registrar.regs[serviceID] = reg
		}

		if err == nil {
			r = registrar
		}
	}

	return
}

// BindRegistrar binds the given Registrar to the enclosing application's lifecycle.
// On startup, Register is called.  On shutdown, Deregister is called.  If there
// is an error on startup, Deregister is also invoked for cleanup.
func BindRegistrar(r Registrar, lc fx.Lifecycle) {
	lc.Append(fx.StartStopHook(
		func() (err error) {
			err = r.Register()
			if err != nil {
				r.Deregister() // ignore errors
			}

			return
		},
		r.Deregister,
	))
}
