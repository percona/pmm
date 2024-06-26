// Code generated by go-swagger; DO NOT EDIT.

package kubernetes

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

// ListStorageClassesReader is a Reader for the ListStorageClasses structure.
type ListStorageClassesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListStorageClassesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListStorageClassesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListStorageClassesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListStorageClassesOK creates a ListStorageClassesOK with default headers values
func NewListStorageClassesOK() *ListStorageClassesOK {
	return &ListStorageClassesOK{}
}

/*
ListStorageClassesOK describes a response with status code 200, with default header values.

A successful response.
*/
type ListStorageClassesOK struct {
	Payload *ListStorageClassesOKBody
}

func (o *ListStorageClassesOK) Error() string {
	return fmt.Sprintf("[POST /v1/management/DBaaS/Kubernetes/StorageClasses/List][%d] listStorageClassesOk  %+v", 200, o.Payload)
}

func (o *ListStorageClassesOK) GetPayload() *ListStorageClassesOKBody {
	return o.Payload
}

func (o *ListStorageClassesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {
	o.Payload = new(ListStorageClassesOKBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListStorageClassesDefault creates a ListStorageClassesDefault with default headers values
func NewListStorageClassesDefault(code int) *ListStorageClassesDefault {
	return &ListStorageClassesDefault{
		_statusCode: code,
	}
}

/*
ListStorageClassesDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type ListStorageClassesDefault struct {
	_statusCode int

	Payload *ListStorageClassesDefaultBody
}

// Code gets the status code for the list storage classes default response
func (o *ListStorageClassesDefault) Code() int {
	return o._statusCode
}

func (o *ListStorageClassesDefault) Error() string {
	return fmt.Sprintf("[POST /v1/management/DBaaS/Kubernetes/StorageClasses/List][%d] ListStorageClasses default  %+v", o._statusCode, o.Payload)
}

func (o *ListStorageClassesDefault) GetPayload() *ListStorageClassesDefaultBody {
	return o.Payload
}

func (o *ListStorageClassesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {
	o.Payload = new(ListStorageClassesDefaultBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
ListStorageClassesBody list storage classes body
swagger:model ListStorageClassesBody
*/
type ListStorageClassesBody struct {
	// Kubernetes cluster name.
	KubernetesClusterName string `json:"kubernetes_cluster_name,omitempty"`
}

// Validate validates this list storage classes body
func (o *ListStorageClassesBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this list storage classes body based on context it is used
func (o *ListStorageClassesBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ListStorageClassesBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ListStorageClassesBody) UnmarshalBinary(b []byte) error {
	var res ListStorageClassesBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ListStorageClassesDefaultBody list storage classes default body
swagger:model ListStorageClassesDefaultBody
*/
type ListStorageClassesDefaultBody struct {
	// code
	Code int32 `json:"code,omitempty"`

	// message
	Message string `json:"message,omitempty"`

	// details
	Details []*ListStorageClassesDefaultBodyDetailsItems0 `json:"details"`
}

// Validate validates this list storage classes default body
func (o *ListStorageClassesDefaultBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateDetails(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *ListStorageClassesDefaultBody) validateDetails(formats strfmt.Registry) error {
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
					return ve.ValidateName("ListStorageClasses default" + "." + "details" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("ListStorageClasses default" + "." + "details" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this list storage classes default body based on the context it is used
func (o *ListStorageClassesDefaultBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateDetails(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *ListStorageClassesDefaultBody) contextValidateDetails(ctx context.Context, formats strfmt.Registry) error {
	for i := 0; i < len(o.Details); i++ {
		if o.Details[i] != nil {
			if err := o.Details[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("ListStorageClasses default" + "." + "details" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("ListStorageClasses default" + "." + "details" + "." + strconv.Itoa(i))
				}
				return err
			}
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (o *ListStorageClassesDefaultBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ListStorageClassesDefaultBody) UnmarshalBinary(b []byte) error {
	var res ListStorageClassesDefaultBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ListStorageClassesDefaultBodyDetailsItems0 list storage classes default body details items0
swagger:model ListStorageClassesDefaultBodyDetailsItems0
*/
type ListStorageClassesDefaultBodyDetailsItems0 struct {
	// at type
	AtType string `json:"@type,omitempty"`
}

// Validate validates this list storage classes default body details items0
func (o *ListStorageClassesDefaultBodyDetailsItems0) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this list storage classes default body details items0 based on context it is used
func (o *ListStorageClassesDefaultBodyDetailsItems0) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ListStorageClassesDefaultBodyDetailsItems0) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ListStorageClassesDefaultBodyDetailsItems0) UnmarshalBinary(b []byte) error {
	var res ListStorageClassesDefaultBodyDetailsItems0
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ListStorageClassesOKBody list storage classes OK body
swagger:model ListStorageClassesOKBody
*/
type ListStorageClassesOKBody struct {
	// Kubernetes storage classes names.
	StorageClasses []string `json:"storage_classes"`
}

// Validate validates this list storage classes OK body
func (o *ListStorageClassesOKBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this list storage classes OK body based on context it is used
func (o *ListStorageClassesOKBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ListStorageClassesOKBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ListStorageClassesOKBody) UnmarshalBinary(b []byte) error {
	var res ListStorageClassesOKBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
