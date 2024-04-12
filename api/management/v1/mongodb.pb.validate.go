// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: management/v1/mongodb.proto

package managementv1

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

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
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

	_ = inventoryv1.LogLevel(0)
)

// Validate checks the field values on AddMongoDBServiceParams with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *AddMongoDBServiceParams) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on AddMongoDBServiceParams with the
// rules defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// AddMongoDBServiceParamsMultiError, or nil if none found.
func (m *AddMongoDBServiceParams) ValidateAll() error {
	return m.validate(true)
}

func (m *AddMongoDBServiceParams) validate(all bool) error {
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
				errors = append(errors, AddMongoDBServiceParamsValidationError{
					field:  "AddNode",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, AddMongoDBServiceParamsValidationError{
					field:  "AddNode",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetAddNode()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return AddMongoDBServiceParamsValidationError{
				field:  "AddNode",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if utf8.RuneCountInString(m.GetServiceName()) < 1 {
		err := AddMongoDBServiceParamsValidationError{
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

	// no validation rules for Socket

	if utf8.RuneCountInString(m.GetPmmAgentId()) < 1 {
		err := AddMongoDBServiceParamsValidationError{
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

	// no validation rules for Username

	// no validation rules for Password

	// no validation rules for QanMongodbProfiler

	// no validation rules for CustomLabels

	// no validation rules for SkipConnectionCheck

	// no validation rules for Tls

	// no validation rules for TlsSkipVerify

	// no validation rules for TlsCertificateKey

	// no validation rules for TlsCertificateKeyFilePassword

	// no validation rules for TlsCa

	// no validation rules for MaxQueryLength

	// no validation rules for MetricsMode

	// no validation rules for AuthenticationMechanism

	// no validation rules for AuthenticationDatabase

	// no validation rules for AgentPassword

	// no validation rules for CollectionsLimit

	// no validation rules for EnableAllCollectors

	// no validation rules for LogLevel

	// no validation rules for ExposeExporter

	if len(errors) > 0 {
		return AddMongoDBServiceParamsMultiError(errors)
	}

	return nil
}

// AddMongoDBServiceParamsMultiError is an error wrapping multiple validation
// errors returned by AddMongoDBServiceParams.ValidateAll() if the designated
// constraints aren't met.
type AddMongoDBServiceParamsMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m AddMongoDBServiceParamsMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m AddMongoDBServiceParamsMultiError) AllErrors() []error { return m }

// AddMongoDBServiceParamsValidationError is the validation error returned by
// AddMongoDBServiceParams.Validate if the designated constraints aren't met.
type AddMongoDBServiceParamsValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e AddMongoDBServiceParamsValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e AddMongoDBServiceParamsValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e AddMongoDBServiceParamsValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e AddMongoDBServiceParamsValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e AddMongoDBServiceParamsValidationError) ErrorName() string {
	return "AddMongoDBServiceParamsValidationError"
}

// Error satisfies the builtin error interface
func (e AddMongoDBServiceParamsValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sAddMongoDBServiceParams.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = AddMongoDBServiceParamsValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = AddMongoDBServiceParamsValidationError{}

// Validate checks the field values on MongoDBServiceResult with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *MongoDBServiceResult) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on MongoDBServiceResult with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// MongoDBServiceResultMultiError, or nil if none found.
func (m *MongoDBServiceResult) ValidateAll() error {
	return m.validate(true)
}

func (m *MongoDBServiceResult) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetService()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, MongoDBServiceResultValidationError{
					field:  "Service",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, MongoDBServiceResultValidationError{
					field:  "Service",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetService()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return MongoDBServiceResultValidationError{
				field:  "Service",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetMongodbExporter()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, MongoDBServiceResultValidationError{
					field:  "MongodbExporter",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, MongoDBServiceResultValidationError{
					field:  "MongodbExporter",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMongodbExporter()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return MongoDBServiceResultValidationError{
				field:  "MongodbExporter",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetQanMongodbProfiler()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, MongoDBServiceResultValidationError{
					field:  "QanMongodbProfiler",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, MongoDBServiceResultValidationError{
					field:  "QanMongodbProfiler",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetQanMongodbProfiler()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return MongoDBServiceResultValidationError{
				field:  "QanMongodbProfiler",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return MongoDBServiceResultMultiError(errors)
	}

	return nil
}

// MongoDBServiceResultMultiError is an error wrapping multiple validation
// errors returned by MongoDBServiceResult.ValidateAll() if the designated
// constraints aren't met.
type MongoDBServiceResultMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m MongoDBServiceResultMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m MongoDBServiceResultMultiError) AllErrors() []error { return m }

// MongoDBServiceResultValidationError is the validation error returned by
// MongoDBServiceResult.Validate if the designated constraints aren't met.
type MongoDBServiceResultValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e MongoDBServiceResultValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e MongoDBServiceResultValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e MongoDBServiceResultValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e MongoDBServiceResultValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e MongoDBServiceResultValidationError) ErrorName() string {
	return "MongoDBServiceResultValidationError"
}

// Error satisfies the builtin error interface
func (e MongoDBServiceResultValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sMongoDBServiceResult.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = MongoDBServiceResultValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = MongoDBServiceResultValidationError{}