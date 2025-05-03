// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"context"

	"github.com/hashicorp/consul/api"
)

// ttl represents a single TTL update task that runs on an interval.
type ttl struct {
	// updater represents the agent used to perform the update.
	updater TTLUpdater

	// def holds the defined parameters for this TTL, such as the id and interval.
	def ttlDefinition

	// newTimer is a factory for creating timers. useful to replace in unit tests.
	newTimer newTimer

	// state is the current health State in the enclosing Registrar.
	state *stateAccessor
}

// update performs an update with the check's current status.
func (t *ttl) update(qo *api.QueryOptions) error {
	s := t.state.State()
	return t.updater.UpdateTTLOpts(
		string(t.def.id),
		s.Output,
		s.Status.String(),
		qo,
	)
}

// run updates the configured check on the supplied interval.
func (t *ttl) run(ctx context.Context) {
	uo := t.def.updateOptions.WithContext(ctx)

	for {
		t.update(uo) // TODO: what to do with the error?

		// be a little more responsive:  don't bother
		// creating a timer if it's not necessary
		if ctx.Err() != nil {
			return
		}

		ch, stop := t.newTimer(t.def.interval)
		select {
		case <-ctx.Done():
			stop()
			return

		case <-ch:
			// continue
		}
	}
}
