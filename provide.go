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
func Provide() fx.Option {
	return fx.Provide(
		fx.Annotate(
			newClient,
			fx.ParamTags(`optional:"true"`),
		),
		newAgent,
		newCatalog,
		newHealth,
	)
}

// ProvideConfig uses the Config object in this package to bootstrap an api.Config.
// This function uses ProvidCustomConfig to build the returned option.
func ProvideConfig() fx.Option {
	return ProvideCustomConfig[Config](newAPIConfig)
}

// ProvideCustomConfig allows a custom configuration object, possibly unmarshaled,
// to be used to bootstrap a consul api.Config.
//
// The options returned by this function take an optional configuration object of
// type C. The given closure, which cannot be nil, will be passed the injected
// value of C and the returned api.Config will then be used by Provide.
//
// Note that C is an optional dependency, to allow flexibility when boostrapping
// an application. The closure must handle default values of C gracefully.
func ProvideCustomConfig[C any, F APIConfigurer[C]](cnv F) fx.Option {
	return fx.Provide(
		fx.Annotate(
			asAPIConfigurer[C](cnv),
			fx.ParamTags(`optional:"true"`),
		),
	)
}
