// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: managementpb/agent/agent.proto

package agentv1beta1

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

// Validate checks the field values on UniversalAgent with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *UniversalAgent) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on UniversalAgent with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in UniversalAgentMultiError,
// or nil if none found.
func (m *UniversalAgent) ValidateAll() error {
	return m.validate(true)
}

func (m *UniversalAgent) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for AgentId

	// no validation rules for IsAgentPasswordSet

	// no validation rules for AgentType

	// no validation rules for AwsAccessKey

	// no validation rules for IsAwsSecretKeySet

	if all {
		switch v := interface{}(m.GetAzureOptions()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "AzureOptions",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "AzureOptions",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetAzureOptions()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return UniversalAgentValidationError{
				field:  "AzureOptions",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetCreatedAt()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "CreatedAt",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "CreatedAt",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetCreatedAt()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return UniversalAgentValidationError{
				field:  "CreatedAt",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for CustomLabels

	// no validation rules for Disabled

	// no validation rules for ListenPort

	// no validation rules for LogLevel

	// no validation rules for MaxQueryLength

	// no validation rules for MaxQueryLogSize

	// no validation rules for MetricsPath

	// no validation rules for MetricsScheme

	if all {
		switch v := interface{}(m.GetMongoDbOptions()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "MongoDbOptions",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "MongoDbOptions",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMongoDbOptions()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return UniversalAgentValidationError{
				field:  "MongoDbOptions",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetMysqlOptions()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "MysqlOptions",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "MysqlOptions",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMysqlOptions()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return UniversalAgentValidationError{
				field:  "MysqlOptions",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for NodeId

	// no validation rules for IsPasswordSet

	// no validation rules for PmmAgentId

	if all {
		switch v := interface{}(m.GetPostgresqlOptions()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "PostgresqlOptions",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "PostgresqlOptions",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetPostgresqlOptions()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return UniversalAgentValidationError{
				field:  "PostgresqlOptions",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for ProcessExecPath

	// no validation rules for PushMetrics

	// no validation rules for QueryExamplesDisabled

	// no validation rules for CommentsParsingDisabled

	// no validation rules for RdsBasicMetricsDisabled

	// no validation rules for RdsEnhancedMetricsDisabled

	// no validation rules for RunsOnNodeId

	// no validation rules for ServiceId

	// no validation rules for Status

	// no validation rules for TableCount

	// no validation rules for TableCountTablestatsGroupLimit

	// no validation rules for Tls

	// no validation rules for TlsSkipVerify

	// no validation rules for Username

	if all {
		switch v := interface{}(m.GetUpdatedAt()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "UpdatedAt",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, UniversalAgentValidationError{
					field:  "UpdatedAt",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetUpdatedAt()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return UniversalAgentValidationError{
				field:  "UpdatedAt",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for Version

	// no validation rules for IsConnected

	// no validation rules for ExposeExporter

	if len(errors) > 0 {
		return UniversalAgentMultiError(errors)
	}

	return nil
}

// UniversalAgentMultiError is an error wrapping multiple validation errors
// returned by UniversalAgent.ValidateAll() if the designated constraints
// aren't met.
type UniversalAgentMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m UniversalAgentMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m UniversalAgentMultiError) AllErrors() []error { return m }

// UniversalAgentValidationError is the validation error returned by
// UniversalAgent.Validate if the designated constraints aren't met.
type UniversalAgentValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e UniversalAgentValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e UniversalAgentValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e UniversalAgentValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e UniversalAgentValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e UniversalAgentValidationError) ErrorName() string { return "UniversalAgentValidationError" }

// Error satisfies the builtin error interface
func (e UniversalAgentValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sUniversalAgent.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = UniversalAgentValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = UniversalAgentValidationError{}

// Validate checks the field values on ListAgentRequest with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *ListAgentRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ListAgentRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ListAgentRequestMultiError, or nil if none found.
func (m *ListAgentRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *ListAgentRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for ServiceId

	// no validation rules for NodeId

	if len(errors) > 0 {
		return ListAgentRequestMultiError(errors)
	}

	return nil
}

// ListAgentRequestMultiError is an error wrapping multiple validation errors
// returned by ListAgentRequest.ValidateAll() if the designated constraints
// aren't met.
type ListAgentRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ListAgentRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ListAgentRequestMultiError) AllErrors() []error { return m }

// ListAgentRequestValidationError is the validation error returned by
// ListAgentRequest.Validate if the designated constraints aren't met.
type ListAgentRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ListAgentRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ListAgentRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ListAgentRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ListAgentRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ListAgentRequestValidationError) ErrorName() string { return "ListAgentRequestValidationError" }

// Error satisfies the builtin error interface
func (e ListAgentRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sListAgentRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ListAgentRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ListAgentRequestValidationError{}

// Validate checks the field values on ListAgentResponse with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *ListAgentResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ListAgentResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ListAgentResponseMultiError, or nil if none found.
func (m *ListAgentResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *ListAgentResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	for idx, item := range m.GetAgents() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, ListAgentResponseValidationError{
						field:  fmt.Sprintf("Agents[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, ListAgentResponseValidationError{
						field:  fmt.Sprintf("Agents[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return ListAgentResponseValidationError{
					field:  fmt.Sprintf("Agents[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return ListAgentResponseMultiError(errors)
	}

	return nil
}

// ListAgentResponseMultiError is an error wrapping multiple validation errors
// returned by ListAgentResponse.ValidateAll() if the designated constraints
// aren't met.
type ListAgentResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ListAgentResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ListAgentResponseMultiError) AllErrors() []error { return m }

// ListAgentResponseValidationError is the validation error returned by
// ListAgentResponse.Validate if the designated constraints aren't met.
type ListAgentResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ListAgentResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ListAgentResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ListAgentResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ListAgentResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ListAgentResponseValidationError) ErrorName() string {
	return "ListAgentResponseValidationError"
}

// Error satisfies the builtin error interface
func (e ListAgentResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sListAgentResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ListAgentResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ListAgentResponseValidationError{}

// Validate checks the field values on UniversalAgent_MySQLOptions with the
// rules defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *UniversalAgent_MySQLOptions) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on UniversalAgent_MySQLOptions with the
// rules defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// UniversalAgent_MySQLOptionsMultiError, or nil if none found.
func (m *UniversalAgent_MySQLOptions) ValidateAll() error {
	return m.validate(true)
}

func (m *UniversalAgent_MySQLOptions) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for IsTlsKeySet

	if len(errors) > 0 {
		return UniversalAgent_MySQLOptionsMultiError(errors)
	}

	return nil
}

// UniversalAgent_MySQLOptionsMultiError is an error wrapping multiple
// validation errors returned by UniversalAgent_MySQLOptions.ValidateAll() if
// the designated constraints aren't met.
type UniversalAgent_MySQLOptionsMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m UniversalAgent_MySQLOptionsMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m UniversalAgent_MySQLOptionsMultiError) AllErrors() []error { return m }

// UniversalAgent_MySQLOptionsValidationError is the validation error returned
// by UniversalAgent_MySQLOptions.Validate if the designated constraints
// aren't met.
type UniversalAgent_MySQLOptionsValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e UniversalAgent_MySQLOptionsValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e UniversalAgent_MySQLOptionsValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e UniversalAgent_MySQLOptionsValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e UniversalAgent_MySQLOptionsValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e UniversalAgent_MySQLOptionsValidationError) ErrorName() string {
	return "UniversalAgent_MySQLOptionsValidationError"
}

// Error satisfies the builtin error interface
func (e UniversalAgent_MySQLOptionsValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sUniversalAgent_MySQLOptions.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = UniversalAgent_MySQLOptionsValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = UniversalAgent_MySQLOptionsValidationError{}

// Validate checks the field values on UniversalAgent_AzureOptions with the
// rules defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *UniversalAgent_AzureOptions) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on UniversalAgent_AzureOptions with the
// rules defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// UniversalAgent_AzureOptionsMultiError, or nil if none found.
func (m *UniversalAgent_AzureOptions) ValidateAll() error {
	return m.validate(true)
}

func (m *UniversalAgent_AzureOptions) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for ClientId

	// no validation rules for IsClientSecretSet

	// no validation rules for ResourceGroup

	// no validation rules for SubscriptionId

	// no validation rules for TenantId

	if len(errors) > 0 {
		return UniversalAgent_AzureOptionsMultiError(errors)
	}

	return nil
}

// UniversalAgent_AzureOptionsMultiError is an error wrapping multiple
// validation errors returned by UniversalAgent_AzureOptions.ValidateAll() if
// the designated constraints aren't met.
type UniversalAgent_AzureOptionsMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m UniversalAgent_AzureOptionsMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m UniversalAgent_AzureOptionsMultiError) AllErrors() []error { return m }

// UniversalAgent_AzureOptionsValidationError is the validation error returned
// by UniversalAgent_AzureOptions.Validate if the designated constraints
// aren't met.
type UniversalAgent_AzureOptionsValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e UniversalAgent_AzureOptionsValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e UniversalAgent_AzureOptionsValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e UniversalAgent_AzureOptionsValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e UniversalAgent_AzureOptionsValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e UniversalAgent_AzureOptionsValidationError) ErrorName() string {
	return "UniversalAgent_AzureOptionsValidationError"
}

// Error satisfies the builtin error interface
func (e UniversalAgent_AzureOptionsValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sUniversalAgent_AzureOptions.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = UniversalAgent_AzureOptionsValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = UniversalAgent_AzureOptionsValidationError{}

// Validate checks the field values on UniversalAgent_MongoDBOptions with the
// rules defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *UniversalAgent_MongoDBOptions) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on UniversalAgent_MongoDBOptions with
// the rules defined in the proto definition for this message. If any rules
// are violated, the result is a list of violation errors wrapped in
// UniversalAgent_MongoDBOptionsMultiError, or nil if none found.
func (m *UniversalAgent_MongoDBOptions) ValidateAll() error {
	return m.validate(true)
}

func (m *UniversalAgent_MongoDBOptions) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for IsTlsCertificateKeySet

	// no validation rules for IsTlsCertificateKeyFilePasswordSet

	// no validation rules for AuthenticationMechanism

	// no validation rules for AuthenticationDatabase

	// no validation rules for CollectionsLimit

	// no validation rules for EnableAllCollectors

	if len(errors) > 0 {
		return UniversalAgent_MongoDBOptionsMultiError(errors)
	}

	return nil
}

// UniversalAgent_MongoDBOptionsMultiError is an error wrapping multiple
// validation errors returned by UniversalAgent_MongoDBOptions.ValidateAll()
// if the designated constraints aren't met.
type UniversalAgent_MongoDBOptionsMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m UniversalAgent_MongoDBOptionsMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m UniversalAgent_MongoDBOptionsMultiError) AllErrors() []error { return m }

// UniversalAgent_MongoDBOptionsValidationError is the validation error
// returned by UniversalAgent_MongoDBOptions.Validate if the designated
// constraints aren't met.
type UniversalAgent_MongoDBOptionsValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e UniversalAgent_MongoDBOptionsValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e UniversalAgent_MongoDBOptionsValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e UniversalAgent_MongoDBOptionsValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e UniversalAgent_MongoDBOptionsValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e UniversalAgent_MongoDBOptionsValidationError) ErrorName() string {
	return "UniversalAgent_MongoDBOptionsValidationError"
}

// Error satisfies the builtin error interface
func (e UniversalAgent_MongoDBOptionsValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sUniversalAgent_MongoDBOptions.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = UniversalAgent_MongoDBOptionsValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = UniversalAgent_MongoDBOptionsValidationError{}

// Validate checks the field values on UniversalAgent_PostgreSQLOptions with
// the rules defined in the proto definition for this message. If any rules
// are violated, the first error encountered is returned, or nil if there are
// no violations.
func (m *UniversalAgent_PostgreSQLOptions) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on UniversalAgent_PostgreSQLOptions with
// the rules defined in the proto definition for this message. If any rules
// are violated, the result is a list of violation errors wrapped in
// UniversalAgent_PostgreSQLOptionsMultiError, or nil if none found.
func (m *UniversalAgent_PostgreSQLOptions) ValidateAll() error {
	return m.validate(true)
}

func (m *UniversalAgent_PostgreSQLOptions) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for IsSslKeySet

	// no validation rules for AutoDiscoveryLimit

	// no validation rules for MaxExporterConnections

	if len(errors) > 0 {
		return UniversalAgent_PostgreSQLOptionsMultiError(errors)
	}

	return nil
}

// UniversalAgent_PostgreSQLOptionsMultiError is an error wrapping multiple
// validation errors returned by
// UniversalAgent_PostgreSQLOptions.ValidateAll() if the designated
// constraints aren't met.
type UniversalAgent_PostgreSQLOptionsMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m UniversalAgent_PostgreSQLOptionsMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m UniversalAgent_PostgreSQLOptionsMultiError) AllErrors() []error { return m }

// UniversalAgent_PostgreSQLOptionsValidationError is the validation error
// returned by UniversalAgent_PostgreSQLOptions.Validate if the designated
// constraints aren't met.
type UniversalAgent_PostgreSQLOptionsValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e UniversalAgent_PostgreSQLOptionsValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e UniversalAgent_PostgreSQLOptionsValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e UniversalAgent_PostgreSQLOptionsValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e UniversalAgent_PostgreSQLOptionsValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e UniversalAgent_PostgreSQLOptionsValidationError) ErrorName() string {
	return "UniversalAgent_PostgreSQLOptionsValidationError"
}

// Error satisfies the builtin error interface
func (e UniversalAgent_PostgreSQLOptionsValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sUniversalAgent_PostgreSQLOptions.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = UniversalAgent_PostgreSQLOptionsValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = UniversalAgent_PostgreSQLOptionsValidationError{}