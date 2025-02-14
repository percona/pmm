// Code generated by go-swagger; DO NOT EDIT.

package postgre_sql

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

// NewAddPostgreSQLParams creates a new AddPostgreSQLParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewAddPostgreSQLParams() *AddPostgreSQLParams {
	return &AddPostgreSQLParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewAddPostgreSQLParamsWithTimeout creates a new AddPostgreSQLParams object
// with the ability to set a timeout on a request.
func NewAddPostgreSQLParamsWithTimeout(timeout time.Duration) *AddPostgreSQLParams {
	return &AddPostgreSQLParams{
		timeout: timeout,
	}
}

// NewAddPostgreSQLParamsWithContext creates a new AddPostgreSQLParams object
// with the ability to set a context for a request.
func NewAddPostgreSQLParamsWithContext(ctx context.Context) *AddPostgreSQLParams {
	return &AddPostgreSQLParams{
		Context: ctx,
	}
}

// NewAddPostgreSQLParamsWithHTTPClient creates a new AddPostgreSQLParams object
// with the ability to set a custom HTTPClient for a request.
func NewAddPostgreSQLParamsWithHTTPClient(client *http.Client) *AddPostgreSQLParams {
	return &AddPostgreSQLParams{
		HTTPClient: client,
	}
}

/*
AddPostgreSQLParams contains all the parameters to send to the API endpoint

	for the add postgre SQL operation.

	Typically these are written to a http.Request.
*/
type AddPostgreSQLParams struct {
	// Body.
	Body AddPostgreSQLBody

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the add postgre SQL params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AddPostgreSQLParams) WithDefaults() *AddPostgreSQLParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the add postgre SQL params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AddPostgreSQLParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the add postgre SQL params
func (o *AddPostgreSQLParams) WithTimeout(timeout time.Duration) *AddPostgreSQLParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the add postgre SQL params
func (o *AddPostgreSQLParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the add postgre SQL params
func (o *AddPostgreSQLParams) WithContext(ctx context.Context) *AddPostgreSQLParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the add postgre SQL params
func (o *AddPostgreSQLParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the add postgre SQL params
func (o *AddPostgreSQLParams) WithHTTPClient(client *http.Client) *AddPostgreSQLParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the add postgre SQL params
func (o *AddPostgreSQLParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the add postgre SQL params
func (o *AddPostgreSQLParams) WithBody(body AddPostgreSQLBody) *AddPostgreSQLParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the add postgre SQL params
func (o *AddPostgreSQLParams) SetBody(body AddPostgreSQLBody) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *AddPostgreSQLParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {
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
