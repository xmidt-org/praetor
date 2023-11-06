// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"github.com/hashicorp/consul/api"
	"go.uber.org/fx"
)

func provideClientConfig(cfg Config) api.Config { return cfg.Client }

func provideRegistrationConfig(cfg Config) RegistrationConfig { return cfg.Registration }

func provideServiceRegistrations(cfg RegistrationConfig) (ServiceRegistrations, error) {
	return NewServiceRegistrations(cfg.Services...)
}

func provideClient(cfg api.Config) (*api.Client, error) {
	return api.NewClient(&cfg)
}

func provideAgent(c *api.Client) *api.Agent { return c.Agent() }

func provideAgentRegisterer(a *api.Agent) AgentRegisterer { return a }

func provideAgentRegistrar(ar AgentRegisterer, rc RegistrationConfig, regs ServiceRegistrations, lc fx.Lifecycle) Registrar {
	r := NewAgentRegistrar(ar, rc.Retry, regs)
	BindRegistrar(r, lc)

	return r
}

func Provide() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				provideClientConfig,
				fx.ParamTags(`optional:"true"`),
			),
			provideRegistrationConfig,
			provideServiceRegistrations,
			provideClient,
			provideAgent,
			provideAgentRegisterer,
			provideAgentRegistrar,
		),
		fx.Invoke(
			func(Registrar) {},
		),
	)
}
