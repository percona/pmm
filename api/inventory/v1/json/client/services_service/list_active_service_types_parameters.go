// Code generated by go-swagger; DO NOT EDIT.

package services_service

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

// NewListActiveServiceTypesParams creates a new ListActiveServiceTypesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListActiveServiceTypesParams() *ListActiveServiceTypesParams {
	return &ListActiveServiceTypesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListActiveServiceTypesParamsWithTimeout creates a new ListActiveServiceTypesParams object
// with the ability to set a timeout on a request.
func NewListActiveServiceTypesParamsWithTimeout(timeout time.Duration) *ListActiveServiceTypesParams {
	return &ListActiveServiceTypesParams{
		timeout: timeout,
	}
}

// NewListActiveServiceTypesParamsWithContext creates a new ListActiveServiceTypesParams object
// with the ability to set a context for a request.
func NewListActiveServiceTypesParamsWithContext(ctx context.Context) *ListActiveServiceTypesParams {
	return &ListActiveServiceTypesParams{
		Context: ctx,
	}
}

// NewListActiveServiceTypesParamsWithHTTPClient creates a new ListActiveServiceTypesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListActiveServiceTypesParamsWithHTTPClient(client *http.Client) *ListActiveServiceTypesParams {
	return &ListActiveServiceTypesParams{
		HTTPClient: client,
	}
}

/*
ListActiveServiceTypesParams contains all the parameters to send to the API endpoint

	for the list active service types operation.

	Typically these are written to a http.Request.
*/
type ListActiveServiceTypesParams struct {
	// Body.
	Body interface{}

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list active service types params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListActiveServiceTypesParams) WithDefaults() *ListActiveServiceTypesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list active service types params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListActiveServiceTypesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list active service types params
func (o *ListActiveServiceTypesParams) WithTimeout(timeout time.Duration) *ListActiveServiceTypesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list active service types params
func (o *ListActiveServiceTypesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list active service types params
func (o *ListActiveServiceTypesParams) WithContext(ctx context.Context) *ListActiveServiceTypesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list active service types params
func (o *ListActiveServiceTypesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list active service types params
func (o *ListActiveServiceTypesParams) WithHTTPClient(client *http.Client) *ListActiveServiceTypesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list active service types params
func (o *ListActiveServiceTypesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the list active service types params
func (o *ListActiveServiceTypesParams) WithBody(body interface{}) *ListActiveServiceTypesParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the list active service types params
func (o *ListActiveServiceTypesParams) SetBody(body interface{}) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *ListActiveServiceTypesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {
	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}