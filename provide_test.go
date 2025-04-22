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

	client  *api.Client
	agent   *api.Agent
	catalog *api.Catalog
	health  *api.Health
	kv      *api.KV
}

func (suite *ProvideSuite) SetupTest() {
	suite.client = nil
	suite.agent = nil
	suite.catalog = nil
	suite.health = nil
	suite.kv = nil
}

func (suite *ProvideSuite) SetupSubTest() {
	suite.SetupTest()
}

// populate returns an option that populates the various services
// that Provide emits.
func (suite *ProvideSuite) populate() fx.Option {
	return fx.Populate(
		&suite.client,
		&suite.agent,
		&suite.catalog,
		&suite.health,
		&suite.kv,
	)
}

// assertServices verifies that all the services that Provide
// emits were set.
func (suite *ProvideSuite) assertServices() {
	suite.NotNil(suite.client)
	suite.NotNil(suite.agent)
	suite.NotNil(suite.catalog)
	suite.NotNil(suite.health)
	suite.NotNil(suite.kv)
}

func (suite *ProvideSuite) TestProvide() {
	suite.Run("WithAPIConfig", func() {
		fxtest.New(
			suite.T(),
			fx.Supply(api.Config{}),
			fx.NopLogger,
			Provide(),
			suite.populate(),
		)

		suite.assertServices()
	})

	suite.Run("NoAPIConfig", func() {
		fxtest.New(
			suite.T(),
			fx.NopLogger,
			Provide(),
			suite.populate(),
		)

		suite.assertServices()
	})
}

func (suite *ProvideSuite) TestProvideConfig() {
	suite.Run("WithPraetorConfig", func() {
		var acfg api.Config
		fxtest.New(
			suite.T(),
			fx.NopLogger,
			fx.Supply(
				Config{
					Scheme:  "http",
					Address: "foobar:8080",
				},
			),
			ProvideConfig(),
			Provide(),
			suite.populate(),
			fx.Populate(&acfg),
		)

		suite.assertServices()
		suite.Equal("http", acfg.Scheme)
		suite.Equal("foobar:8080", acfg.Address)
	})

	suite.Run("NoPraetorConfig", func() {
		var acfg api.Config
		fxtest.New(
			suite.T(),
			fx.NopLogger,
			ProvideConfig(),
			Provide(),
			suite.populate(),
			fx.Populate(&acfg), // just verify that an api.Config was created
		)

		suite.assertServices()
	})
}

func TestProvide(t *testing.T) {
	suite.Run(t, new(ProvideSuite))
}
