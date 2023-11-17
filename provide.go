// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"github.com/hashicorp/consul/api"
	"go.uber.org/fx"
)

func provideClient(cfg api.Config) (*api.Client, error) {
	return api.NewClient(&cfg)
}

func provideAgent(c *api.Client) *api.Agent {
	return c.Agent()
}

func provideCatalog(c *api.Client) *api.Catalog {
	return c.Catalog()
}

func provideHealth(c *api.Client) *api.Health {
	return c.Health()
}

func provideKV(c *api.Client) *api.KV {
	return c.KV()
}

// Provide bootstraps a consul *api.Client from an option api.Config.
// If no api.Config is supplied, a default client is created.  Note that
// the api.Config is used by value, not as a pointer.
//
// A few of the most commonly used client endpoints are provided as components:
//
//   - *api.Agent
//   - *api.Catalog
//   - *api.Health
//   - *api.KV
func Provide() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				provideClient,
				fx.ParamTags(`optional:"true"`),
			),
			provideAgent,
			provideCatalog,
			provideHealth,
			provideKV,
		),
	)
}
