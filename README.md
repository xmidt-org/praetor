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

Praetor provides a basic, opinionated way of integrating consul into a go.uber.org/fx application.  In particular, it allows for external configuration to drive service registration and discovery.  Praetor also binds service registration and discovery to the application lifecycle.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Install](#install)
- [Contributing](#contributing)

## Code of Conduct

This project and everyone participating in it are governed by the [XMiDT Code Of Conduct](https://xmidt.io/docs/community/code_of_conduct/). 
By participating, you agree to this Code.

## Install

go get -u github.com/xmidt-org/praetor

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).
