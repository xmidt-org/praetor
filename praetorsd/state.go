// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

//go:generate stringer -type=Status -linecomment

// Status represents the Consul service status. The String() value
// for this type is the correct value to use with Consul's service check API.
type Status int

const (
	// Any is a wildcard status. It's not intended to be used as an actual service status.
	Any Status = iota // any

	// Passing indicates that a service is fully healthy.
	Passing // passing

	// Warning indicates that a service can still take some traffic, but
	// that something is wrong.
	Warning // warning

	// Critical means that a service cannot take traffic and is down.
	Critical // critical

	// Maintenance indicates a service that is temporarily unavailable,
	// most often due to server maintenance.
	Maintenance // maintenance
)

// State is a service's overall state. The zero value of this type represents
// a healthy service with no output.
type State struct {
	// Output is the additional detail text associated with this state. The content
	// of this type is not used by Consul, and it can be anything. Most often, it is
	// either (1) simple, human readable text, or (2) a JSON object.
	Output string

	// Status is the Consul service status.
	Status Status
}
