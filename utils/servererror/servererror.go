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
	"net/http"
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
	// authentication error - internal ones included - onto HTTP 401 as well. Those are not
	// caused by wrong credentials and must not be reported as such.
	case httpCode == http.StatusUnauthorized:
		return "Please check PMM Server logs"
	}

	return ""
}
