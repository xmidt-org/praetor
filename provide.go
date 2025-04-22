// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"github.com/hashicorp/consul/api"
	"go.uber.org/fx"
)

func newClient(acfg api.Config) (*api.Client, error) {
	return api.NewClient(&acfg)
}

func newAgent(c *api.Client) *api.Agent {
	return c.Agent()
}

func newCatalog(c *api.Client) *api.Catalog {
	return c.Catalog()
}

func newHealth(c *api.Client) *api.Health {
	return c.Health()
}

func newKV(c *api.Client) *api.KV {
	return c.KV()
}

// Provide sets up the dependency injection infrastructure for Consul.
//
// An api.Config may be present in the application.  If so, that will be used
// to construct the consul agent.  Otherwise, an empty api.Config will be used.
//
// The following components are emitted by this provider:
//
//   - *api.Client
//   - *api.Agent
//   - *api.Catalog
//   - *api.Health
//   - *api.KV
func Provide() fx.Option {
	return fx.Provide(
		fx.Annotate(
			newClient,
			fx.ParamTags(`optional:"true"`),
		),
		newAgent,
		newCatalog,
		newHealth,
		newKV,
	)
}

// ProvideConfig uses the praetor Config object in this package to bootstrap an api.Config.
// The praetor Config is optional, and if not present a default api.Config will be created.
func ProvideConfig() fx.Option {
	return fx.Provide(
		fx.Annotate(
			newAPIConfig,
			fx.ParamTags(`optional:"true"`),
		),
	)
}
