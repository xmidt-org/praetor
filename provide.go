// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"github.com/hashicorp/consul/api"
	"go.uber.org/fx"
)

func Provide() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(cfg Config) api.Config {
					return cfg.Client
				},
				fx.ParamTags(`optional:"true"`),
			),
			fx.Annotate(
				func(cfg Config) RegistrationConfig {
					return cfg.Registration
				},
				fx.ParamTags(`optional:"true"`),
			),
			func(cfg api.Config) (*api.Client, error) {
				return api.NewClient(&cfg)
			},
			func(c *api.Client) *api.Agent {
				return c.Agent()
			},
			func(a *api.Agent) AgentRegisterer {
				return a
			},
			func(ar AgentRegisterer, rc RegistrationConfig, lc fx.Lifecycle) (r Registrar, err error) {
				r, err = NewAgentRegistrar(
					ar,
					rc.Retry,
					rc.Services...,
				)

				if err == nil {
					BindRegistrar(r, lc)
				}

				return
			},
		),
		fx.Invoke(
			func(Registrar) {},
		),
	)
}
