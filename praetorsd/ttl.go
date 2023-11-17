// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"go.uber.org/multierr"
)

type ticker func(time.Duration) (<-chan time.Time, func())

func defaultTicker(d time.Duration) (<-chan time.Time, func()) {
	t := time.NewTicker(d)
	return t.C, t.Stop
}

// AgentTTLer describes the behavior of updating an Agent's TTL check.
type AgentTTLer interface {
	UpdateTTLOpts(checkID, output, status string, q *api.QueryOptions) error
}

type ttlCheck struct {
	serviceID ServiceID
	checkID   CheckID
	interval  time.Duration
	state     HealthState
	ticker    ticker
}

// TTL is a manager for updating any TTL checks in the background.
type TTL struct {
	checks []ttlCheck

	lock sync.Mutex
	ctx  context.Context
}

func NewTTL(r Registrar, h *Health, regs ServiceRegistrations) (t *TTL, err error) {
	regs.EachCheck(func(serviceID ServiceID, checkID CheckID, check api.AgentServiceCheck) {
		if len(check.TTL) == 0 {
			return
		}

		_, timeErr := time.ParseDuration(check.TTL)
		if timeErr != nil {
			err = multierr.Append(
				err,
				fmt.Errorf(
					"Invalid TTL duration for service [%s] check [%s]: %s",
					serviceID,
					checkID,
					timeErr,
				),
			)

			return
		}
	})

	return
}

func (t *TTL) updateTTLTask(ctx context.Context, checkID CheckID, ttl time.Duration, states <-chan HealthState) {
}
