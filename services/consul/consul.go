package consul

import (
	"path/filepath"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
)

const (
	// Prefix for all keys in KV operations.
	Prefix = "/percona/"
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

// GetKV returns value for a given key from Consul.
func (c *Client) GetKV(key string) ([]byte, error) {
	pair, _, err := c.c.KV().Get(filepath.Join(Prefix, key), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return pair.Value, nil
}

// PutKV puts given key/value pair into Consul.
func (c *Client) PutKV(key string, value []byte) error {
	pair := &api.KVPair{Key: filepath.Join(Prefix, key), Value: value}
	_, err := c.c.KV().Put(pair, nil)
	return errors.WithStack(err)
}
