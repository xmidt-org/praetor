// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"net/http"
	"reflect"

	"github.com/hashicorp/consul/api"
)

// Option is a functional option for tailoring the consul client
// configuration prior to creating it.  Each option can modify the
// *api.Config prior to it being passed to api.NewClient.
type Option func(*api.Config) error

var (
	optionType        = reflect.TypeOf(Option(nil))
	noErrorOptionType = reflect.TypeOf((func(*api.Config))(nil))
)

// OptionFunc represents the types of functions that can be coerced into Options.
type OptionFunc interface {
	~func(*api.Config) error | ~func(*api.Config)
}

// AsOption coerces a function into an Option.
func AsOption[OF OptionFunc](of OF) Option {
	// trivial conversions
	switch oft := any(of).(type) {
	case Option:
		return oft

	case func(*api.Config):
		return func(cfg *api.Config) error {
			oft(cfg)
			return nil
		}
	}

	// now we convert to the underlying type
	ofv := reflect.ValueOf(of)
	if ofv.CanConvert(optionType) {
		return ofv.Convert(optionType).Interface().(Option)
	}

	// there are only (2) types, so the other type must be it
	f := ofv.Convert(noErrorOptionType).Interface().(func(*api.Config))
	return func(cfg *api.Config) error {
		f(cfg)
		return nil
	}
}

// WithHTTPClient configures the consul client with a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *api.Config) error {
		cfg.HttpClient = client
		return nil
	}
}
