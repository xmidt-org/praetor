// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import "github.com/hashicorp/consul/api"

// Datacenters is the strategy used to obtain a list of datacenters.
type Datacenters interface {
	Get() ([]string, error)
}

type catalogDatacenters struct {
	catalog *api.Catalog
}

func (cd catalogDatacenters) Get() ([]string, error) {
	return cd.catalog.Datacenters()
}

// NewDatacenters returns a Datacenters strategy backed by the
// client's Catalog.
func NewDatacenters(client *api.Client) Datacenters {
	return catalogDatacenters{
		catalog: client.Catalog(),
	}
}
