// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import "time"

// newTimer is a factory for timers. this level of indirection allows
// unit tests to inject timers under test control.
type newTimer func(time.Duration) (<-chan time.Time, func() bool)

// defaultNewTimer delegates to time.NewTimer.
func defaultNewTimer(d time.Duration) (<-chan time.Time, func() bool) {
	t := time.NewTimer(d)
	return t.C, t.Stop
}
