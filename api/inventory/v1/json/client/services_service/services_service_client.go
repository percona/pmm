// Code generated by go-swagger; DO NOT EDIT.

package services_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new services service API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for services service API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	AddCustomLabels(params *AddCustomLabelsParams, opts ...ClientOption) (*AddCustomLabelsOK, error)

	AddExternalService(params *AddExternalServiceParams, opts ...ClientOption) (*AddExternalServiceOK, error)

	AddHAProxyService(params *AddHAProxyServiceParams, opts ...ClientOption) (*AddHAProxyServiceOK, error)

	AddMongoDBService(params *AddMongoDBServiceParams, opts ...ClientOption) (*AddMongoDBServiceOK, error)

	AddMySQLService(params *AddMySQLServiceParams, opts ...ClientOption) (*AddMySQLServiceOK, error)

	AddPostgreSQLService(params *AddPostgreSQLServiceParams, opts ...ClientOption) (*AddPostgreSQLServiceOK, error)

	AddProxySQLService(params *AddProxySQLServiceParams, opts ...ClientOption) (*AddProxySQLServiceOK, error)

	ChangeService(params *ChangeServiceParams, opts ...ClientOption) (*ChangeServiceOK, error)

	GetService(params *GetServiceParams, opts ...ClientOption) (*GetServiceOK, error)

	ListActiveServiceTypes(params *ListActiveServiceTypesParams, opts ...ClientOption) (*ListActiveServiceTypesOK, error)

	ListServices(params *ListServicesParams, opts ...ClientOption) (*ListServicesOK, error)

	RemoveCustomLabels(params *RemoveCustomLabelsParams, opts ...ClientOption) (*RemoveCustomLabelsOK, error)

	RemoveService(params *RemoveServiceParams, opts ...ClientOption) (*RemoveServiceOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
AddCustomLabels adds replace custom labels

Adds or replaces (if the key exists) custom labels for a Service.
*/
func (a *Client) AddCustomLabels(params *AddCustomLabelsParams, opts ...ClientOption) (*AddCustomLabelsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddCustomLabelsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "AddCustomLabels",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/CustomLabels/Add",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &AddCustomLabelsReader{formats: a.formats},
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
	success, ok := result.(*AddCustomLabelsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*AddCustomLabelsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
AddExternalService adds external service

Adds External Service.
*/
func (a *Client) AddExternalService(params *AddExternalServiceParams, opts ...ClientOption) (*AddExternalServiceOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddExternalServiceParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "AddExternalService",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/AddExternalService",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &AddExternalServiceReader{formats: a.formats},
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
	success, ok := result.(*AddExternalServiceOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*AddExternalServiceDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
AddHAProxyService adds HA proxy service

Adds HAProxy Service.
*/
func (a *Client) AddHAProxyService(params *AddHAProxyServiceParams, opts ...ClientOption) (*AddHAProxyServiceOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddHAProxyServiceParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "AddHAProxyService",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/AddHAProxyService",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &AddHAProxyServiceReader{formats: a.formats},
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
	success, ok := result.(*AddHAProxyServiceOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*AddHAProxyServiceDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
AddMongoDBService adds mongo DB service

Adds MongoDB Service.
*/
func (a *Client) AddMongoDBService(params *AddMongoDBServiceParams, opts ...ClientOption) (*AddMongoDBServiceOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddMongoDBServiceParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "AddMongoDBService",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/AddMongoDB",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &AddMongoDBServiceReader{formats: a.formats},
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
	success, ok := result.(*AddMongoDBServiceOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*AddMongoDBServiceDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
AddMySQLService adds my SQL service

Adds MySQL Service.
*/
func (a *Client) AddMySQLService(params *AddMySQLServiceParams, opts ...ClientOption) (*AddMySQLServiceOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddMySQLServiceParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "AddMySQLService",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/AddMySQL",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &AddMySQLServiceReader{formats: a.formats},
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
	success, ok := result.(*AddMySQLServiceOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*AddMySQLServiceDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
AddPostgreSQLService adds postgre SQL service

Adds PostgreSQL Service.
*/
func (a *Client) AddPostgreSQLService(params *AddPostgreSQLServiceParams, opts ...ClientOption) (*AddPostgreSQLServiceOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddPostgreSQLServiceParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "AddPostgreSQLService",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/AddPostgreSQL",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &AddPostgreSQLServiceReader{formats: a.formats},
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
	success, ok := result.(*AddPostgreSQLServiceOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*AddPostgreSQLServiceDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
AddProxySQLService adds proxy SQL service

Adds ProxySQL Service.
*/
func (a *Client) AddProxySQLService(params *AddProxySQLServiceParams, opts ...ClientOption) (*AddProxySQLServiceOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddProxySQLServiceParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "AddProxySQLService",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/AddProxySQL",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &AddProxySQLServiceReader{formats: a.formats},
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
	success, ok := result.(*AddProxySQLServiceOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*AddProxySQLServiceDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ChangeService changes service

Changes service configuration. If a new cluster label is specified, it removes all backup/restore tasks scheduled for the related services. Fails if there are running backup/restore tasks.
*/
func (a *Client) ChangeService(params *ChangeServiceParams, opts ...ClientOption) (*ChangeServiceOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewChangeServiceParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "ChangeService",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/Change",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ChangeServiceReader{formats: a.formats},
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
	success, ok := result.(*ChangeServiceOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ChangeServiceDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetService gets service

Returns a single Service by ID.
*/
func (a *Client) GetService(params *GetServiceParams, opts ...ClientOption) (*GetServiceOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetServiceParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "GetService",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/Get",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &GetServiceReader{formats: a.formats},
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
	success, ok := result.(*GetServiceOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetServiceDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ListActiveServiceTypes lists active service types

Returns a list of active Service types.
*/
func (a *Client) ListActiveServiceTypes(params *ListActiveServiceTypesParams, opts ...ClientOption) (*ListActiveServiceTypesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListActiveServiceTypesParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "ListActiveServiceTypes",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/ListTypes",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListActiveServiceTypesReader{formats: a.formats},
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
	success, ok := result.(*ListActiveServiceTypesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListActiveServiceTypesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ListServices lists services

Returns a list of Services filtered by type.
*/
func (a *Client) ListServices(params *ListServicesParams, opts ...ClientOption) (*ListServicesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListServicesParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "ListServices",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/List",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListServicesReader{formats: a.formats},
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
	success, ok := result.(*ListServicesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListServicesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
RemoveCustomLabels removes custom labels

Removes custom labels from a Service by key.
*/
func (a *Client) RemoveCustomLabels(params *RemoveCustomLabelsParams, opts ...ClientOption) (*RemoveCustomLabelsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewRemoveCustomLabelsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "RemoveCustomLabels",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/CustomLabels/Remove",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &RemoveCustomLabelsReader{formats: a.formats},
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
	success, ok := result.(*RemoveCustomLabelsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*RemoveCustomLabelsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
RemoveService removes service

Removes Service.
*/
func (a *Client) RemoveService(params *RemoveServiceParams, opts ...ClientOption) (*RemoveServiceOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewRemoveServiceParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "RemoveService",
		Method:             "POST",
		PathPattern:        "/v1/inventory/Services/Remove",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &RemoveServiceReader{formats: a.formats},
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
	success, ok := result.(*RemoveServiceOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*RemoveServiceDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}