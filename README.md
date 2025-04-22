# praetor

praetor integrates go.uber.org/fx with consul.

[![Build Status](https://github.com/xmidt-org/praetor/workflows/CI/badge.svg)](https://github.com/xmidt-org/praetor/actions)
[![codecov.io](http://codecov.io/github/xmidt-org/praetor/coverage.svg?branch=main)](http://codecov.io/github/xmidt-org/praetor?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/praetor)](https://goreportcard.com/report/github.com/xmidt-org/praetor)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/praetor/blob/main/LICENSE)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=xmidt-org_PROJECT&metric=alert_status)](https://sonarcloud.io/dashboard?id=xmidt-org_PROJECT)
[![GitHub release](https://img.shields.io/github/release/xmidt-org/praetor.svg)](CHANGELOG.md)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/xmidt-org/praetor)](https://pkg.go.dev/github.com/xmidt-org/praetor)

## Summary

Praetor provides a basic, opinionated way of integrating consul into a go.uber.org/fx application.

## Table of Contents

- [Usage](#usage)
- [Code of Conduct](#code-of-conduct)
- [Install](#install)
- [Contributing](#contributing)

## Usage

`praetor.Provide()` creates an `*api.Client` object as well as several of the commonly used services.

```go
import github.com/xmidt-org/praetor

app := fx.New(
    praetor.Provide(),
    fx.Invoke(
        // praetor.Provide() makes the following possible:
        func(client *api.Config) {
            // ...
        },
        func(agent *api.Agent) {
            // ...
        },
        func(agent *api.Catalog) {
            // ...
        },
        func(agent *api.Health) {
            // ...
        },
        func(agent *api.KV) {
            // ...
        },
    ),
)
```

If an `api.Config` is provided within the application, it will be used to create the consul client.

```go
import github.com/xmidt-org/praetor

app := fx.New(
    fx.Supply(
        // this api.Config can come from external sources
        api.Config{
            Scheme: "https",
            Address: "foobar.com",
        }
    ),
    praetor.Provide(),
)
```

A custom configuration can be easily integrated using the standard `go.uber.org/fx` tools.

```go
import github.com/xmidt-org/praetor

type MyConfiguration struct {
    Scheme string
    Address string

    // anything else desired ....
}

app := fx.New(
    fx.Supply(
        MyConfiguration{
            Scheme: "https",
            Address: "foobar.com",
        }
    ),
    praetor.Provide(),
    fx.Provide(
        // this will be used by praetor
        func(src MyConfiguration) api.Config {
            return api.Config{
                Scheme: src.Scheme,
                Address: src.Address,
            }
        },
    ),
)
```

## Code of Conduct

This project and everyone participating in it are governed by the [XMiDT Code Of Conduct](https://xmidt.io/docs/community/code_of_conduct/). 
By participating, you agree to this Code.

## Install

go get -u github.com/xmidt-org/praetor

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).
