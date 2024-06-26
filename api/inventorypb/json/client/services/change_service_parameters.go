// Code generated by go-swagger; DO NOT EDIT.

package services

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

// NewChangeServiceParams creates a new ChangeServiceParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewChangeServiceParams() *ChangeServiceParams {
	return &ChangeServiceParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewChangeServiceParamsWithTimeout creates a new ChangeServiceParams object
// with the ability to set a timeout on a request.
func NewChangeServiceParamsWithTimeout(timeout time.Duration) *ChangeServiceParams {
	return &ChangeServiceParams{
		timeout: timeout,
	}
}

// NewChangeServiceParamsWithContext creates a new ChangeServiceParams object
// with the ability to set a context for a request.
func NewChangeServiceParamsWithContext(ctx context.Context) *ChangeServiceParams {
	return &ChangeServiceParams{
		Context: ctx,
	}
}

// NewChangeServiceParamsWithHTTPClient creates a new ChangeServiceParams object
// with the ability to set a custom HTTPClient for a request.
func NewChangeServiceParamsWithHTTPClient(client *http.Client) *ChangeServiceParams {
	return &ChangeServiceParams{
		HTTPClient: client,
	}
}

/*
ChangeServiceParams contains all the parameters to send to the API endpoint

	for the change service operation.

	Typically these are written to a http.Request.
*/
type ChangeServiceParams struct {
	// Body.
	Body ChangeServiceBody

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the change service params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ChangeServiceParams) WithDefaults() *ChangeServiceParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the change service params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ChangeServiceParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the change service params
func (o *ChangeServiceParams) WithTimeout(timeout time.Duration) *ChangeServiceParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the change service params
func (o *ChangeServiceParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the change service params
func (o *ChangeServiceParams) WithContext(ctx context.Context) *ChangeServiceParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the change service params
func (o *ChangeServiceParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the change service params
func (o *ChangeServiceParams) WithHTTPClient(client *http.Client) *ChangeServiceParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the change service params
func (o *ChangeServiceParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the change service params
func (o *ChangeServiceParams) WithBody(body ChangeServiceBody) *ChangeServiceParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the change service params
func (o *ChangeServiceParams) SetBody(body ChangeServiceBody) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *ChangeServiceParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {
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
