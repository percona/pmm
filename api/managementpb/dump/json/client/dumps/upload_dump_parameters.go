// Code generated by go-swagger; DO NOT EDIT.

package dumps

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

// NewUploadDumpParams creates a new UploadDumpParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewUploadDumpParams() *UploadDumpParams {
	return &UploadDumpParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewUploadDumpParamsWithTimeout creates a new UploadDumpParams object
// with the ability to set a timeout on a request.
func NewUploadDumpParamsWithTimeout(timeout time.Duration) *UploadDumpParams {
	return &UploadDumpParams{
		timeout: timeout,
	}
}

// NewUploadDumpParamsWithContext creates a new UploadDumpParams object
// with the ability to set a context for a request.
func NewUploadDumpParamsWithContext(ctx context.Context) *UploadDumpParams {
	return &UploadDumpParams{
		Context: ctx,
	}
}

// NewUploadDumpParamsWithHTTPClient creates a new UploadDumpParams object
// with the ability to set a custom HTTPClient for a request.
func NewUploadDumpParamsWithHTTPClient(client *http.Client) *UploadDumpParams {
	return &UploadDumpParams{
		HTTPClient: client,
	}
}

/*
UploadDumpParams contains all the parameters to send to the API endpoint

	for the upload dump operation.

	Typically these are written to a http.Request.
*/
type UploadDumpParams struct {
	// Body.
	Body UploadDumpBody

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the upload dump params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UploadDumpParams) WithDefaults() *UploadDumpParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the upload dump params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UploadDumpParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the upload dump params
func (o *UploadDumpParams) WithTimeout(timeout time.Duration) *UploadDumpParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the upload dump params
func (o *UploadDumpParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the upload dump params
func (o *UploadDumpParams) WithContext(ctx context.Context) *UploadDumpParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the upload dump params
func (o *UploadDumpParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the upload dump params
func (o *UploadDumpParams) WithHTTPClient(client *http.Client) *UploadDumpParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the upload dump params
func (o *UploadDumpParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the upload dump params
func (o *UploadDumpParams) WithBody(body UploadDumpBody) *UploadDumpParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the upload dump params
func (o *UploadDumpParams) SetBody(body UploadDumpBody) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *UploadDumpParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {
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