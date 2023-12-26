// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package tests provides tests helpers.
package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AssertGRPCError checks that actual error has the same gRPC error code and message as expected error.
func AssertGRPCError(tb testing.TB, expected *status.Status, actual error) {
	tb.Helper()

	s, ok := status.FromError(actual)
	if !assert.True(tb, ok, "expected gRPC Status, got %T:\n%s", actual, actual) {
		return
	}
	err := s.Err()
	if !assert.Error(tb, err) { //nolint:testifylint
		return
	}
	assert.Equal(tb, expected.Err().Error(), err.Error()) // gives the best error message
}

// AssertGRPCErrorRE checks that actual error has expected gRPC error code, and error messages
// matches expected regular expression.
func AssertGRPCErrorRE(tb testing.TB, expectedCode codes.Code, expectedMessageRE string, actual error) {
	tb.Helper()

	s, ok := status.FromError(actual)
	if !assert.True(tb, ok, "expected gRPC Status, got %T:\n%s", actual, actual) {
		return
	}
	assert.Equal(tb, int(expectedCode), int(s.Code()), "gRPC status codes are not equal") // int() to log in decimal, not hex
	assert.Regexp(tb, expectedMessageRE, s.Message(), "gRPC status message does not match")
}

// AssertGRPCErrorCode checks the error codes are equal.
func AssertGRPCErrorCode(tb testing.TB, expectedCode codes.Code, actual error) {
	tb.Helper()

	s, ok := status.FromError(actual)
	if !assert.True(tb, ok, "expected gRPC Status, got %T:\n%s", actual, actual) {
		return
	}

	assert.Equal(tb, expectedCode, s.Code())
}
