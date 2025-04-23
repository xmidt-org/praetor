// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/suite"
)

type StatusTestSuite struct {
	suite.Suite
}

// TestString verifies the Consul values for each Status.
func (suite *StatusTestSuite) TestString() {
	suite.Equal(api.HealthAny, Any.String())
	suite.Equal(api.HealthPassing, Passing.String())
	suite.Equal(api.HealthWarning, Warning.String())
	suite.Equal(api.HealthCritical, Critical.String())
	suite.Equal(api.HealthMaint, Maintenance.String())
}

func TestStatus(t *testing.T) {
	suite.Run(t, new(StatusTestSuite))
}
