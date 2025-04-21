// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"reflect"
	"time"

	"github.com/hashicorp/consul/api"
)

// APIConfigurer is a closure type that can translate a custom confuguration
// object into a consul api.Config.
//
// The signature of this closure is flexible.  It may return an *api.Config or
// an api.Config. However, it is always api.Config (the non-pointer type) that
// is consumed by praetor. Additionally, this closure can return an optional
// second error result.
type APIConfigurer[T any] interface {
	~func(T) api.Config |
		~func(T) (api.Config, error) |
		~func(T) *api.Config |
		~func(T) (*api.Config, error)
}

// tryAsAPIConfigurer encapsulates an attempt to convert src into a target closure.
// If no conversion is possible, this function returns false.
func tryAsAPIConfigurer[F any](src reflect.Value) (f F, ok bool) {
	ft := reflect.TypeOf(f)
	if ok = src.CanConvert(ft); ok {
		f = src.Convert(ft).Interface().(F)
	}

	return
}

// asAPIConfigurer normalizes an APIConfigurer closure into a common signature.
func asAPIConfigurer[T any, F APIConfigurer[T]](f F) func(T) (api.Config, error) {
	fv := reflect.ValueOf(f)

	if af, ok := tryAsAPIConfigurer[func(T) *api.Config](fv); ok {
		return func(cfg T) (acfg api.Config, _ error) {
			if p := af(cfg); p != nil {
				acfg = *p
			}

			return
		}
	}

	if af, ok := tryAsAPIConfigurer[func(T) (*api.Config, error)](fv); ok {
		return func(cfg T) (acfg api.Config, err error) {
			var p *api.Config
			if p, err = af(cfg); p != nil {
				acfg = *p
			}

			return
		}
	}

	if af, ok := tryAsAPIConfigurer[func(T) api.Config](fv); ok {
		return func(cfg T) (api.Config, error) {
			return af(cfg), nil
		}
	}

	// at this point, there's only (1) possible type left
	af, _ := tryAsAPIConfigurer[func(T) (api.Config, error)](fv)
	return af
}

// BasicAuthConfig holds the HTTP basic authorization credentials for Consul.
type BasicAuthConfig struct {
	// UserName is the HTTP basic auth user name.
	UserName string `json:"userName" yaml:"userName" mapstructure:"userName"`

	// Password is the HTTP basic auth user name.
	Password string `json:"password" yaml:"password" mapstructure:"password"`
}

// TLSConfig holds the TLS options supported by praetor.
type TLSConfig struct {
	// Address is the optional address of the consul server. If set, this field's value
	// is used as the TLS ServerName.
	Address string `json:"address" yaml:"address" mapstructure:"address"`

	// CAFile is the system path to a CA certificate bundle used for Consul communication.
	// Defaults to the system bundle if not specified.
	CAFile string `json:"caFile" yaml:"caFile" mapstructure:"caFile"`

	// CAPath is the system directory of CA certificates used for Consul communication.
	// Defaults to the system bundle if not specified.
	CAPath string `json:"caPath" yaml:"caPath" mapstructure:"caPath"`

	// CertificateFile is the system file for the certificate used in Consul communication.
	// If this is set, KeyFile must also be set.
	CertificateFile string `json:"certificateFile" yaml:"certificateFile" mapstructure:"certificateFile"`

	// KeyFile is the system file for the key used in Consul communication.
	// If this is set, CertificateFile must also be set.
	KeyFile string `json:"keyFile" yaml:"keyFile" mapstructure:"keyFile"`

	// InsecureSkipVerify controls whether TLS host verification is disabled.
	InsecureSkipVerify bool `json:"insecureSkipVerify" yaml:"insecureSkipVerify" mapstructure:"insecureSkipVerify"`
}

// Config is an easily unmarshalable configuration that praetor uses to create
// a consul api.Config. Fields in this struct mirror those of api.Config. This type implements
// APIConfigurer and thus can be used with ProvideConfig.
//
// This type provides a simple, externalizable configuration for consul. It is optional.
// An application can supply an api.Config directly.
type Config struct {
	// Scheme is the URI scheme of the consul server.
	Scheme string `json:"scheme" yaml:"scheme" mapstructure:"scheme"`

	// Address is the address of the consul server, including port.
	Address string `json:"address" yaml:"address" mapstructure:"address"`

	// PathPrefix is the URI path prefix to use when consul is behind an API gateway.
	PathPrefix string `json:"pathPrefix" yaml:"pathPrefix" mapstructure:"pathPrefix"`

	// Datacenter is the optional datacenter to use when interacting with the agent.
	// If unset, the datacenter of the agent is used.
	Datacenter string `json:"datacenter" yaml:"datacenter" mapstructure:"datacenter"`

	// WaitTime specifies the time that watches will block. If unset, the agent's
	// default will be used.
	WaitTime time.Duration `json:"waitTime" yaml:"waitTime" mapstructure:"waitTime"`

	// Token is a per request ACL token. If unset, the agent's token is used.
	Token string `json:"token" yaml:"token" mapstructure:"token"`

	// TokenFile is a file containing the per request ACL token.
	TokenFile string `json:"tokenFile" yaml:"tokenFile" mapstructure:"tokenFile"`

	// Namespace is the namespace to send to the agent in requests where no namespace is set.
	Namespace string `json:"namespace" yaml:"namespace" mapstructure:"namespace"`

	// Partition is the partition to send to the agent in requests where no namespace is set.
	Partition string `json:"partition" yaml:"partition" mapstructure:"partition"`

	// BasicAuth defines the HTTP basic credentials for interacting with the agent.
	BasicAuth BasicAuthConfig `json:"basicAuth" yaml:"basicAuth" mapstructure:"basicAuth"`

	// TLS defines the TLS configuration to use for the consul server.
	TLS TLSConfig `json:"tls" yaml:"tls" mapstructure:"tls"`
}

// newAPIConfig is an APIConfigurer that can be passed to ProvideCustomConfig.
func newAPIConfig(src Config) (dst api.Config) {
	dst = api.Config{
		Scheme:     src.Scheme,
		Address:    src.Address,
		PathPrefix: src.PathPrefix,
		Datacenter: src.Datacenter,
		WaitTime:   src.WaitTime,
		Token:      src.Token,
		TokenFile:  src.TokenFile,
		Namespace:  src.Namespace,
		Partition:  src.Partition,
		TLSConfig: api.TLSConfig{
			Address:            src.TLS.Address,
			CAFile:             src.TLS.CAFile,
			CAPath:             src.TLS.CAPath,
			CertFile:           src.TLS.CertificateFile,
			KeyFile:            src.TLS.KeyFile,
			InsecureSkipVerify: src.TLS.InsecureSkipVerify,
		},
	}

	if len(src.BasicAuth.UserName) > 0 {
		dst.HttpAuth = &api.HttpBasicAuth{
			Username: src.BasicAuth.UserName,
			Password: src.BasicAuth.Password,
		}
	}

	return
}
