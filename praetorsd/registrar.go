// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import "github.com/hashicorp/consul/api"

// Registrar represents a service which can register and deregister
// services with consul.
type Registrar interface {
	Register(*api.AgentServiceRegistration, api.ServiceRegisterOpts) error
	Deregister(serviceID string, opts *api.QueryOptions) error
}

type nopRegistrar struct{}

func (nr nopRegistrar) Register(*api.AgentServiceRegistration, api.ServiceRegisterOpts) error {
	return nil
}

func (nr nopRegistrar) Deregister(string, *api.QueryOptions) error {
	return nil
}

type agentRegistrar struct {
	agent *api.Agent
}

func (ar agentRegistrar) Register(reg *api.AgentServiceRegistration, opts api.ServiceRegisterOpts) error {
	return ar.agent.ServiceRegisterOpts(reg, opts)
}

func (ar agentRegistrar) Deregister(serviceID string, opts *api.QueryOptions) error {
	return ar.agent.ServiceDeregisterOpts(serviceID, opts)
}

// NewRegistrar produces a Registrar from a consul client.  If the client is nil,
// a nop Registrar is returned that does nothing.  Otherwise, a Registrar is
// created from the client's Agent instance.
func NewRegistrar(client *api.Client) Registrar {
	if client == nil {
		// allow registration to be disabled via a nil client
		return nopRegistrar{}
	}

	return agentRegistrar{
		agent: client.Agent(),
	}
}
