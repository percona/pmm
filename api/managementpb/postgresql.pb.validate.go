// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: managementpb/postgresql.proto

package managementpb

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

	inventorypb "github.com/percona/pmm/api/inventorypb"
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

	_ = inventorypb.LogLevel(0)
)

// Validate checks the field values on AddPostgreSQLRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *AddPostgreSQLRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on AddPostgreSQLRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// AddPostgreSQLRequestMultiError, or nil if none found.
func (m *AddPostgreSQLRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *AddPostgreSQLRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for NodeId

	// no validation rules for NodeName

	if all {
		switch v := interface{}(m.GetAddNode()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, AddPostgreSQLRequestValidationError{
					field:  "AddNode",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, AddPostgreSQLRequestValidationError{
					field:  "AddNode",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetAddNode()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return AddPostgreSQLRequestValidationError{
				field:  "AddNode",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if utf8.RuneCountInString(m.GetServiceName()) < 1 {
		err := AddPostgreSQLRequestValidationError{
			field:  "ServiceName",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	// no validation rules for Address

	// no validation rules for Port

	// no validation rules for Database

	// no validation rules for Socket

	if utf8.RuneCountInString(m.GetPmmAgentId()) < 1 {
		err := AddPostgreSQLRequestValidationError{
			field:  "PmmAgentId",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	// no validation rules for Environment

	// no validation rules for Cluster

	// no validation rules for ReplicationSet

	if utf8.RuneCountInString(m.GetUsername()) < 1 {
		err := AddPostgreSQLRequestValidationError{
			field:  "Username",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	// no validation rules for Password

	// no validation rules for QanPostgresqlPgstatementsAgent

	// no validation rules for QanPostgresqlPgstatmonitorAgent

	// no validation rules for MaxQueryLength

	// no validation rules for DisableQueryExamples

	// no validation rules for CustomLabels

	// no validation rules for SkipConnectionCheck

	// no validation rules for DisableCommentsParsing

	// no validation rules for Tls

	// no validation rules for TlsSkipVerify

	// no validation rules for MetricsMode

	// no validation rules for TlsCa

	// no validation rules for TlsCert

	// no validation rules for TlsKey

	// no validation rules for AgentPassword

	// no validation rules for LogLevel

	// no validation rules for AutoDiscoveryLimit

	// no validation rules for ExposeExporter

	if len(errors) > 0 {
		return AddPostgreSQLRequestMultiError(errors)
	}

	return nil
}

// AddPostgreSQLRequestMultiError is an error wrapping multiple validation
// errors returned by AddPostgreSQLRequest.ValidateAll() if the designated
// constraints aren't met.
type AddPostgreSQLRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m AddPostgreSQLRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m AddPostgreSQLRequestMultiError) AllErrors() []error { return m }

// AddPostgreSQLRequestValidationError is the validation error returned by
// AddPostgreSQLRequest.Validate if the designated constraints aren't met.
type AddPostgreSQLRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e AddPostgreSQLRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e AddPostgreSQLRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e AddPostgreSQLRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e AddPostgreSQLRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e AddPostgreSQLRequestValidationError) ErrorName() string {
	return "AddPostgreSQLRequestValidationError"
}

// Error satisfies the builtin error interface
func (e AddPostgreSQLRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sAddPostgreSQLRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = AddPostgreSQLRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = AddPostgreSQLRequestValidationError{}

// Validate checks the field values on AddPostgreSQLResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *AddPostgreSQLResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on AddPostgreSQLResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// AddPostgreSQLResponseMultiError, or nil if none found.
func (m *AddPostgreSQLResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *AddPostgreSQLResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetService()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, AddPostgreSQLResponseValidationError{
					field:  "Service",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, AddPostgreSQLResponseValidationError{
					field:  "Service",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetService()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return AddPostgreSQLResponseValidationError{
				field:  "Service",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetPostgresExporter()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, AddPostgreSQLResponseValidationError{
					field:  "PostgresExporter",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, AddPostgreSQLResponseValidationError{
					field:  "PostgresExporter",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetPostgresExporter()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return AddPostgreSQLResponseValidationError{
				field:  "PostgresExporter",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetQanPostgresqlPgstatementsAgent()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, AddPostgreSQLResponseValidationError{
					field:  "QanPostgresqlPgstatementsAgent",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, AddPostgreSQLResponseValidationError{
					field:  "QanPostgresqlPgstatementsAgent",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetQanPostgresqlPgstatementsAgent()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return AddPostgreSQLResponseValidationError{
				field:  "QanPostgresqlPgstatementsAgent",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetQanPostgresqlPgstatmonitorAgent()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, AddPostgreSQLResponseValidationError{
					field:  "QanPostgresqlPgstatmonitorAgent",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, AddPostgreSQLResponseValidationError{
					field:  "QanPostgresqlPgstatmonitorAgent",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetQanPostgresqlPgstatmonitorAgent()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return AddPostgreSQLResponseValidationError{
				field:  "QanPostgresqlPgstatmonitorAgent",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return AddPostgreSQLResponseMultiError(errors)
	}

	return nil
}

// AddPostgreSQLResponseMultiError is an error wrapping multiple validation
// errors returned by AddPostgreSQLResponse.ValidateAll() if the designated
// constraints aren't met.
type AddPostgreSQLResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m AddPostgreSQLResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m AddPostgreSQLResponseMultiError) AllErrors() []error { return m }

// AddPostgreSQLResponseValidationError is the validation error returned by
// AddPostgreSQLResponse.Validate if the designated constraints aren't met.
type AddPostgreSQLResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e AddPostgreSQLResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e AddPostgreSQLResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e AddPostgreSQLResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e AddPostgreSQLResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e AddPostgreSQLResponseValidationError) ErrorName() string {
	return "AddPostgreSQLResponseValidationError"
}

// Error satisfies the builtin error interface
func (e AddPostgreSQLResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sAddPostgreSQLResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = AddPostgreSQLResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = AddPostgreSQLResponseValidationError{}