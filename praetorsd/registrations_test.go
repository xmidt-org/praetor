// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"testing"

	"maps"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/suite"
)

type ServiceRegistrationChecksTestSuite struct {
	suite.Suite
}

func (suite *ServiceRegistrationChecksTestSuite) TestIteration() {
	testCases := []struct {
		name          string
		registrations []api.AgentServiceRegistration
		expected      map[CheckKey]api.AgentServiceCheck
	}{
		{
			name:     "Empty",
			expected: map[CheckKey]api.AgentServiceCheck{},
		},
		{
			name: "One",
			registrations: []api.AgentServiceRegistration{
				{
					ID: "service0",
					Check: &api.AgentServiceCheck{
						CheckID: "first",
					},
					Checks: api.AgentServiceChecks{
						&api.AgentServiceCheck{
							Name: "second",
						},
					},
				},
			},
			expected: map[CheckKey]api.AgentServiceCheck{
				{ServiceID: "service0", CheckID: "first"}: {
					CheckID: "first",
				},
				{ServiceID: "service0", CheckID: "second"}: {
					Name: "second",
				},
			},
		},
		{
			name: "Three",
			registrations: []api.AgentServiceRegistration{
				{
					ID: "service0",
					Checks: api.AgentServiceChecks{
						&api.AgentServiceCheck{
							Name: "first",
						},
					},
				},
				{
					Name: "service1",
					Check: &api.AgentServiceCheck{
						Name: "second",
					},
				},
				{
					ID:   "service2",
					Name: "shouldnotbeused",
					Check: &api.AgentServiceCheck{
						CheckID: "third",
					},
					Checks: api.AgentServiceChecks{
						&api.AgentServiceCheck{
							Name: "fourth",
						},
						&api.AgentServiceCheck{
							CheckID: "fifth",
							Name:    "shouldnotbeused", // CheckID takes precedence
						},
					},
				},
			},
			expected: map[CheckKey]api.AgentServiceCheck{
				{ServiceID: "service0", CheckID: "first"}: {
					Name: "first",
				},
				{ServiceID: "service1", CheckID: "second"}: {
					Name: "second",
				},
				{ServiceID: "service2", CheckID: "third"}: {
					CheckID: "third",
				},
				{ServiceID: "service2", CheckID: "fourth"}: {
					Name: "fourth",
				},
				{ServiceID: "service2", CheckID: "fifth"}: {
					CheckID: "fifth",
					Name:    "shouldnotbeused",
				},
			},
		},
	}

	for _, testCase := range testCases {
		suite.Run(testCase.name, func() {
			suite.Equal(
				len(testCase.expected),
				ServiceRegistrationChecksLen(testCase.registrations...),
			)
		})

		actual := maps.Collect(ServiceRegistrationChecks(testCase.registrations...))
		suite.Equal(testCase.expected, actual)
	}
}

func (suite *ServiceRegistrationChecksTestSuite) TestEarlyReturn() {
	registration := api.AgentServiceRegistration{
		ID: "service",
		Check: &api.AgentServiceCheck{
			Name: "first",
		},
		Checks: api.AgentServiceChecks{
			{
				CheckID: "second",
			},
			{
				Name: "third",
			},
		},
	}

	suite.Run("First", func() {
		for key := range ServiceRegistrationChecks(registration) {
			suite.Equal(CheckID("first"), key.CheckID)
			suite.Equal(ServiceID("service"), key.ServiceID)
			break // should prevent any further calls
		}
	})

	suite.Run("Second", func() {
		i := 0
		for key := range ServiceRegistrationChecks(registration) {
			switch i {
			case 0:
				suite.Equal(CheckID("first"), key.CheckID)
				suite.Equal(ServiceID("service"), key.ServiceID)
			case 1:
				suite.Equal(CheckID("second"), key.CheckID)
				suite.Equal(ServiceID("service"), key.ServiceID)
			default:
				suite.Fail("early return should have prevented subsequent calls")
			}

			if i == 1 {
				break
			}

			i++
		}
	})
}

func TestServiceRegistrationChecks(t *testing.T) {
	suite.Run(t, new(ServiceRegistrationChecksTestSuite))
}

type RegistrationsBuilderTestSuite struct {
	suite.Suite
}

func (suite *RegistrationsBuilderTestSuite) testBuildSuccess() {
	testCases := []struct {
		name          string
		registrations []api.AgentServiceRegistration
		expected      map[ServiceID]api.AgentServiceRegistration
	}{
		{
			name:     "Empty",
			expected: map[ServiceID]api.AgentServiceRegistration{},
		},
		{
			name: "One",
			registrations: []api.AgentServiceRegistration{
				{
					Name: "service",
				},
			},
			expected: map[ServiceID]api.AgentServiceRegistration{
				"service": {Name: "service"},
			},
		},
		{
			name: "Three",
			registrations: []api.AgentServiceRegistration{
				{
					Name: "service0",
				},
				{
					ID: "service1",
					Check: &api.AgentServiceCheck{
						CheckID: "first",
					},
				},
				{
					ID:   "service2",
					Name: "shouldnotbeused",
					Checks: api.AgentServiceChecks{
						{
							Name: "second",
						},
						{
							CheckID: "third",
							Name:    "shouldnotbeused",
						},
					},
				},
			},
			expected: map[ServiceID]api.AgentServiceRegistration{
				"service0": {Name: "service0"},
				"service1": {
					ID: "service1",
					Check: &api.AgentServiceCheck{
						CheckID: "first",
					},
				},
				"service2": {
					ID:   "service2",
					Name: "shouldnotbeused",
					Checks: api.AgentServiceChecks{
						{
							Name: "second",
						},
						{
							CheckID: "third",
							Name:    "shouldnotbeused",
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		suite.Run(testCase.name, func() {
			var rb RegistrationsBuilder
			suite.NoError(rb.Err())

			suite.Same(&rb, rb.AddServiceRegistrations(testCase.registrations...))
			suite.NoError(rb.Err())

			r, err := rb.Build()
			suite.Require().NoError(err)
			suite.Require().NotNil(r)
			suite.Equal(len(testCase.expected), r.ServiceRegistrationsLen())

			actual := maps.Collect(r.ServiceRegistrations())
			suite.Equal(testCase.expected, actual)

			// early return
			i := 0
			for range r.ServiceRegistrations() {
				if i > 0 {
					suite.Fail("early return should have prevented further invocations")
				}

				i++
				break
			}
		})
	}
}

func (suite *RegistrationsBuilderTestSuite) testBuildError() {
	testCases := []struct {
		name          string
		registrations []api.AgentServiceRegistration
	}{
		{
			name: "OneServiceDuplicateCheck",
			registrations: []api.AgentServiceRegistration{
				{
					ID: "service",
					Checks: api.AgentServiceChecks{
						&api.AgentServiceCheck{
							CheckID: "duplicate",
						},
						&api.AgentServiceCheck{
							Name: "duplicate",
						},
					},
				},
			},
		},
		{
			name: "TwoServicesDuplicateCheck",
			registrations: []api.AgentServiceRegistration{
				{
					ID: "service0",
					Checks: api.AgentServiceChecks{
						&api.AgentServiceCheck{
							CheckID: "duplicate",
						},
					},
				},
				{
					ID: "service1",
					Checks: api.AgentServiceChecks{
						&api.AgentServiceCheck{
							Name: "duplicate",
						},
					},
				},
			},
		},
		{
			name: "DuplicateService",
			registrations: []api.AgentServiceRegistration{
				{
					ID: "duplicate",
				},
				{
					Name: "duplicate",
				},
			},
		},
		{
			name: "NoServiceID",
			registrations: []api.AgentServiceRegistration{
				{}, // no identifier
			},
		},
	}

	for _, testCase := range testCases {
		suite.Run(testCase.name, func() {
			var rb RegistrationsBuilder
			rb.AddServiceRegistrations(testCase.registrations...)
			suite.Error(rb.Err())

			r, err := rb.Build()
			suite.Error(err)
			suite.Nil(r)

			suite.NoError(rb.Err()) // should have been reset
		})
	}
}

func (suite *RegistrationsBuilderTestSuite) TestBuild() {
	suite.Run("Success", suite.testBuildSuccess)
	suite.Run("Error", suite.testBuildError)
}

func TestRegistrationsBuilder(t *testing.T) {
	suite.Run(t, new(RegistrationsBuilderTestSuite))
}
