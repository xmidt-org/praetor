package praetor

import (
	"errors"
	"net/http"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/suite"
)

const testAddress = "localhost:1234"

type OptionSuite struct {
	suite.Suite
}

func (suite *OptionSuite) testAsOptionWithOption() {
	suite.Run("Success", func() {
		opt := Option(func(cfg *api.Config) error {
			cfg.Address = testAddress
			return nil
		})

		var cfg api.Config
		err := AsOption(opt)(&cfg)
		suite.NoError(err)
		suite.Equal(testAddress, cfg.Address)
	})

	suite.Run("Fail", func() {
		expectedErr := errors.New("expected")
		opt := Option(func(cfg *api.Config) error {
			return expectedErr
		})

		var cfg api.Config
		err := AsOption(opt)(&cfg)
		suite.ErrorIs(err, expectedErr)
	})
}

func (suite *OptionSuite) testAsOptionWithClosure() {
	suite.Run("Success", func() {
		opt := func(cfg *api.Config) error {
			cfg.Address = testAddress
			return nil
		}

		var cfg api.Config
		err := AsOption(opt)(&cfg)
		suite.NoError(err)
		suite.Equal(testAddress, cfg.Address)
	})

	suite.Run("Fail", func() {
		expectedErr := errors.New("expected")
		opt := func(cfg *api.Config) error {
			return expectedErr
		}

		var cfg api.Config
		err := AsOption(opt)(&cfg)
		suite.ErrorIs(err, expectedErr)
	})
}

func (suite *OptionSuite) testAsOptionNoError() {
	opt := func(cfg *api.Config) {
		cfg.Address = testAddress
	}

	var cfg api.Config
	err := AsOption(opt)(&cfg)
	suite.NoError(err)
	suite.Equal(testAddress, cfg.Address)
}

func (suite *OptionSuite) testAsOptionCustomType() {
	type TestFunc func(*api.Config)
	var opt TestFunc = func(cfg *api.Config) {
		cfg.Address = testAddress
	}

	var cfg api.Config
	err := AsOption(opt)(&cfg)
	suite.NoError(err)
	suite.Equal(testAddress, cfg.Address)
}

func (suite *OptionSuite) TestAsOption() {
	suite.Run("WithOption", suite.testAsOptionWithOption)
	suite.Run("WithClosure", suite.testAsOptionWithClosure)
	suite.Run("NoError", suite.testAsOptionNoError)
	suite.Run("CustomType", suite.testAsOptionCustomType)
}

func (suite *OptionSuite) TestWithHTTPClient() {
	c := new(http.Client)
	var cfg api.Config
	suite.NoError(
		WithHTTPClient(c)(&cfg),
	)

	suite.Same(c, cfg.HttpClient)
}

func TestOption(t *testing.T) {
	suite.Run(t, new(OptionSuite))
}
