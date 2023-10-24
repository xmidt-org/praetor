// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"net/http"

	"github.com/hashicorp/consul/api"
	"go.uber.org/multierr"
)

// Option is a functional option for tailoring the consul client
// configuration prior to creating it.  Each option can modify the
// *api.Config prior to it being passed to api.NewClient.
type Option func(*api.Config) error

// AsOption bundles one or more functions into a single Option.
func AsOption[O ~func(*api.Config) error](opts ...O) Option {
	return func(cfg *api.Config) (err error) {
		for _, o := range opts {
			err = multierr.Append(err, o(cfg))
		}

		return
	}
}

// WithHTTPClient configures the consul client with a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *api.Config) error {
		cfg.HttpClient = client
		return nil
	}
}
