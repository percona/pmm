// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: backup/v1/artifacts.proto

package backupv1

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on Artifact with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Artifact) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Artifact with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in ArtifactMultiError, or nil
// if none found.
func (m *Artifact) ValidateAll() error {
	return m.validate(true)
}

func (m *Artifact) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for ArtifactId

	// no validation rules for Name

	// no validation rules for Vendor

	// no validation rules for LocationId

	// no validation rules for LocationName

	// no validation rules for ServiceId

	// no validation rules for ServiceName

	// no validation rules for DataModel

	// no validation rules for Status

	if all {
		switch v := interface{}(m.GetCreatedAt()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, ArtifactValidationError{
					field:  "CreatedAt",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, ArtifactValidationError{
					field:  "CreatedAt",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetCreatedAt()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ArtifactValidationError{
				field:  "CreatedAt",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for Mode

	// no validation rules for IsShardedCluster

	// no validation rules for Folder

	for idx, item := range m.GetMetadataList() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, ArtifactValidationError{
						field:  fmt.Sprintf("MetadataList[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, ArtifactValidationError{
						field:  fmt.Sprintf("MetadataList[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return ArtifactValidationError{
					field:  fmt.Sprintf("MetadataList[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return ArtifactMultiError(errors)
	}

	return nil
}

// ArtifactMultiError is an error wrapping multiple validation errors returned
// by Artifact.ValidateAll() if the designated constraints aren't met.
type ArtifactMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ArtifactMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ArtifactMultiError) AllErrors() []error { return m }

// ArtifactValidationError is the validation error returned by
// Artifact.Validate if the designated constraints aren't met.
type ArtifactValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ArtifactValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ArtifactValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ArtifactValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ArtifactValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ArtifactValidationError) ErrorName() string { return "ArtifactValidationError" }

// Error satisfies the builtin error interface
func (e ArtifactValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sArtifact.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ArtifactValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ArtifactValidationError{}

// Validate checks the field values on ListArtifactsRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *ListArtifactsRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ListArtifactsRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ListArtifactsRequestMultiError, or nil if none found.
func (m *ListArtifactsRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *ListArtifactsRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return ListArtifactsRequestMultiError(errors)
	}

	return nil
}

// ListArtifactsRequestMultiError is an error wrapping multiple validation
// errors returned by ListArtifactsRequest.ValidateAll() if the designated
// constraints aren't met.
type ListArtifactsRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ListArtifactsRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ListArtifactsRequestMultiError) AllErrors() []error { return m }

// ListArtifactsRequestValidationError is the validation error returned by
// ListArtifactsRequest.Validate if the designated constraints aren't met.
type ListArtifactsRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ListArtifactsRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ListArtifactsRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ListArtifactsRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ListArtifactsRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ListArtifactsRequestValidationError) ErrorName() string {
	return "ListArtifactsRequestValidationError"
}

// Error satisfies the builtin error interface
func (e ListArtifactsRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sListArtifactsRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ListArtifactsRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ListArtifactsRequestValidationError{}

// Validate checks the field values on ListArtifactsResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *ListArtifactsResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ListArtifactsResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ListArtifactsResponseMultiError, or nil if none found.
func (m *ListArtifactsResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *ListArtifactsResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	for idx, item := range m.GetArtifacts() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, ListArtifactsResponseValidationError{
						field:  fmt.Sprintf("Artifacts[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, ListArtifactsResponseValidationError{
						field:  fmt.Sprintf("Artifacts[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return ListArtifactsResponseValidationError{
					field:  fmt.Sprintf("Artifacts[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return ListArtifactsResponseMultiError(errors)
	}

	return nil
}

// ListArtifactsResponseMultiError is an error wrapping multiple validation
// errors returned by ListArtifactsResponse.ValidateAll() if the designated
// constraints aren't met.
type ListArtifactsResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ListArtifactsResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ListArtifactsResponseMultiError) AllErrors() []error { return m }

// ListArtifactsResponseValidationError is the validation error returned by
// ListArtifactsResponse.Validate if the designated constraints aren't met.
type ListArtifactsResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ListArtifactsResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ListArtifactsResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ListArtifactsResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ListArtifactsResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ListArtifactsResponseValidationError) ErrorName() string {
	return "ListArtifactsResponseValidationError"
}

// Error satisfies the builtin error interface
func (e ListArtifactsResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sListArtifactsResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ListArtifactsResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ListArtifactsResponseValidationError{}

// Validate checks the field values on DeleteArtifactRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *DeleteArtifactRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on DeleteArtifactRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// DeleteArtifactRequestMultiError, or nil if none found.
func (m *DeleteArtifactRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *DeleteArtifactRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for ArtifactId

	// no validation rules for RemoveFiles

	if len(errors) > 0 {
		return DeleteArtifactRequestMultiError(errors)
	}

	return nil
}

// DeleteArtifactRequestMultiError is an error wrapping multiple validation
// errors returned by DeleteArtifactRequest.ValidateAll() if the designated
// constraints aren't met.
type DeleteArtifactRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m DeleteArtifactRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m DeleteArtifactRequestMultiError) AllErrors() []error { return m }

// DeleteArtifactRequestValidationError is the validation error returned by
// DeleteArtifactRequest.Validate if the designated constraints aren't met.
type DeleteArtifactRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DeleteArtifactRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DeleteArtifactRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DeleteArtifactRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DeleteArtifactRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DeleteArtifactRequestValidationError) ErrorName() string {
	return "DeleteArtifactRequestValidationError"
}

// Error satisfies the builtin error interface
func (e DeleteArtifactRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDeleteArtifactRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DeleteArtifactRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DeleteArtifactRequestValidationError{}

// Validate checks the field values on DeleteArtifactResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *DeleteArtifactResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on DeleteArtifactResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// DeleteArtifactResponseMultiError, or nil if none found.
func (m *DeleteArtifactResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *DeleteArtifactResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return DeleteArtifactResponseMultiError(errors)
	}

	return nil
}

// DeleteArtifactResponseMultiError is an error wrapping multiple validation
// errors returned by DeleteArtifactResponse.ValidateAll() if the designated
// constraints aren't met.
type DeleteArtifactResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m DeleteArtifactResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m DeleteArtifactResponseMultiError) AllErrors() []error { return m }

// DeleteArtifactResponseValidationError is the validation error returned by
// DeleteArtifactResponse.Validate if the designated constraints aren't met.
type DeleteArtifactResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DeleteArtifactResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DeleteArtifactResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DeleteArtifactResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DeleteArtifactResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DeleteArtifactResponseValidationError) ErrorName() string {
	return "DeleteArtifactResponseValidationError"
}

// Error satisfies the builtin error interface
func (e DeleteArtifactResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDeleteArtifactResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DeleteArtifactResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DeleteArtifactResponseValidationError{}

// Validate checks the field values on PitrTimerange with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *PitrTimerange) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on PitrTimerange with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in PitrTimerangeMultiError, or
// nil if none found.
func (m *PitrTimerange) ValidateAll() error {
	return m.validate(true)
}

func (m *PitrTimerange) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetStartTimestamp()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, PitrTimerangeValidationError{
					field:  "StartTimestamp",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, PitrTimerangeValidationError{
					field:  "StartTimestamp",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetStartTimestamp()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return PitrTimerangeValidationError{
				field:  "StartTimestamp",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetEndTimestamp()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, PitrTimerangeValidationError{
					field:  "EndTimestamp",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, PitrTimerangeValidationError{
					field:  "EndTimestamp",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetEndTimestamp()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return PitrTimerangeValidationError{
				field:  "EndTimestamp",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return PitrTimerangeMultiError(errors)
	}

	return nil
}

// PitrTimerangeMultiError is an error wrapping multiple validation errors
// returned by PitrTimerange.ValidateAll() if the designated constraints
// aren't met.
type PitrTimerangeMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m PitrTimerangeMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m PitrTimerangeMultiError) AllErrors() []error { return m }

// PitrTimerangeValidationError is the validation error returned by
// PitrTimerange.Validate if the designated constraints aren't met.
type PitrTimerangeValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e PitrTimerangeValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e PitrTimerangeValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e PitrTimerangeValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e PitrTimerangeValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e PitrTimerangeValidationError) ErrorName() string { return "PitrTimerangeValidationError" }

// Error satisfies the builtin error interface
func (e PitrTimerangeValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sPitrTimerange.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = PitrTimerangeValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = PitrTimerangeValidationError{}

// Validate checks the field values on ListPitrTimerangesRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *ListPitrTimerangesRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ListPitrTimerangesRequest with the
// rules defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ListPitrTimerangesRequestMultiError, or nil if none found.
func (m *ListPitrTimerangesRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *ListPitrTimerangesRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for ArtifactId

	if len(errors) > 0 {
		return ListPitrTimerangesRequestMultiError(errors)
	}

	return nil
}

// ListPitrTimerangesRequestMultiError is an error wrapping multiple validation
// errors returned by ListPitrTimerangesRequest.ValidateAll() if the
// designated constraints aren't met.
type ListPitrTimerangesRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ListPitrTimerangesRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ListPitrTimerangesRequestMultiError) AllErrors() []error { return m }

// ListPitrTimerangesRequestValidationError is the validation error returned by
// ListPitrTimerangesRequest.Validate if the designated constraints aren't met.
type ListPitrTimerangesRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ListPitrTimerangesRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ListPitrTimerangesRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ListPitrTimerangesRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ListPitrTimerangesRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ListPitrTimerangesRequestValidationError) ErrorName() string {
	return "ListPitrTimerangesRequestValidationError"
}

// Error satisfies the builtin error interface
func (e ListPitrTimerangesRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sListPitrTimerangesRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ListPitrTimerangesRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ListPitrTimerangesRequestValidationError{}

// Validate checks the field values on ListPitrTimerangesResponse with the
// rules defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *ListPitrTimerangesResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ListPitrTimerangesResponse with the
// rules defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ListPitrTimerangesResponseMultiError, or nil if none found.
func (m *ListPitrTimerangesResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *ListPitrTimerangesResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	for idx, item := range m.GetTimeranges() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, ListPitrTimerangesResponseValidationError{
						field:  fmt.Sprintf("Timeranges[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, ListPitrTimerangesResponseValidationError{
						field:  fmt.Sprintf("Timeranges[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return ListPitrTimerangesResponseValidationError{
					field:  fmt.Sprintf("Timeranges[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return ListPitrTimerangesResponseMultiError(errors)
	}

	return nil
}

// ListPitrTimerangesResponseMultiError is an error wrapping multiple
// validation errors returned by ListPitrTimerangesResponse.ValidateAll() if
// the designated constraints aren't met.
type ListPitrTimerangesResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ListPitrTimerangesResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ListPitrTimerangesResponseMultiError) AllErrors() []error { return m }

// ListPitrTimerangesResponseValidationError is the validation error returned
// by ListPitrTimerangesResponse.Validate if the designated constraints aren't met.
type ListPitrTimerangesResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ListPitrTimerangesResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ListPitrTimerangesResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ListPitrTimerangesResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ListPitrTimerangesResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ListPitrTimerangesResponseValidationError) ErrorName() string {
	return "ListPitrTimerangesResponseValidationError"
}

// Error satisfies the builtin error interface
func (e ListPitrTimerangesResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sListPitrTimerangesResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ListPitrTimerangesResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ListPitrTimerangesResponseValidationError{}