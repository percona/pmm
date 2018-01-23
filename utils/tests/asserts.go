// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package tests contains test helpers.
package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AssertGRPCError(t testing.TB, expected *status.Status, actual error) {
	t.Helper()

	s, ok := status.FromError(actual)
	if !assert.True(t, ok, "expected gRPC Status, got %T:\n%s", actual, actual) {
		return
	}
	assert.Equal(t, expected.Code(), s.Code(), "gRPC status codes are not equal")
	assert.Equal(t, expected.Message(), s.Message(), "gRPC status messages are not equal")
}

func AssertGRPCErrorRE(t testing.TB, expectedCode codes.Code, expectedMessageRE string, actual error) {
	t.Helper()

	s, ok := status.FromError(actual)
	if !assert.True(t, ok, "expected gRPC Status, got %T:\n%s", actual, actual) {
		return
	}
	assert.Equal(t, expectedCode, s.Code(), "gRPC status codes are not equal")
	assert.Regexp(t, expectedMessageRE, s.Message(), "gRPC status message does not match")
}
