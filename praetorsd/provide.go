// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"errors"

	"github.com/hashicorp/consul/api"
	"go.uber.org/fx"
)

var (
	ErrNoClientOrAgent = errors.New("A consul client or agent must be supplied")
)

// ConsulDependences holds the consul components that need to be supplied for this
// package's bootstrapping to function.
type ConsulDependencies struct {
	fx.In

	// Client is the consul *api.Client within the enclosing fx application.
	// This component is optional to allow an application to provide an agent
	// directly instead.  The Agent endpoint of this Client is used for
	// service registration and discovery.
	//
	// Either a Client or Agent must be supplied.  If both are supplied,
	// this package takes the Agent.
	Client *api.Client `optional:"true"`

	// Agent is the consul *api.Agent endpoint within the enclosing fx application.
	// This component is optional to allow an application so provide a client
	// instead.
	//
	// Either a Client or Agent must be supplied.  If this field is supplied,
	// it is always used regardless of the Client field.
	Agent *api.Agent `optional:"true"`
}

func (cd ConsulDependencies) agent() (a *api.Agent, err error) {
	switch {
	case cd.Agent != nil:
		a = cd.Agent

	case cd.Client != nil:
		a = cd.Client.Agent()

	default:
		err = ErrNoClientOrAgent
	}

	return
}

func provideAgentRegisterer(in ConsulDependencies) (a AgentRegisterer, err error) {
	return in.agent()
}

func provideServiceRegistrations(cfg RegistrationConfig) (ServiceRegistrations, error) {
	return NewServiceRegistrations(cfg.Services...)
}

func provideRegistrar(ar AgentRegisterer, cfg RegistrationConfig, sr ServiceRegistrations) Registrar {
	return NewAgentRegistrar(
		ar,
		cfg.Retry,
		sr,
	)
}

// Provide bootstraps consul service discovery within an enclosing
// fx application.
func Provide() fx.Option {
	return fx.Options(
		fx.Provide(
			provideAgentRegisterer,
			fx.Annotate(
				provideServiceRegistrations,
				fx.ParamTags(`optional:"true"`),
			),
			fx.Annotate(
				provideRegistrar,
				fx.ParamTags(
					"",
					`optional:"true"`,
					"",
				),
			),
		),
	)
}
