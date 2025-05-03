// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ServiceRegistrationChecksTestSuite struct {
	suite.Suite
}

func TestServiceRegistrationChecks(t *testing.T) {
	suite.Run(t, new(ServiceRegistrationChecksTestSuite))
}

type RegistrationsBuilderTestSuite struct {
	suite.Suite
}

func TestRegistrationsBuilder(t *testing.T) {
	suite.Run(t, new(RegistrationsBuilderTestSuite))
}
