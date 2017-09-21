// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package consul provides facilities for working with Consul.
package consul

import (
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
)

const (
	// prefix for all keys in KV operations.
	prefix = "percona/"
)

// Client represents a client for Consul API.
// All keys in KV operations are prefixed to avoid collisions.
type Client struct {
	c *api.Client
}

// NewClient creates a new client for given Consul address.
func NewClient(addr string) (*Client, error) {
	c, err := api.NewClient(&api.Config{
		Address: addr,
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot connect to Consul")
	}
	return &Client{c}, nil
}

// GetKV returns value for a given key from Consul, or nil, if key does not exist.
func (c *Client) GetKV(key string) ([]byte, error) {
	pair, _, err := c.c.KV().Get(path.Join(prefix, key), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if pair == nil {
		return nil, nil
	}
	return pair.Value, nil
}

// PutKV puts given key/value pair into Consul.
func (c *Client) PutKV(key string, value []byte) error {
	pair := &api.KVPair{Key: path.Join(prefix, key), Value: value}
	_, err := c.c.KV().Put(pair, nil)
	return errors.WithStack(err)
}
