// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"github.com/hashicorp/consul/api"
	"go.uber.org/fx"
	"go.uber.org/multierr"
)

// Decorate is an uber/fx decorator that returns a new consul client Config
// that results from applying any number of options to an existing Config.
// If no options are supplied, this function returns a clone of the original.
func Decorate(original api.Config, opts ...Option) (cfg api.Config, err error) {
	cfg = original
	for _, o := range opts {
		err = multierr.Append(err, o(&cfg))
	}

	return
}

// New is the standard constructor for a consul client.  It allows for
// any number of options to tailor the configuration after the api.Config has
// been unmarshaled or obtained from some external source.
//
// This function may be used directly with fx.Provide as a constructor.  More
// commonly, the Provide function in this package is preferred since it allows
// simpler annotation.
func New(cfg api.Config, opts ...Option) (c *api.Client, err error) {
	cfg, err = Decorate(cfg, opts...)
	if err == nil {
		c, err = api.NewClient(&cfg)
	}

	return
}

// Provide gives a very simple, opinionated way of using New within an fx.App.
// It assumes a global, unnamed api.Config optional dependency and zero or more ClientOptions
// in a value group named 'consul.options'.
//
// Zero or more options that are external to the enclosing fx.App may be supplied to this
// provider function.  This allows the consul Client to be modified by command-line options,
// hardcoded values, etc.  Any external options supplied to this function take precedence
// over injected options.
//
// This provider emits a global, unnamed *api.Client.
func Provide(external ...Option) fx.Option {
	ctor := New
	if len(external) > 0 {
		ctor = func(cfg api.Config, injected ...Option) (*api.Client, error) {
			return New(cfg, append(injected, external...)...)
		}
	}

	return fx.Provide(
		fx.Annotate(
			ctor,
			fx.ParamTags(`optional:"true"`, `group:"consul.options"`),
		),
	)
}
