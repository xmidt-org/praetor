// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"github.com/hashicorp/consul/api"
	"go.uber.org/fx"
)

func newClient(cfg api.Config) (*api.Client, error) {
	return api.NewClient(&cfg)
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
// This provider expects an api.Config to be present in the application
// (NOT an *api.Config). In order to bootstrap using praetor's cofiguration,
// use ProvideConfig in addition to this function.
//
// The following components are emitted by this provider:
//
//   - *api.Client
//   - *api.Agent
//   - *api.Catalog
//   - *api.Health
func Provide() fx.Option {
	return fx.Provide(
		newClient,
		newAgent,
		newCatalog,
		newHealth,
	)
}

// ProvideConfig bootstraps an api.Config using a praetor Config.
//
// NOTE: In order to inject a custom *http.Client or *http.Transport,
// use fx.Decorate and decorate the api.Config.
func ProvideConfig() fx.Option {
	return fx.Provide(
		NewAPIConfig,
	)
}
