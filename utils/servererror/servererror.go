// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package servererror turns errors returned by PMM Server API calls into messages a CLI user
// can act on. It is shared by pmm-admin and pmm-agent: both talk to PMM Server over the same
// transport and both expose the same --server-insecure-tls flag, so both need the same hints.
package servererror

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-openapi/runtime"
)

// InsecureTLSFlag is the name of the flag which disables PMM Server TLS certificate
// validation; pmm-admin and pmm-agent spell it the same way.
const InsecureTLSFlag = "--server-insecure-tls"

// gRPC status codes carried in PMM Server response payloads. They are part of the wire format
// (https://grpc.io/docs/guides/status-codes/) and are spelled out here rather than taken from
// google.golang.org/grpc/codes so that pmm-admin does not have to link the gRPC packages.
const (
	codePermissionDenied = 7
	codeUnauthenticated  = 16
)

// IsTLSCertificateError reports whether err was caused by a failure to verify the TLS
// certificate presented by PMM Server. PMM Server is shipped with a self-signed certificate
// issued for localhost only, so this is the expected outcome of addressing it by any other
// host name.
func IsTLSCertificateError(err error) bool {
	if err == nil {
		return false
	}

	var (
		verificationErr *tls.CertificateVerificationError
		hostnameErr     x509.HostnameError
		authorityErr    x509.UnknownAuthorityError
		invalidErr      x509.CertificateInvalidError
	)

	// Since Go 1.20 crypto/tls wraps both chain and host name failures in
	// tls.CertificateVerificationError on every platform, including the macOS and Windows
	// system verifiers. The bare x509 errors are matched too, so that errors constructed or
	// re-wrapped by callers - and by this package's tests - are still recognised.
	return errors.As(err, &verificationErr) ||
		errors.As(err, &hostnameErr) ||
		errors.As(err, &authorityErr) ||
		errors.As(err, &invalidErr)
}

// WrapTLSError appends a hint about --server-insecure-tls to TLS certificate verification
// failures, naming host as the address the certificate was checked against. Other errors, and
// failures seen while certificate validation is already disabled, are returned unchanged.
func WrapTLSError(err error, host string, insecureTLS bool) error {
	if insecureTLS || !IsTLSCertificateError(err) {
		return err
	}

	reason := "PMM Server TLS certificate could not be verified: it is either self-signed or not valid for the requested host."
	if host != "" {
		reason = fmt.Sprintf(
			"PMM Server TLS certificate could not be verified: it is either self-signed or not valid for host %q.",
			host,
		)
	}

	return fmt.Errorf("%w.\n%s\nRe-run the command with %s to skip PMM Server TLS certificate validation, "+
		"or configure PMM Server with a certificate valid for that host", err, reason, InsecureTLSFlag)
}

// AuthHint returns a sentence, without trailing punctuation, explaining an authentication
// error reported by PMM Server. It returns an empty string when the response does not describe
// one. The httpCode argument is the HTTP status and grpcCode the gRPC code carried in the
// response payload, which is zero when PMM Server did not send one.
//
// Callers own the punctuation and the separator: pmm-admin appends the hint to a single-line
// message, pmm-agent puts it on a line of its own.
func AuthHint(httpCode int, grpcCode int32) string {
	switch {
	// PMM Server reports rejected credentials with the gRPC Unauthenticated code. It sends
	// no gRPC code only on paths which never reach the API, so a bare HTTP 401 means the
	// credentials were rejected as well.
	case grpcCode == codeUnauthenticated,
		httpCode == http.StatusUnauthorized && grpcCode == 0:
		return "Please check username and password"

	// The user authenticated but their role does not allow the request. nginx serves PMM
	// Server's PermissionDenied as HTTP 403 with a static body carrying the same gRPC code,
	// so this is not a credentials problem and must not be reported as one.
	case grpcCode == codePermissionDenied,
		httpCode == http.StatusForbidden && grpcCode == 0:
		return "Please check that your PMM user has sufficient permissions"

	// nginx auth_request accepts 401 and 403 only, so PMM Server maps every other
	// authentication error - internal ones included - onto one of those two statuses. Those
	// are not caused by wrong credentials and must not be reported as such, but they must
	// still be explained: leaving 403 out here dropped the hint the CLIs used to print.
	case httpCode == http.StatusUnauthorized, httpCode == http.StatusForbidden:
		return "Please check PMM Server logs"
	}

	return ""
}

// NginxHint explains a NginxError: nginx answered instead of PMM Server's API, most commonly
// because pmm-managed itself is down. nginx's auth_request also runs on this path, but PMM
// Server's shipped configuration always has it answer in JSON, so a rejected login reaches
// AuthHint instead - this hint is not about credentials. Like AuthHint it carries no trailing
// punctuation: callers own it.
const NginxHint = "PMM Server did not return an API response. Please check pmm-managed logs, " +
	"and that PMM Server is running"

// NginxError is the body of a response served by the nginx which fronts PMM Server rather than
// by the API itself. Its content type is HTML or plain text - never the JSON the generated
// clients expect - and a caller gets one both when pmm-managed is down and when a request is
// rejected before it reaches the API.
type NginxError string

// Error implements the error interface.
func (e NginxError) Error() string {
	return "response from nginx: " + string(e)
}

// GoString implements fmt.GoStringer, used by the %#v debug logging of both CLIs.
func (e NginxError) GoString() string {
	return fmt.Sprintf("NginxError(%q)", string(e))
}

// maxNginxBodySize bounds how much of a response NginxConsumer buffers. A real nginx or
// default error page is a few hundred bytes; the limit exists so that a misconfigured proxy or
// a hostile server answering with an unbounded body cannot be used to exhaust memory.
const maxNginxBodySize = 64 * 1024

// NginxConsumer returns a go-openapi consumer which turns a response body into a NginxError.
// Both CLIs install it for the content types the PMM Server API never answers with, so that an
// nginx page surfaces as an error of its own instead of as a JSON decoding failure.
//
// The body is trimmed: nginx pages end with a newline, which would otherwise leave a blank line
// between the page and whatever the caller prints after it. A body over maxNginxBodySize is
// truncated rather than rejected outright, since even a partial nginx page - unlike a read
// failure - is still evidence of what happened and worth reporting.
func NginxConsumer() runtime.ConsumerFunc {
	return func(reader io.Reader, _ any) error {
		b, err := io.ReadAll(io.LimitReader(reader, maxNginxBodySize+1))
		if err != nil {
			return fmt.Errorf("reading response from nginx: %w", err)
		}

		truncated := len(b) > maxNginxBodySize
		if truncated {
			b = b[:maxNginxBodySize]
		}

		msg := strings.TrimSpace(string(b))
		if truncated {
			msg += " [truncated]"
		}

		return NginxError(msg)
	}
}

// check interfaces.
var (
	_ error            = NginxError("")
	_ fmt.GoStringer   = NginxError("")
	_ runtime.Consumer = NginxConsumer()
)
