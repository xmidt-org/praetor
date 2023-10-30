package praetor

import (
	"github.com/hashicorp/consul/api"
	"github.com/xmidt-org/retry"
)

type Query struct {
	Service     string
	Tags        []string
	PassingOnly bool
	Options     *api.QueryOptions
}

// RegistrationConfig is the service registration portion of praetor's configuration.
// This will typically be obtained externally via the Config.
type RegistrationConfig struct {
	Retry    retry.Config          `json:"retry" yaml:"retry"`
	Services []ServiceRegistration `json:"services" yaml:"services"`
}

type Config struct {
	Client       api.Config         `json:"client" yaml:"client"`
	Registration RegistrationConfig `json:"registration" yaml:"registration"`
}
