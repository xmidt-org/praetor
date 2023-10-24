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
)

type ProvideSuite struct {
	suite.Suite
}

func (suite *ProvideSuite) testDecorateSuccess() {
	original := api.Config{
		Address: "localhost:1234",
	}

	decorated, err := Decorate(
		original,
		func(cfg *api.Config) error {
			cfg.Scheme = "https"
			return nil
		},
	)

	suite.NoError(err)
	suite.Equal(api.Config{Address: "localhost:1234", Scheme: "https"}, decorated)
}

func (suite *ProvideSuite) testDecorateOptionError() {
	original := api.Config{
		Address: "localhost:1234",
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
			Address: "localhost:1234",
		},
		func(cfg *api.Config) error {
			cfg.Scheme = "https"
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
			Address: "localhost:1234",
		},
	)

	suite.NoError(err)
	suite.NotNil(c)
}

func (suite *ProvideSuite) testNewOptionError() {
	expectedErr := errors.New("expected")

	c, actualErr := New(
		api.Config{
			Address: "localhost:1234",
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

func (suite *ProvideSuite) TestExperiment() {
	var c *api.Client
	app := fx.New(
		Provide(),
		fx.Provide(
			fx.Annotate(
				func() Option {
					return func(cfg *api.Config) error {
						suite.T().Log("option 1")
						return nil
					}
				},
				fx.ResultTags(`group:"consul.options"`),
			),
			fx.Annotate(
				func() []Option {
					return []Option{
						func(cfg *api.Config) error {
							suite.T().Log("option 2")
							return nil
						},
					}
				},
				fx.ResultTags(`group:"consul.options"`),
			),
		),
		fx.Populate(&c),
	)

	suite.NoError(app.Err())
	suite.NotNil(c)
}

func TestProvide(t *testing.T) {
	suite.Run(t, new(ProvideSuite))
}
