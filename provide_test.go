// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
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

// provideSuccessApp creates an *fxtest.App using Provide() and asserts
// that everything went correctly.
func (suite *ProvideSuite) provideSuccessApp(extra ...fx.Option) {
	var (
		client *api.Client

		agent   *api.Agent
		catalog *api.Catalog
		health  *api.Health
		kv      *api.KV
	)

	app := fxtest.New(
		suite.T(),
		append(
			extra,
			Provide(),
			fx.Populate(
				&client,
				&agent,
				&catalog,
				&health,
				&kv,
			),
		)...,
	)

	suite.Require().NotNil(app)
	suite.Require().NoError(app.Err())

	suite.NotNil(client)
	suite.NotNil(agent)
	suite.NotNil(catalog)
	suite.NotNil(health)
	suite.NotNil(kv)
}

func (suite *ProvideSuite) testProvideNoAPIConfig() {
	suite.provideSuccessApp()
}

func (suite *ProvideSuite) testProvideWithAPIConfig() {
	suite.provideSuccessApp(
		fx.Supply(
			api.Config{
				Scheme: "https",
			},
		),
	)
}

func (suite *ProvideSuite) TestProvide() {
	suite.Run("NoAPIConfig", suite.testProvideNoAPIConfig)
	suite.Run("WithAPIConfig", suite.testProvideWithAPIConfig)
}

func TestProvide(t *testing.T) {
	suite.Run(t, new(ProvideSuite))
}
