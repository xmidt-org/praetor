// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetorsd

import "github.com/hashicorp/consul/api"

// Query represents a consul service query.
type Query struct {
	Service     string
	Tags        []string
	PassingOnly bool
	Options     *api.QueryOptions
}

// Service is the praetor representation of a consul service.  It exposes the common
// properties of services from across the various ways to query.
type Service struct {
	ID                string
	Name              string
	Tags              []string
	Meta              map[string]string
	Port              int
	Address           string
	TaggedAddresses   map[string]api.ServiceAddress
	Namespace         string
	Partition         string
	Datacenter        string
	Locality          *api.Locality
	EnableTagOverride bool

	CreateIndex uint64
	ModifyIndex uint64
}

// Answer is the answer to a service Query.
type Answer struct {
	Meta     *api.QueryMeta
	Services []Service
}

// Services is a way to query for consul services.  Implementations may be backed
// by the health endpoint, the catalog endpoint, or some arbitrary endpoint.
type Services interface {
	Get(Query) (Answer, error)
}

type healthServices struct {
	health *api.Health
}

func (hs healthServices) transform(se *api.ServiceEntry) (s Service) {
	s.ID = se.Service.ID
	s.Name = se.Service.Service
	s.Tags = se.Service.Tags
	s.Meta = se.Service.Meta
	s.Port = se.Service.Port
	s.Address = se.Service.Address
	s.TaggedAddresses = se.Service.TaggedAddresses
	s.Namespace = se.Service.Namespace
	s.Partition = se.Service.Partition
	s.Datacenter = se.Service.Datacenter
	s.Locality = se.Service.Locality
	s.EnableTagOverride = se.Service.EnableTagOverride
	s.CreateIndex = se.Service.CreateIndex
	s.ModifyIndex = se.Service.ModifyIndex

	return
}

func (hs healthServices) Get(q Query) (a Answer, err error) {
	var rawServices []*api.ServiceEntry
	rawServices, a.Meta, err = hs.health.ServiceMultipleTags(
		q.Service,
		q.Tags,
		q.PassingOnly,
		q.Options,
	)

	if err == nil {
		a.Services = make([]Service, 0, len(rawServices))
		for _, se := range rawServices {
			a.Services = append(a.Services, hs.transform(se))
		}
	}

	return
}

type catalogServices struct {
	catalog *api.Catalog
}

func (cs catalogServices) transform(c *api.CatalogService) (s Service) {
	s.ID = c.ServiceID
	s.Name = c.ServiceName
	s.Tags = c.ServiceTags
	s.Meta = c.ServiceMeta
	s.Port = c.ServicePort
	s.Address = c.ServiceAddress
	s.TaggedAddresses = c.ServiceTaggedAddresses
	s.Namespace = c.Namespace
	s.Partition = c.Partition
	s.Datacenter = c.Datacenter
	s.Locality = c.ServiceLocality
	s.EnableTagOverride = c.ServiceEnableTagOverride
	s.CreateIndex = c.CreateIndex
	s.ModifyIndex = c.ModifyIndex

	return
}

func (cs catalogServices) Get(q Query) (a Answer, err error) {
	var rawServices []*api.CatalogService
	rawServices, a.Meta, err = cs.catalog.ServiceMultipleTags(
		q.Service,
		q.Tags,
		q.Options,
	)

	if err == nil {
		a.Services = make([]Service, 0, len(rawServices))
		for _, se := range rawServices {
			a.Services = append(a.Services, cs.transform(se))
		}
	}

	return
}

// NewHealthServices produces a Services strategy backed by
// the client's Health endpoint.  The returned Services will
// honor the Query.PassingOnly flag.
func NewHealthServices(client *api.Client) Services {
	return healthServices{
		health: client.Health(),
	}
}

// NewCatalogServices produces a Services strategy backed by the
// client's Catalog endpoint.  The returned Services will *NOT*
// honor the Query.PassingOnly flag.
func NewCatalogServices(client *api.Client) Services {
	return catalogServices{
		catalog: client.Catalog(),
	}
}
