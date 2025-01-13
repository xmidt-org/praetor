// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type ProvideSuite struct {
	suite.Suite
}

func (suite *ProvideSuite) TestProvide() {
	var (
		client  *api.Client
		agent   *api.Agent
		catalog *api.Catalog
		health  *api.Health

		app = fxtest.New(
			suite.T(),
			fx.Supply(api.Config{}),
			Provide(),
			fx.Populate(
				&client,
				&agent,
				&catalog,
				&health,
			),
		)
	)

	suite.NoError(app.Err())
	suite.NotNil(client)
	suite.NotNil(agent)
	suite.NotNil(catalog)
	suite.NotNil(health)
}

func (suite *ProvideSuite) TestProvideConfig() {
	var (
		config  api.Config
		client  *api.Client
		agent   *api.Agent
		catalog *api.Catalog
		health  *api.Health

		app = fxtest.New(
			suite.T(),
			fx.Supply(
				Config{
					Scheme:  "http",
					Address: "foobar:8080",
				},
			),
			Provide(),
			ProvideConfig(),
			fx.Populate(
				&config,
				&client,
				&agent,
				&catalog,
				&health,
			),
		)
	)

	suite.NoError(app.Err())
	suite.Equal("http", config.Scheme)
	suite.Equal("foobar:8080", config.Address)
	suite.NotNil(client)
	suite.NotNil(agent)
	suite.NotNil(catalog)
	suite.NotNil(health)
}

func TestProvide(t *testing.T) {
	suite.Run(t, new(ProvideSuite))
}
