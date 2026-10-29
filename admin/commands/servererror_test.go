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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

// gRPC status codes PMM Server carries in its response payloads.
const (
	grpcUnauthenticated  = 16
	grpcInternal         = 13
	grpcPermissionDenied = 7
	grpcNotFound         = 5
)

func TestServerErrorMessage(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		err      Error
		expected string
	}{
		"rejected credentials": {
			err:      Error{Code: 401, Error: "Invalid username or password", GRPCCode: grpcUnauthenticated},
			expected: "Invalid username or password. Please check username and password.",
		},
		"internal error mapped to 401": {
			// PMM-15186 reported this exact response being blamed on the credentials.
			// The trailing period of the message must not be doubled up.
			err:      Error{Code: 401, Error: "Internal server error.", GRPCCode: grpcInternal},
			expected: "Internal server error. Please check PMM Server logs.",
		},
		"401 without a gRPC code": {
			err:      Error{Code: 401, Error: "Unauthorized"},
			expected: "Unauthorized. Please check username and password.",
		},
		"not found": {
			err:      Error{Code: 404, Error: "Agent with ID 722fbfc8 not found.", GRPCCode: grpcNotFound},
			expected: "Agent with ID 722fbfc8 not found.",
		},
		"forbidden": {
			err:      Error{Code: 403, Error: "Access denied", GRPCCode: grpcPermissionDenied},
			expected: "Access denied. Please check that your PMM user has sufficient permissions.",
		},
		"internal error mapped to 403": {
			err:      Error{Code: 403, Error: "Internal server error.", GRPCCode: grpcInternal},
			expected: "Internal server error. Please check PMM Server logs.",
		},
		"message ending in an ellipsis": {
			// Only the final period may go: trimming every trailing one truncated
			// the message.
			err:      Error{Code: 401, Error: "Waiting for PMM Server...", GRPCCode: grpcInternal},
			expected: "Waiting for PMM Server... Please check PMM Server logs.",
		},
		"message ending in a newline": {
			err:      Error{Code: 401, Error: "Internal server error.\n", GRPCCode: grpcInternal},
			expected: "Internal server error. Please check PMM Server logs.",
		},
		"message ending in a period and a space": {
			err:      Error{Code: 401, Error: "Internal server error. ", GRPCCode: grpcInternal},
			expected: "Internal server error. Please check PMM Server logs.",
		},
		"empty message": {
			// The generated payloads omit the message, so the hint must be able to
			// stand on its own instead of being introduced by a stray period.
			err:      Error{Code: 401, Error: "", GRPCCode: grpcUnauthenticated},
			expected: "Please check username and password.",
		},
		"empty message without a hint": {
			err:      Error{Code: 404, Error: "", GRPCCode: grpcNotFound},
			expected: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, ServerErrorMessage(tc.err))
		})
	}
}

// TestGetErrorFromGeneratedResponse checks the reflection in GetError against a real
// generated response type, including the gRPC code ServerErrorMessage relies on.
func TestGetErrorFromGeneratedResponse(t *testing.T) {
	t.Parallel()

	t.Run("with gRPC code", func(t *testing.T) {
		t.Parallel()

		resp := agents.NewChangeAgentDefault(401)
		resp.Payload = &agents.ChangeAgentDefaultBody{ //nolint:exhaustruct
			Code:    grpcUnauthenticated,
			Message: "Invalid username or password",
		}

		e := GetError(resp)
		assert.Equal(t, 401, e.Code)
		assert.Equal(t, "Invalid username or password", e.Error)
		assert.Equal(t, int32(16), e.GRPCCode)
		assert.Equal(t, "Invalid username or password. Please check username and password.", ServerErrorMessage(e))
	})

	t.Run("internal error mapped to 401", func(t *testing.T) {
		t.Parallel()

		// The exact response reported in PMM-15186.
		resp := agents.NewChangeAgentDefault(401)
		resp.Payload = &agents.ChangeAgentDefaultBody{ //nolint:exhaustruct
			Code:    grpcInternal,
			Message: "Internal server error.",
		}

		e := GetError(resp)
		assert.Equal(t, int32(13), e.GRPCCode)
		assert.Equal(t, "Internal server error. Please check PMM Server logs.", ServerErrorMessage(e))
	})

	t.Run("without gRPC code", func(t *testing.T) {
		t.Parallel()

		resp := agents.NewChangeAgentDefault(404)
		resp.Payload = &agents.ChangeAgentDefaultBody{Message: "Agent not found."} //nolint:exhaustruct

		e := GetError(resp)
		assert.Equal(t, 404, e.Code)
		assert.Zero(t, e.GRPCCode)
		assert.Equal(t, "Agent not found.", ServerErrorMessage(e))
	})
}

// TestServerErrorJSONShapeUnchanged guards the documented `pmm-admin --json` error shape:
// GRPCCode is internal and must not leak into it.
func TestServerErrorJSONShapeUnchanged(t *testing.T) {
	t.Parallel()

	b, err := json.Marshal(Error{Code: 401, Error: "Invalid username or password", GRPCCode: grpcUnauthenticated})
	require.NoError(t, err)
	assert.JSONEq(t, `{"code":401,"error":"Invalid username or password"}`, string(b))
}
