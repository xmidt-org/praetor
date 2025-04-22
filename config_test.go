// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
}

// newSimpleConfig creates a praetor Config with the simple fields set.
func (suite *ConfigTestSuite) newSimpleConfig() Config {
	return Config{
		Scheme:     "ftp",
		Address:    "foobar:8080",
		PathPrefix: "/prefix",
		Datacenter: "abc",
		WaitTime:   5 * time.Minute,
		Token:      "xyz",
		TokenFile:  "/etc/app/token",
		Namespace:  "namespace",
		Partition:  "partition",
	}
}

// assertSimpleFields asserts that the given consul api.Config's simple fields
// matches what is set by newSimpleConfig.
func (suite *ConfigTestSuite) assertSimpleFields(cfg api.Config) {
	suite.Equal("ftp", cfg.Scheme)
	suite.Equal("foobar:8080", cfg.Address)
	suite.Equal("/prefix", cfg.PathPrefix)
	suite.Equal("abc", cfg.Datacenter)
	suite.Equal(5*time.Minute, cfg.WaitTime)
	suite.Equal("xyz", cfg.Token)
	suite.Equal("/etc/app/token", cfg.TokenFile)
	suite.Equal("namespace", cfg.Namespace)
	suite.Equal("partition", cfg.Partition)
	suite.Nil(cfg.HttpClient)
	suite.Nil(cfg.Transport)
}

func (suite *ConfigTestSuite) testNewAPIConfigSimple() {
	cfg := newAPIConfig(
		suite.newSimpleConfig(),
	)

	suite.assertSimpleFields(cfg)
	suite.Nil(cfg.HttpAuth)
	suite.Equal(api.TLSConfig{}, cfg.TLSConfig)
}

func (suite *ConfigTestSuite) testNewAPIConfigHttpAuth() {
	src := suite.newSimpleConfig()
	src.BasicAuth.UserName = "user"
	src.BasicAuth.Password = "password"

	cfg := newAPIConfig(src)

	suite.assertSimpleFields(cfg)
	suite.Equal(api.TLSConfig{}, cfg.TLSConfig)
	suite.Require().NotNil(cfg.HttpAuth)
	suite.Equal(
		api.HttpBasicAuth{
			Username: "user",
			Password: "password",
		},
		*cfg.HttpAuth,
	)
}

func (suite *ConfigTestSuite) testNewAPIConfigTLS() {
	src := suite.newSimpleConfig()
	src.TLS.Address = "foobar:9090"
	src.TLS.CAFile = "/etc/app/cafile"
	src.TLS.CAPath = "/etc/app/capath"
	src.TLS.CertificateFile = "/etc/app/certificateFile"
	src.TLS.KeyFile = "/etc/app/keyFile"
	src.TLS.InsecureSkipVerify = true

	cfg := newAPIConfig(src)

	suite.assertSimpleFields(cfg)
	suite.Nil(cfg.HttpAuth)
	suite.Equal(
		api.TLSConfig{
			Address:            "foobar:9090",
			CAFile:             "/etc/app/cafile",
			CAPath:             "/etc/app/capath",
			CertFile:           "/etc/app/certificateFile",
			KeyFile:            "/etc/app/keyFile",
			InsecureSkipVerify: true,
		},
		cfg.TLSConfig,
	)
}

func (suite *ConfigTestSuite) TestNewAPIConfig() {
	suite.Run("Simple", suite.testNewAPIConfigSimple)
	suite.Run("HttpAuth", suite.testNewAPIConfigHttpAuth)
	suite.Run("TLS", suite.testNewAPIConfigTLS)
}

func TestConfig(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
