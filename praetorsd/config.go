// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"github.com/hashicorp/consul/api"
	"github.com/xmidt-org/retry"
)

type Query struct {
	Service     string
	Tags        []string
	PassingOnly bool
	Options     *api.QueryOptions
}

// RegistrationConfig is the service registration portion of praetor's configuration.
// This will typically be obtained externally via the Config.
type RegistrationConfig struct {
	// Retry is the backoff configuration for retrying service registrations.  If not
	// supplied, no retries are performed.
	//
	// Service deregistrations are never retried.
	Retry retry.Config `json:"retry" yaml:"retry"`

	// Services holds the set of consul service descriptions that this application should
	// register.  These will be this application's identity within consul.
	Services []ServiceRegistration `json:"services" yaml:"services"`
}
