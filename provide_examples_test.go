// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/consul/api"
	"go.uber.org/fx"
)

func ExampleProvide_simple() {
	fx.New(
		fx.NopLogger,
		fx.Supply(api.Config{}), // this consul client config can be obtained however desired
		Provide(),
		fx.Invoke(
			// code and have any of these types as dependencies:

			func(client *api.Client) {
				fmt.Println("client")
			},
			func(agent *api.Agent) {
				fmt.Println("agent")
			},
			func(agent *api.Catalog) {
				fmt.Println("catalog")
			},
			func(agent *api.Health) {
				fmt.Println("health")
			},
		),
	)

	// Output:
	// client
	// agent
	// catalog
	// health
}

func ExampleProvide_useconfig() {
	fx.New(
		fx.NopLogger,
		// this praetor Config can be obtained externally, e.g. unmarshaled
		fx.Supply(Config{
			Scheme:  "https",
			Address: "foobar:8080",
		}),
		ProvideConfig(),
		Provide(),
		fx.Invoke(
			func(client *api.Client) {
				fmt.Println("client")
			},
			func(agent *api.Agent) {
				fmt.Println("agent")
			},
			func(agent *api.Catalog) {
				fmt.Println("catalog")
			},
			func(agent *api.Health) {
				fmt.Println("health")
			},
		),
	)

	// Output:
	// client
	// agent
	// catalog
	// health
}

func ExampleProvide_injectcustomclient() {
	fx.New(
		fx.NopLogger,
		fx.Supply(Config{
			Scheme:  "https",
			Address: "foobar:8080",
		}),
		fx.Supply(
			// we want to use this HTTP client for consul
			&http.Client{
				Timeout: 5 * time.Minute,
			},
		),
		// use standard fx decoration to add a custom HTTP client
		fx.Decorate(
			func(original api.Config, customClient *http.Client) api.Config {
				original.HttpClient = customClient
				return original
			},
		),
		ProvideConfig(),
		Provide(),
		fx.Invoke(
			func(client *api.Client) {
				fmt.Println("client")
			},
			func(agent *api.Agent) {
				fmt.Println("agent")
			},
			func(agent *api.Catalog) {
				fmt.Println("catalog")
			},
			func(agent *api.Health) {
				fmt.Println("health")
			},
		),
	)

	// Output:
	// client
	// agent
	// catalog
	// health
}
