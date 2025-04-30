// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"github.com/hashicorp/consul/api"
)

// AgentRegisterer is the low-level behavior of anything that can actually
// perform a service registration.
type AgentRegisterer interface {
	ServiceRegisterOpts(*api.AgentServiceRegistration, api.ServiceRegisterOpts) error
}

// AgentDeregisterer is the low-level behavior of anything that can actually
// perform a service deregistration.
type AgentDeregisterer interface {
	ServiceDeregisterOpts(serviceID string, opts *api.QueryOptions) error
}

// TTLUpdater is the low-level behavior of anything that can actually
// update the status of a TTL check.
type TTLUpdater interface {
	UpdateTTLOpts(checkID, output, status string, opts *api.QueryOptions) error
}
