// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import (
	"iter"

	"github.com/hashicorp/consul/api"
)

// ChecksLen returns the count of registered checks associated with the service.
func ChecksLen(service api.AgentServiceRegistration) (n int) {
	n = len(service.Checks)
	if service.Check != nil {
		n++
	}

	return
}

// Checks allows easy iteration over a service's health check definitions.
func Checks(service api.AgentServiceRegistration) iter.Seq2[int, api.AgentServiceCheck] {
	return func(f func(int, api.AgentServiceCheck) bool) {
		base := 0
		if service.Check != nil {
			if !f(base, *service.Check) {
				return
			}

			base++
		}

		for i, check := range service.Checks {
			if !f(base+i, *check) {
				return
			}
		}
	}
}
