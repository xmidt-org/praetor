// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"github.com/hashicorp/consul/api"
	"go.uber.org/fx"
)

func newAgentRegisterer(a *api.Agent) AgentRegisterer     { return a }
func newAgentDeregisterer(a *api.Agent) AgentDeregisterer { return a }
func newTTLUpdater(a *api.Agent) TTLUpdater               { return a }

// provideAgent requires a consul *api.Agent and produces each of the agent
// interfaces defined in this package. A client can further decorate each
// of these interfaces via fx.Decorate.
func provideAgent() fx.Option {
	return fx.Provide(
		newAgentRegisterer,
		newAgentDeregisterer,
		newTTLUpdater,
	)
}

type registrarsIn struct {
	fx.In

	Definitions       *Definitions `optional:"true"`
	AgentRegisterer   AgentRegisterer
	AgentDeregisterer AgentDeregisterer
	TTLUpdater        TTLUpdater

	Lifecycle fx.Lifecycle
}

// newRegistrars is the internal constructor for a Registrars component
// based on fx.App dependencies.
func newRegistrars(in registrarsIn) (rs Registrars, err error) {
	rs, err = NewRegistrars(
		in.Definitions,
		WithAgentRegisterer(in.AgentRegisterer),
		WithAgentDeregisterer(in.AgentDeregisterer),
	)

	if err == nil {
		for _, r := range rs.Registrars() {
			in.Lifecycle.Append(
				fx.StartStopHook(
					r.Register,
					r.Deregister,
				),
			)
		}
	}

	return
}

// Provide creates the service discovery components required to manage an applications
// registered consul service endpoints.
//
// A Definitions bundle, of type *Definitions, can be present in the enclosing application.
// This bundle can be built from application configuration or in any other desired way.
// The Registrars component that this function creates will use this Definitions bundle.
// If no Definitions bundle is present, the created Registrars will be empty and no
// services will be managed or registered with consul.
//
// A consul *api.Agent must be present in the application. This can be built with
// praetor.Provide or by other means.
//
// One component per agent interface in this package is also created. Client code can
// use fx.Decorate to decorate any of these components:
//
//   - AgentRegisterer
//   - AgentDeregisterer
//   - TTLUpdater
func Provide() fx.Option {
	return fx.Options(
		provideAgent(),
		fx.Provide(
			newRegistrars,
		),
		fx.Invoke(
			func(Registrars) {},
		),
	)
}
