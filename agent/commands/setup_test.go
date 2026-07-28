// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"crypto/x509"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
	"github.com/percona/pmm/utils/servererror"
)

// registerDefault builds the error the generated client returns for a failed registration.
func registerDefault(httpCode int, grpcCode int32, message string) *mservice.RegisterNodeDefault {
	resp := mservice.NewRegisterNodeDefault(httpCode)
	resp.Payload = &mservice.RegisterNodeDefaultBody{ //nolint:exhaustruct
		Code:    grpcCode,
		Message: message,
	}

	return resp
}

func TestRegisterErrorMessage(t *testing.T) {
	t.Parallel()

	const (
		grpcUnauthenticated  = 16
		grpcInternal         = 13
		grpcPermissionDenied = 7
		grpcAlreadyExists    = 6
	)

	t.Run("rejected credentials", func(t *testing.T) {
		t.Parallel()

		msg := registerErrorMessage(
			registerDefault(http.StatusUnauthorized, grpcUnauthenticated, "Invalid username or password"),
			"pmm-server", false)
		assert.Equal(t, "Invalid username or password\nPlease check username and password", msg)
	})

	t.Run("internal error mapped to 401", func(t *testing.T) {
		t.Parallel()

		// PMM Server maps internal authentication errors onto HTTP 401 as well, so the
		// credentials must not be blamed for them.
		msg := registerErrorMessage(
			registerDefault(http.StatusUnauthorized, grpcInternal, "Internal server error."),
			"pmm-server", false)
		assert.Equal(t, "Internal server error.\nPlease check PMM Server logs", msg)
	})

	t.Run("access denied", func(t *testing.T) {
		t.Parallel()

		// Not a credentials problem: the user authenticated but lacks the required role.
		msg := registerErrorMessage(
			registerDefault(http.StatusForbidden, grpcPermissionDenied, "Access denied"),
			"pmm-server", false)
		assert.Equal(t, "Access denied", msg)
	})

	t.Run("node already exists", func(t *testing.T) {
		t.Parallel()

		msg := registerErrorMessage(
			registerDefault(http.StatusConflict, grpcAlreadyExists, "Node with name \"node\" already exists."),
			"pmm-server", false)
		assert.Equal(t, "Node with name \"node\" already exists. If you want override node, use --force option", msg)
	})

	t.Run("certificate cannot be verified", func(t *testing.T) {
		t.Parallel()

		certErr := &url.Error{
			Op:  "Post",
			URL: "https://pmm-server:8443/v1/management/nodes",
			Err: x509.HostnameError{Certificate: &x509.Certificate{}, Host: "pmm-server"}, //nolint:exhaustruct
		}

		msg := registerErrorMessage(certErr, "pmm-server", false)
		assert.Contains(t, msg, "PMM Server TLS certificate could not be verified")
		assert.Contains(t, msg, `not valid for host "pmm-server"`)
		assert.Contains(t, msg, servererror.InsecureTLSFlag)
	})

	t.Run("certificate error with validation disabled", func(t *testing.T) {
		t.Parallel()

		certErr := &url.Error{
			Op:  "Post",
			URL: "https://pmm-server:8443/v1/management/nodes",
			Err: x509.HostnameError{Certificate: &x509.Certificate{}, Host: "pmm-server"}, //nolint:exhaustruct
		}

		msg := registerErrorMessage(certErr, "pmm-server", true)
		assert.Equal(t, certErr.Error(), msg)
		assert.NotContains(t, msg, servererror.InsecureTLSFlag)
	})

	t.Run("nginx response", func(t *testing.T) {
		t.Parallel()

		msg := registerErrorMessage(nginxError("502 Bad Gateway"), "pmm-server", false)
		assert.Equal(t, "response from nginx: 502 Bad Gateway.\nPlease check pmm-managed logs.", msg)
	})
}
