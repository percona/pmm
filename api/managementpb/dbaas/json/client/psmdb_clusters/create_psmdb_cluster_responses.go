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

// CreatePSMDBClusterReader is a Reader for the CreatePSMDBCluster structure.
type CreatePSMDBClusterReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreatePSMDBClusterReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewCreatePSMDBClusterOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewCreatePSMDBClusterDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreatePSMDBClusterOK creates a CreatePSMDBClusterOK with default headers values
func NewCreatePSMDBClusterOK() *CreatePSMDBClusterOK {
	return &CreatePSMDBClusterOK{}
}

/*
CreatePSMDBClusterOK describes a response with status code 200, with default header values.

A successful response.
*/
type CreatePSMDBClusterOK struct {
	Payload interface{}
}

func (o *CreatePSMDBClusterOK) Error() string {
	return fmt.Sprintf("[POST /v1/management/DBaaS/PSMDBCluster/Create][%d] createPsmdbClusterOk  %+v", 200, o.Payload)
}

func (o *CreatePSMDBClusterOK) GetPayload() interface{} {
	return o.Payload
}

func (o *CreatePSMDBClusterOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {
	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreatePSMDBClusterDefault creates a CreatePSMDBClusterDefault with default headers values
func NewCreatePSMDBClusterDefault(code int) *CreatePSMDBClusterDefault {
	return &CreatePSMDBClusterDefault{
		_statusCode: code,
	}
}

/*
CreatePSMDBClusterDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type CreatePSMDBClusterDefault struct {
	_statusCode int

	Payload *CreatePSMDBClusterDefaultBody
}

// Code gets the status code for the create PSMDB cluster default response
func (o *CreatePSMDBClusterDefault) Code() int {
	return o._statusCode
}

func (o *CreatePSMDBClusterDefault) Error() string {
	return fmt.Sprintf("[POST /v1/management/DBaaS/PSMDBCluster/Create][%d] CreatePSMDBCluster default  %+v", o._statusCode, o.Payload)
}

func (o *CreatePSMDBClusterDefault) GetPayload() *CreatePSMDBClusterDefaultBody {
	return o.Payload
}

func (o *CreatePSMDBClusterDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {
	o.Payload = new(CreatePSMDBClusterDefaultBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
CreatePSMDBClusterBody create PSMDB cluster body
swagger:model CreatePSMDBClusterBody
*/
type CreatePSMDBClusterBody struct {
	// Kubernetes cluster name.
	KubernetesClusterName string `json:"kubernetes_cluster_name,omitempty"`

	// PSMDB cluster name.
	// a DNS-1035 label must consist of lower case alphanumeric characters or '-',
	// start with an alphabetic character, and end with an alphanumeric character
	// (e.g. 'my-name',  or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')
	Name string `json:"name,omitempty"`

	// Make DB cluster accessible outside of K8s cluster.
	Expose bool `json:"expose,omitempty"`

	// Make DB cluster accessible via public internet.
	InternetFacing bool `json:"internet_facing,omitempty"`

	// Apply IP source ranges against the cluster.
	SourceRanges []string `json:"source_ranges"`

	// params
	Params *CreatePSMDBClusterParamsBodyParams `json:"params,omitempty"`

	// template
	Template *CreatePSMDBClusterParamsBodyTemplate `json:"template,omitempty"`
}

// Validate validates this create PSMDB cluster body
func (o *CreatePSMDBClusterBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateParams(formats); err != nil {
		res = append(res, err)
	}

	if err := o.validateTemplate(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *CreatePSMDBClusterBody) validateParams(formats strfmt.Registry) error {
	if swag.IsZero(o.Params) { // not required
		return nil
	}

	if o.Params != nil {
		if err := o.Params.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "params")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "params")
			}
			return err
		}
	}

	return nil
}

func (o *CreatePSMDBClusterBody) validateTemplate(formats strfmt.Registry) error {
	if swag.IsZero(o.Template) { // not required
		return nil
	}

	if o.Template != nil {
		if err := o.Template.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "template")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "template")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this create PSMDB cluster body based on the context it is used
func (o *CreatePSMDBClusterBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateParams(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := o.contextValidateTemplate(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *CreatePSMDBClusterBody) contextValidateParams(ctx context.Context, formats strfmt.Registry) error {
	if o.Params != nil {
		if err := o.Params.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "params")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "params")
			}
			return err
		}
	}

	return nil
}

func (o *CreatePSMDBClusterBody) contextValidateTemplate(ctx context.Context, formats strfmt.Registry) error {
	if o.Template != nil {
		if err := o.Template.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "template")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "template")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (o *CreatePSMDBClusterBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreatePSMDBClusterBody) UnmarshalBinary(b []byte) error {
	var res CreatePSMDBClusterBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
CreatePSMDBClusterDefaultBody create PSMDB cluster default body
swagger:model CreatePSMDBClusterDefaultBody
*/
type CreatePSMDBClusterDefaultBody struct {
	// code
	Code int32 `json:"code,omitempty"`

	// message
	Message string `json:"message,omitempty"`

	// details
	Details []*CreatePSMDBClusterDefaultBodyDetailsItems0 `json:"details"`
}

// Validate validates this create PSMDB cluster default body
func (o *CreatePSMDBClusterDefaultBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateDetails(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *CreatePSMDBClusterDefaultBody) validateDetails(formats strfmt.Registry) error {
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
					return ve.ValidateName("CreatePSMDBCluster default" + "." + "details" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("CreatePSMDBCluster default" + "." + "details" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this create PSMDB cluster default body based on the context it is used
func (o *CreatePSMDBClusterDefaultBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateDetails(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *CreatePSMDBClusterDefaultBody) contextValidateDetails(ctx context.Context, formats strfmt.Registry) error {
	for i := 0; i < len(o.Details); i++ {
		if o.Details[i] != nil {
			if err := o.Details[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("CreatePSMDBCluster default" + "." + "details" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("CreatePSMDBCluster default" + "." + "details" + "." + strconv.Itoa(i))
				}
				return err
			}
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (o *CreatePSMDBClusterDefaultBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreatePSMDBClusterDefaultBody) UnmarshalBinary(b []byte) error {
	var res CreatePSMDBClusterDefaultBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
CreatePSMDBClusterDefaultBodyDetailsItems0 create PSMDB cluster default body details items0
swagger:model CreatePSMDBClusterDefaultBodyDetailsItems0
*/
type CreatePSMDBClusterDefaultBodyDetailsItems0 struct {
	// at type
	AtType string `json:"@type,omitempty"`
}

// Validate validates this create PSMDB cluster default body details items0
func (o *CreatePSMDBClusterDefaultBodyDetailsItems0) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this create PSMDB cluster default body details items0 based on context it is used
func (o *CreatePSMDBClusterDefaultBodyDetailsItems0) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *CreatePSMDBClusterDefaultBodyDetailsItems0) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreatePSMDBClusterDefaultBodyDetailsItems0) UnmarshalBinary(b []byte) error {
	var res CreatePSMDBClusterDefaultBodyDetailsItems0
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
CreatePSMDBClusterParamsBodyParams PSMDBClusterParams represents PSMDB cluster parameters that can be updated.
swagger:model CreatePSMDBClusterParamsBodyParams
*/
type CreatePSMDBClusterParamsBodyParams struct {
	// Cluster size.
	ClusterSize int32 `json:"cluster_size,omitempty"`

	// Docker image used for PSMDB.
	Image string `json:"image,omitempty"`

	// backup
	Backup *CreatePSMDBClusterParamsBodyParamsBackup `json:"backup,omitempty"`

	// replicaset
	Replicaset *CreatePSMDBClusterParamsBodyParamsReplicaset `json:"replicaset,omitempty"`

	// restore
	Restore *CreatePSMDBClusterParamsBodyParamsRestore `json:"restore,omitempty"`
}

// Validate validates this create PSMDB cluster params body params
func (o *CreatePSMDBClusterParamsBodyParams) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateBackup(formats); err != nil {
		res = append(res, err)
	}

	if err := o.validateReplicaset(formats); err != nil {
		res = append(res, err)
	}

	if err := o.validateRestore(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *CreatePSMDBClusterParamsBodyParams) validateBackup(formats strfmt.Registry) error {
	if swag.IsZero(o.Backup) { // not required
		return nil
	}

	if o.Backup != nil {
		if err := o.Backup.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "params" + "." + "backup")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "params" + "." + "backup")
			}
			return err
		}
	}

	return nil
}

func (o *CreatePSMDBClusterParamsBodyParams) validateReplicaset(formats strfmt.Registry) error {
	if swag.IsZero(o.Replicaset) { // not required
		return nil
	}

	if o.Replicaset != nil {
		if err := o.Replicaset.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "params" + "." + "replicaset")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "params" + "." + "replicaset")
			}
			return err
		}
	}

	return nil
}

func (o *CreatePSMDBClusterParamsBodyParams) validateRestore(formats strfmt.Registry) error {
	if swag.IsZero(o.Restore) { // not required
		return nil
	}

	if o.Restore != nil {
		if err := o.Restore.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "params" + "." + "restore")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "params" + "." + "restore")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this create PSMDB cluster params body params based on the context it is used
func (o *CreatePSMDBClusterParamsBodyParams) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateBackup(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := o.contextValidateReplicaset(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := o.contextValidateRestore(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *CreatePSMDBClusterParamsBodyParams) contextValidateBackup(ctx context.Context, formats strfmt.Registry) error {
	if o.Backup != nil {
		if err := o.Backup.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "params" + "." + "backup")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "params" + "." + "backup")
			}
			return err
		}
	}

	return nil
}

func (o *CreatePSMDBClusterParamsBodyParams) contextValidateReplicaset(ctx context.Context, formats strfmt.Registry) error {
	if o.Replicaset != nil {
		if err := o.Replicaset.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "params" + "." + "replicaset")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "params" + "." + "replicaset")
			}
			return err
		}
	}

	return nil
}

func (o *CreatePSMDBClusterParamsBodyParams) contextValidateRestore(ctx context.Context, formats strfmt.Registry) error {
	if o.Restore != nil {
		if err := o.Restore.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "params" + "." + "restore")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "params" + "." + "restore")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyParams) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyParams) UnmarshalBinary(b []byte) error {
	var res CreatePSMDBClusterParamsBodyParams
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
CreatePSMDBClusterParamsBodyParamsBackup Backup configuration for a database cluster
swagger:model CreatePSMDBClusterParamsBodyParamsBackup
*/
type CreatePSMDBClusterParamsBodyParamsBackup struct {
	// Backup Location id of stored backup location in PMM.
	LocationID string `json:"location_id,omitempty"`

	// Keep copies represents how many copies should retain.
	KeepCopies int32 `json:"keep_copies,omitempty"`

	// Cron expression represents cron expression
	CronExpression string `json:"cron_expression,omitempty"`

	// Service acccount used for backups
	ServiceAccount string `json:"service_account,omitempty"`
}

// Validate validates this create PSMDB cluster params body params backup
func (o *CreatePSMDBClusterParamsBodyParamsBackup) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this create PSMDB cluster params body params backup based on context it is used
func (o *CreatePSMDBClusterParamsBodyParamsBackup) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyParamsBackup) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyParamsBackup) UnmarshalBinary(b []byte) error {
	var res CreatePSMDBClusterParamsBodyParamsBackup
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
CreatePSMDBClusterParamsBodyParamsReplicaset ReplicaSet container parameters.
// TODO Do not use inner messages in all public APIs (for consistency).
swagger:model CreatePSMDBClusterParamsBodyParamsReplicaset
*/
type CreatePSMDBClusterParamsBodyParamsReplicaset struct {
	// Disk size in bytes.
	DiskSize string `json:"disk_size,omitempty"`

	// Configuration for PSMDB cluster
	Configuration string `json:"configuration,omitempty"`

	// Storage Class for PSMDB cluster.
	StorageClass string `json:"storage_class,omitempty"`

	// compute resources
	ComputeResources *CreatePSMDBClusterParamsBodyParamsReplicasetComputeResources `json:"compute_resources,omitempty"`
}

// Validate validates this create PSMDB cluster params body params replicaset
func (o *CreatePSMDBClusterParamsBodyParamsReplicaset) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateComputeResources(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *CreatePSMDBClusterParamsBodyParamsReplicaset) validateComputeResources(formats strfmt.Registry) error {
	if swag.IsZero(o.ComputeResources) { // not required
		return nil
	}

	if o.ComputeResources != nil {
		if err := o.ComputeResources.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "params" + "." + "replicaset" + "." + "compute_resources")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "params" + "." + "replicaset" + "." + "compute_resources")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this create PSMDB cluster params body params replicaset based on the context it is used
func (o *CreatePSMDBClusterParamsBodyParamsReplicaset) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateComputeResources(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *CreatePSMDBClusterParamsBodyParamsReplicaset) contextValidateComputeResources(ctx context.Context, formats strfmt.Registry) error {
	if o.ComputeResources != nil {
		if err := o.ComputeResources.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("body" + "." + "params" + "." + "replicaset" + "." + "compute_resources")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("body" + "." + "params" + "." + "replicaset" + "." + "compute_resources")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyParamsReplicaset) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyParamsReplicaset) UnmarshalBinary(b []byte) error {
	var res CreatePSMDBClusterParamsBodyParamsReplicaset
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
CreatePSMDBClusterParamsBodyParamsReplicasetComputeResources ComputeResources represents container computer resources requests or limits.
swagger:model CreatePSMDBClusterParamsBodyParamsReplicasetComputeResources
*/
type CreatePSMDBClusterParamsBodyParamsReplicasetComputeResources struct {
	// CPUs in milliCPUs; 1000m = 1 vCPU.
	CPUm int32 `json:"cpu_m,omitempty"`

	// Memory in bytes.
	MemoryBytes string `json:"memory_bytes,omitempty"`
}

// Validate validates this create PSMDB cluster params body params replicaset compute resources
func (o *CreatePSMDBClusterParamsBodyParamsReplicasetComputeResources) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this create PSMDB cluster params body params replicaset compute resources based on context it is used
func (o *CreatePSMDBClusterParamsBodyParamsReplicasetComputeResources) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyParamsReplicasetComputeResources) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyParamsReplicasetComputeResources) UnmarshalBinary(b []byte) error {
	var res CreatePSMDBClusterParamsBodyParamsReplicasetComputeResources
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
CreatePSMDBClusterParamsBodyParamsRestore Restore represents restoration payload to restore a database cluster from backup
swagger:model CreatePSMDBClusterParamsBodyParamsRestore
*/
type CreatePSMDBClusterParamsBodyParamsRestore struct {
	// Backup location in PMM.
	LocationID string `json:"location_id,omitempty"`

	// Destination filename.
	Destination string `json:"destination,omitempty"`

	// K8s Secrets name.
	SecretsName string `json:"secrets_name,omitempty"`
}

// Validate validates this create PSMDB cluster params body params restore
func (o *CreatePSMDBClusterParamsBodyParamsRestore) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this create PSMDB cluster params body params restore based on context it is used
func (o *CreatePSMDBClusterParamsBodyParamsRestore) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyParamsRestore) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyParamsRestore) UnmarshalBinary(b []byte) error {
	var res CreatePSMDBClusterParamsBodyParamsRestore
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
CreatePSMDBClusterParamsBodyTemplate create PSMDB cluster params body template
swagger:model CreatePSMDBClusterParamsBodyTemplate
*/
type CreatePSMDBClusterParamsBodyTemplate struct {
	// Template CR name.
	Name string `json:"name,omitempty"`

	// Template CR kind.
	Kind string `json:"kind,omitempty"`
}

// Validate validates this create PSMDB cluster params body template
func (o *CreatePSMDBClusterParamsBodyTemplate) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this create PSMDB cluster params body template based on context it is used
func (o *CreatePSMDBClusterParamsBodyTemplate) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyTemplate) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreatePSMDBClusterParamsBodyTemplate) UnmarshalBinary(b []byte) error {
	var res CreatePSMDBClusterParamsBodyTemplate
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}