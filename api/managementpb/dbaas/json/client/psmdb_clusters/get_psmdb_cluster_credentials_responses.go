// Code generated by go-swagger; DO NOT EDIT.

package psmdb_clusters

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// GetPSMDBClusterCredentialsReader is a Reader for the GetPSMDBClusterCredentials structure.
type GetPSMDBClusterCredentialsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetPSMDBClusterCredentialsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetPSMDBClusterCredentialsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewGetPSMDBClusterCredentialsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetPSMDBClusterCredentialsOK creates a GetPSMDBClusterCredentialsOK with default headers values
func NewGetPSMDBClusterCredentialsOK() *GetPSMDBClusterCredentialsOK {
	return &GetPSMDBClusterCredentialsOK{}
}

/*
GetPSMDBClusterCredentialsOK describes a response with status code 200, with default header values.

A successful response.
*/
type GetPSMDBClusterCredentialsOK struct {
	Payload *GetPSMDBClusterCredentialsOKBody
}

func (o *GetPSMDBClusterCredentialsOK) Error() string {
	return fmt.Sprintf("[POST /v1/management/DBaaS/PSMDBClusters/GetCredentials][%d] getPsmdbClusterCredentialsOk  %+v", 200, o.Payload)
}

func (o *GetPSMDBClusterCredentialsOK) GetPayload() *GetPSMDBClusterCredentialsOKBody {
	return o.Payload
}

func (o *GetPSMDBClusterCredentialsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {
	o.Payload = new(GetPSMDBClusterCredentialsOKBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetPSMDBClusterCredentialsDefault creates a GetPSMDBClusterCredentialsDefault with default headers values
func NewGetPSMDBClusterCredentialsDefault(code int) *GetPSMDBClusterCredentialsDefault {
	return &GetPSMDBClusterCredentialsDefault{
		_statusCode: code,
	}
}

/*
GetPSMDBClusterCredentialsDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type GetPSMDBClusterCredentialsDefault struct {
	_statusCode int

	Payload *GetPSMDBClusterCredentialsDefaultBody
}

// Code gets the status code for the get PSMDB cluster credentials default response
func (o *GetPSMDBClusterCredentialsDefault) Code() int {
	return o._statusCode
}

func (o *GetPSMDBClusterCredentialsDefault) Error() string {
	return fmt.Sprintf("[POST /v1/management/DBaaS/PSMDBClusters/GetCredentials][%d] GetPSMDBClusterCredentials default  %+v", o._statusCode, o.Payload)
}

func (o *GetPSMDBClusterCredentialsDefault) GetPayload() *GetPSMDBClusterCredentialsDefaultBody {
	return o.Payload
}

func (o *GetPSMDBClusterCredentialsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {
	o.Payload = new(GetPSMDBClusterCredentialsDefaultBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
GetPSMDBClusterCredentialsBody get PSMDB cluster credentials body
swagger:model GetPSMDBClusterCredentialsBody
*/
type GetPSMDBClusterCredentialsBody struct {
	// Kubernetes cluster name.
	KubernetesClusterName string `json:"kubernetes_cluster_name,omitempty"`

	// PSMDB cluster name.
	Name string `json:"name,omitempty"`
}

// Validate validates this get PSMDB cluster credentials body
func (o *GetPSMDBClusterCredentialsBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this get PSMDB cluster credentials body based on context it is used
func (o *GetPSMDBClusterCredentialsBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *GetPSMDBClusterCredentialsBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetPSMDBClusterCredentialsBody) UnmarshalBinary(b []byte) error {
	var res GetPSMDBClusterCredentialsBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
GetPSMDBClusterCredentialsDefaultBody get PSMDB cluster credentials default body
swagger:model GetPSMDBClusterCredentialsDefaultBody
*/
type GetPSMDBClusterCredentialsDefaultBody struct {
	// code
	Code int32 `json:"code,omitempty"`

	// message
	Message string `json:"message,omitempty"`

	// details
	Details []*GetPSMDBClusterCredentialsDefaultBodyDetailsItems0 `json:"details"`
}

// Validate validates this get PSMDB cluster credentials default body
func (o *GetPSMDBClusterCredentialsDefaultBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateDetails(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *GetPSMDBClusterCredentialsDefaultBody) validateDetails(formats strfmt.Registry) error {
	if swag.IsZero(o.Details) { // not required
		return nil
	}

	for i := 0; i < len(o.Details); i++ {
		if swag.IsZero(o.Details[i]) { // not required
			continue
		}

		if o.Details[i] != nil {
			if err := o.Details[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("GetPSMDBClusterCredentials default" + "." + "details" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("GetPSMDBClusterCredentials default" + "." + "details" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this get PSMDB cluster credentials default body based on the context it is used
func (o *GetPSMDBClusterCredentialsDefaultBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateDetails(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *GetPSMDBClusterCredentialsDefaultBody) contextValidateDetails(ctx context.Context, formats strfmt.Registry) error {
	for i := 0; i < len(o.Details); i++ {
		if o.Details[i] != nil {
			if err := o.Details[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("GetPSMDBClusterCredentials default" + "." + "details" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("GetPSMDBClusterCredentials default" + "." + "details" + "." + strconv.Itoa(i))
				}
				return err
			}
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (o *GetPSMDBClusterCredentialsDefaultBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetPSMDBClusterCredentialsDefaultBody) UnmarshalBinary(b []byte) error {
	var res GetPSMDBClusterCredentialsDefaultBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
GetPSMDBClusterCredentialsDefaultBodyDetailsItems0 get PSMDB cluster credentials default body details items0
swagger:model GetPSMDBClusterCredentialsDefaultBodyDetailsItems0
*/
type GetPSMDBClusterCredentialsDefaultBodyDetailsItems0 struct {
	// at type
	AtType string `json:"@type,omitempty"`
}

// Validate validates this get PSMDB cluster credentials default body details items0
func (o *GetPSMDBClusterCredentialsDefaultBodyDetailsItems0) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this get PSMDB cluster credentials default body details items0 based on context it is used
func (o *GetPSMDBClusterCredentialsDefaultBodyDetailsItems0) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *GetPSMDBClusterCredentialsDefaultBodyDetailsItems0) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetPSMDBClusterCredentialsDefaultBodyDetailsItems0) UnmarshalBinary(b []byte) error {
	var res GetPSMDBClusterCredentialsDefaultBodyDetailsItems0
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
GetPSMDBClusterCredentialsOKBody get PSMDB cluster credentials OK body
swagger:model GetPSMDBClusterCredentialsOKBody
*/
type GetPSMDBClusterCredentialsOKBody struct {
	// connection credentials
	ConnectionCredentials *GetPSMDBClusterCredentialsOKBodyConnectionCredentials `json:"connection_credentials,omitempty"`
}

// Validate validates this get PSMDB cluster credentials OK body
func (o *GetPSMDBClusterCredentialsOKBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateConnectionCredentials(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *GetPSMDBClusterCredentialsOKBody) validateConnectionCredentials(formats strfmt.Registry) error {
	if swag.IsZero(o.ConnectionCredentials) { // not required
		return nil
	}

	if o.ConnectionCredentials != nil {
		if err := o.ConnectionCredentials.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("getPsmdbClusterCredentialsOk" + "." + "connection_credentials")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("getPsmdbClusterCredentialsOk" + "." + "connection_credentials")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this get PSMDB cluster credentials OK body based on the context it is used
func (o *GetPSMDBClusterCredentialsOKBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateConnectionCredentials(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *GetPSMDBClusterCredentialsOKBody) contextValidateConnectionCredentials(ctx context.Context, formats strfmt.Registry) error {
	if o.ConnectionCredentials != nil {
		if err := o.ConnectionCredentials.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("getPsmdbClusterCredentialsOk" + "." + "connection_credentials")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("getPsmdbClusterCredentialsOk" + "." + "connection_credentials")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (o *GetPSMDBClusterCredentialsOKBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetPSMDBClusterCredentialsOKBody) UnmarshalBinary(b []byte) error {
	var res GetPSMDBClusterCredentialsOKBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
GetPSMDBClusterCredentialsOKBodyConnectionCredentials PSMDBCredentials is a credentials to connect to PSMDB.
// TODO Do not use inner messages in all public APIs (for consistency).
swagger:model GetPSMDBClusterCredentialsOKBodyConnectionCredentials
*/
type GetPSMDBClusterCredentialsOKBodyConnectionCredentials struct {
	// MongoDB username.
	Username string `json:"username,omitempty"`

	// MongoDB password.
	Password string `json:"password,omitempty"`

	// MongoDB host.
	Host string `json:"host,omitempty"`

	// MongoDB port.
	Port int32 `json:"port,omitempty"`

	// Replicaset name.
	Replicaset string `json:"replicaset,omitempty"`
}

// Validate validates this get PSMDB cluster credentials OK body connection credentials
func (o *GetPSMDBClusterCredentialsOKBodyConnectionCredentials) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this get PSMDB cluster credentials OK body connection credentials based on context it is used
func (o *GetPSMDBClusterCredentialsOKBodyConnectionCredentials) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *GetPSMDBClusterCredentialsOKBodyConnectionCredentials) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetPSMDBClusterCredentialsOKBodyConnectionCredentials) UnmarshalBinary(b []byte) error {
	var res GetPSMDBClusterCredentialsOKBodyConnectionCredentials
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}