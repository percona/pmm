// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: managementpb/dbaas/logs.proto

package dbaasv1beta1

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

// Validate checks the field values on Logs with the rules defined in the proto
// definition for this message. If any rules are violated, the first error
// encountered is returned, or nil if there are no violations.
func (m *Logs) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Logs with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in LogsMultiError, or nil if none found.
func (m *Logs) ValidateAll() error {
	return m.validate(true)
}

func (m *Logs) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Pod

	// no validation rules for Container

	if len(errors) > 0 {
		return LogsMultiError(errors)
	}

	return nil
}

// LogsMultiError is an error wrapping multiple validation errors returned by
// Logs.ValidateAll() if the designated constraints aren't met.
type LogsMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m LogsMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m LogsMultiError) AllErrors() []error { return m }

// LogsValidationError is the validation error returned by Logs.Validate if the
// designated constraints aren't met.
type LogsValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e LogsValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e LogsValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e LogsValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e LogsValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e LogsValidationError) ErrorName() string { return "LogsValidationError" }

// Error satisfies the builtin error interface
func (e LogsValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sLogs.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = LogsValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = LogsValidationError{}

// Validate checks the field values on GetLogsRequest with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *GetLogsRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on GetLogsRequest with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in GetLogsRequestMultiError,
// or nil if none found.
func (m *GetLogsRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *GetLogsRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if utf8.RuneCountInString(m.GetKubernetesClusterName()) < 1 {
		err := GetLogsRequestValidationError{
			field:  "KubernetesClusterName",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if utf8.RuneCountInString(m.GetClusterName()) < 1 {
		err := GetLogsRequestValidationError{
			field:  "ClusterName",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return GetLogsRequestMultiError(errors)
	}

	return nil
}

// GetLogsRequestMultiError is an error wrapping multiple validation errors
// returned by GetLogsRequest.ValidateAll() if the designated constraints
// aren't met.
type GetLogsRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m GetLogsRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m GetLogsRequestMultiError) AllErrors() []error { return m }

// GetLogsRequestValidationError is the validation error returned by
// GetLogsRequest.Validate if the designated constraints aren't met.
type GetLogsRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetLogsRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetLogsRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetLogsRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetLogsRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetLogsRequestValidationError) ErrorName() string { return "GetLogsRequestValidationError" }

// Error satisfies the builtin error interface
func (e GetLogsRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetLogsRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetLogsRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetLogsRequestValidationError{}

// Validate checks the field values on GetLogsResponse with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *GetLogsResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on GetLogsResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// GetLogsResponseMultiError, or nil if none found.
func (m *GetLogsResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *GetLogsResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	for idx, item := range m.GetLogs() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, GetLogsResponseValidationError{
						field:  fmt.Sprintf("Logs[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, GetLogsResponseValidationError{
						field:  fmt.Sprintf("Logs[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return GetLogsResponseValidationError{
					field:  fmt.Sprintf("Logs[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return GetLogsResponseMultiError(errors)
	}

	return nil
}

// GetLogsResponseMultiError is an error wrapping multiple validation errors
// returned by GetLogsResponse.ValidateAll() if the designated constraints
// aren't met.
type GetLogsResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m GetLogsResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m GetLogsResponseMultiError) AllErrors() []error { return m }

// GetLogsResponseValidationError is the validation error returned by
// GetLogsResponse.Validate if the designated constraints aren't met.
type GetLogsResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetLogsResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetLogsResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetLogsResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetLogsResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetLogsResponseValidationError) ErrorName() string { return "GetLogsResponseValidationError" }

// Error satisfies the builtin error interface
func (e GetLogsResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetLogsResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetLogsResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetLogsResponseValidationError{}