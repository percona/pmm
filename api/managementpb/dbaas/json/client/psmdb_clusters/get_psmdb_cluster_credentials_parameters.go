// Code generated by go-swagger; DO NOT EDIT.

package psmdb_clusters

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewGetPSMDBClusterCredentialsParams creates a new GetPSMDBClusterCredentialsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetPSMDBClusterCredentialsParams() *GetPSMDBClusterCredentialsParams {
	return &GetPSMDBClusterCredentialsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetPSMDBClusterCredentialsParamsWithTimeout creates a new GetPSMDBClusterCredentialsParams object
// with the ability to set a timeout on a request.
func NewGetPSMDBClusterCredentialsParamsWithTimeout(timeout time.Duration) *GetPSMDBClusterCredentialsParams {
	return &GetPSMDBClusterCredentialsParams{
		timeout: timeout,
	}
}

// NewGetPSMDBClusterCredentialsParamsWithContext creates a new GetPSMDBClusterCredentialsParams object
// with the ability to set a context for a request.
func NewGetPSMDBClusterCredentialsParamsWithContext(ctx context.Context) *GetPSMDBClusterCredentialsParams {
	return &GetPSMDBClusterCredentialsParams{
		Context: ctx,
	}
}

// NewGetPSMDBClusterCredentialsParamsWithHTTPClient creates a new GetPSMDBClusterCredentialsParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetPSMDBClusterCredentialsParamsWithHTTPClient(client *http.Client) *GetPSMDBClusterCredentialsParams {
	return &GetPSMDBClusterCredentialsParams{
		HTTPClient: client,
	}
}

/*
GetPSMDBClusterCredentialsParams contains all the parameters to send to the API endpoint

	for the get PSMDB cluster credentials operation.

	Typically these are written to a http.Request.
*/
type GetPSMDBClusterCredentialsParams struct {
	// Body.
	Body GetPSMDBClusterCredentialsBody

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get PSMDB cluster credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetPSMDBClusterCredentialsParams) WithDefaults() *GetPSMDBClusterCredentialsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get PSMDB cluster credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetPSMDBClusterCredentialsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get PSMDB cluster credentials params
func (o *GetPSMDBClusterCredentialsParams) WithTimeout(timeout time.Duration) *GetPSMDBClusterCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get PSMDB cluster credentials params
func (o *GetPSMDBClusterCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get PSMDB cluster credentials params
func (o *GetPSMDBClusterCredentialsParams) WithContext(ctx context.Context) *GetPSMDBClusterCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get PSMDB cluster credentials params
func (o *GetPSMDBClusterCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get PSMDB cluster credentials params
func (o *GetPSMDBClusterCredentialsParams) WithHTTPClient(client *http.Client) *GetPSMDBClusterCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get PSMDB cluster credentials params
func (o *GetPSMDBClusterCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the get PSMDB cluster credentials params
func (o *GetPSMDBClusterCredentialsParams) WithBody(body GetPSMDBClusterCredentialsBody) *GetPSMDBClusterCredentialsParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the get PSMDB cluster credentials params
func (o *GetPSMDBClusterCredentialsParams) SetBody(body GetPSMDBClusterCredentialsBody) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *GetPSMDBClusterCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {
	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if err := r.SetBodyParam(o.Body); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}