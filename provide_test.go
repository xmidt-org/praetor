// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"errors"
	"net/http"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type ProvideSuite struct {
	suite.Suite
}

func (suite *ProvideSuite) testDecorateSuccess() {
	original := api.Config{
		Address: testAddress,
	}

	decorated, err := Decorate(
		original,
		func(cfg *api.Config) error {
			cfg.Scheme = testScheme
			return nil
		},
	)

	suite.NoError(err)
	suite.Equal(api.Config{Address: testAddress, Scheme: testScheme}, decorated)
}

func (suite *ProvideSuite) testDecorateOptionError() {
	original := api.Config{
		Address: testAddress,
	}

	expectedErr := errors.New("expected")

	_, actualErr := Decorate(
		original,
		func(cfg *api.Config) error {
			return expectedErr
		},
	)

	suite.ErrorIs(actualErr, expectedErr)
}

func (suite *ProvideSuite) TestDecorate() {
	suite.Run("Success", suite.testDecorateSuccess)
	suite.Run("OptionError", suite.testDecorateOptionError)
}

func (suite *ProvideSuite) testNewSuccess() {
	c, err := New(
		api.Config{
			Address: testAddress,
		},
		func(cfg *api.Config) error {
			cfg.Scheme = testScheme
			return nil
		},
		func(cfg *api.Config) error {
			cfg.HttpClient = new(http.Client)
			return nil
		},
	)

	suite.NoError(err)
	suite.NotNil(c)
}

func (suite *ProvideSuite) testNewNoOptions() {
	c, err := New(
		api.Config{
			Address: testAddress,
		},
	)

	suite.NoError(err)
	suite.NotNil(c)
}

func (suite *ProvideSuite) testNewOptionError() {
	expectedErr := errors.New("expected")

	c, actualErr := New(
		api.Config{
			Address: testAddress,
		},
		func(*api.Config) error {
			return expectedErr
		},
	)

	suite.ErrorIs(actualErr, expectedErr)
	suite.Nil(c)
}

func (suite *ProvideSuite) TestNew() {
	suite.Run("Success", suite.testNewSuccess)
	suite.Run("NoOptions", suite.testNewNoOptions)
	suite.Run("OptionError", suite.testNewOptionError)
}

func (suite *ProvideSuite) testProvideDefault() {
	var c *api.Client
	app := fxtest.New(
		suite.T(),
		Provide(),
		fx.Populate(&c),
	)

	suite.NoError(app.Err())
	suite.NotNil(c)
}

func (suite *ProvideSuite) testProvideWithOptions() {
	var (
		c  *api.Client
		hc = new(http.Client)

		external1 = func(cfg *api.Config) {
			// injected options should execute first
			suite.Equal("different:9999", cfg.Address)
			suite.Equal(testScheme, cfg.Scheme)
			cfg.Address = testAddress
		}

		external2 = WithHTTPClient(hc)
	)

	app := fxtest.New(
		suite.T(),
		fx.Supply(
			fx.Annotate(
				Option(func(cfg *api.Config) error {
					cfg.Address = "different:9999"
					return nil
				}),
				fx.ResultTags(`group:"consul.options"`),
			),
			fx.Annotate(
				Option(func(cfg *api.Config) error {
					cfg.Scheme = testScheme
					return nil
				}),
				fx.ResultTags(`group:"consul.options"`),
			),
		),
		Provide(AsOption(external1), external2),
		fx.Populate(&c),
	)

	suite.NoError(app.Err())
	suite.NotNil(c)
}

func (suite *ProvideSuite) testProvideWithConfig() {
	var c *api.Client
	app := fxtest.New(
		suite.T(),
		fx.Supply(
			api.Config{
				Address: "original:8888",
			},
			fx.Annotate(
				Option(func(cfg *api.Config) error {
					// the original configuration should be visible here
					suite.Equal("original:8888", cfg.Address)
					cfg.Address = "different:9999"
					return nil
				}),
				fx.ResultTags(`group:"consul.options"`),
			),
		),
		Provide(),
		fx.Populate(&c),
	)

	suite.NoError(app.Err())
	suite.NotNil(c)
}

func (suite *ProvideSuite) TestProvide() {
	suite.Run("Default", suite.testProvideDefault)
	suite.Run("WithOptions", suite.testProvideWithOptions)
	suite.Run("WithConfig", suite.testProvideWithConfig)
}

func TestProvide(t *testing.T) {
	suite.Run(t, new(ProvideSuite))
}
