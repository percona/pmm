// Code generated by go-swagger; DO NOT EDIT.

package proxy_sql_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new proxy sql service API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for proxy sql service API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	AddProxySQL(params *AddProxySQLParams, opts ...ClientOption) (*AddProxySQLOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
AddProxySQL adds proxy SQL

Adds ProxySQL Service and starts several Agents. It automatically adds a service to inventory, which is running on provided "node_id", then adds "proxysql_exporter" with provided "pmm_agent_id" and other parameters.
*/
func (a *Client) AddProxySQL(params *AddProxySQLParams, opts ...ClientOption) (*AddProxySQLOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddProxySQLParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "AddProxySQL",
		Method:             "POST",
		PathPattern:        "/v1/management/ProxySQL/Add",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &AddProxySQLReader{formats: a.formats},
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*AddProxySQLOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*AddProxySQLDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}