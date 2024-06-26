// Code generated by go-swagger; DO NOT EDIT.

package role

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

// DeleteRoleReader is a Reader for the DeleteRole structure.
type DeleteRoleReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteRoleReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteRoleOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewDeleteRoleDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteRoleOK creates a DeleteRoleOK with default headers values
func NewDeleteRoleOK() *DeleteRoleOK {
	return &DeleteRoleOK{}
}

/*
DeleteRoleOK describes a response with status code 200, with default header values.

A successful response.
*/
type DeleteRoleOK struct {
	Payload interface{}
}

func (o *DeleteRoleOK) Error() string {
	return fmt.Sprintf("[POST /v1/management/Role/Delete][%d] deleteRoleOk  %+v", 200, o.Payload)
}

func (o *DeleteRoleOK) GetPayload() interface{} {
	return o.Payload
}

func (o *DeleteRoleOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {
	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewDeleteRoleDefault creates a DeleteRoleDefault with default headers values
func NewDeleteRoleDefault(code int) *DeleteRoleDefault {
	return &DeleteRoleDefault{
		_statusCode: code,
	}
}

/*
DeleteRoleDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type DeleteRoleDefault struct {
	_statusCode int

	Payload *DeleteRoleDefaultBody
}

// Code gets the status code for the delete role default response
func (o *DeleteRoleDefault) Code() int {
	return o._statusCode
}

func (o *DeleteRoleDefault) Error() string {
	return fmt.Sprintf("[POST /v1/management/Role/Delete][%d] DeleteRole default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteRoleDefault) GetPayload() *DeleteRoleDefaultBody {
	return o.Payload
}

func (o *DeleteRoleDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {
	o.Payload = new(DeleteRoleDefaultBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
DeleteRoleBody delete role body
swagger:model DeleteRoleBody
*/
type DeleteRoleBody struct {
	// role id
	RoleID int64 `json:"role_id,omitempty"`

	// Role ID to be used as a replacement for the role. Additional logic applies.
	ReplacementRoleID int64 `json:"replacement_role_id,omitempty"`
}

// Validate validates this delete role body
func (o *DeleteRoleBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this delete role body based on context it is used
func (o *DeleteRoleBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *DeleteRoleBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *DeleteRoleBody) UnmarshalBinary(b []byte) error {
	var res DeleteRoleBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
DeleteRoleDefaultBody delete role default body
swagger:model DeleteRoleDefaultBody
*/
type DeleteRoleDefaultBody struct {
	// code
	Code int32 `json:"code,omitempty"`

	// message
	Message string `json:"message,omitempty"`

	// details
	Details []*DeleteRoleDefaultBodyDetailsItems0 `json:"details"`
}

// Validate validates this delete role default body
func (o *DeleteRoleDefaultBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateDetails(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *DeleteRoleDefaultBody) validateDetails(formats strfmt.Registry) error {
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
					return ve.ValidateName("DeleteRole default" + "." + "details" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("DeleteRole default" + "." + "details" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this delete role default body based on the context it is used
func (o *DeleteRoleDefaultBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateDetails(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *DeleteRoleDefaultBody) contextValidateDetails(ctx context.Context, formats strfmt.Registry) error {
	for i := 0; i < len(o.Details); i++ {
		if o.Details[i] != nil {
			if err := o.Details[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("DeleteRole default" + "." + "details" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("DeleteRole default" + "." + "details" + "." + strconv.Itoa(i))
				}
				return err
			}
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (o *DeleteRoleDefaultBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *DeleteRoleDefaultBody) UnmarshalBinary(b []byte) error {
	var res DeleteRoleDefaultBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
DeleteRoleDefaultBodyDetailsItems0 delete role default body details items0
swagger:model DeleteRoleDefaultBodyDetailsItems0
*/
type DeleteRoleDefaultBodyDetailsItems0 struct {
	// at type
	AtType string `json:"@type,omitempty"`
}

// Validate validates this delete role default body details items0
func (o *DeleteRoleDefaultBodyDetailsItems0) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this delete role default body details items0 based on context it is used
func (o *DeleteRoleDefaultBodyDetailsItems0) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *DeleteRoleDefaultBodyDetailsItems0) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *DeleteRoleDefaultBodyDetailsItems0) UnmarshalBinary(b []byte) error {
	var res DeleteRoleDefaultBodyDetailsItems0
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
