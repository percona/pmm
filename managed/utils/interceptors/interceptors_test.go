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

package interceptors

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLogRequest(t *testing.T) {
	t.Parallel()

	l := logrus.WithField("test", t.Name())

	for _, tc := range []struct {
		name string
		err  error
		code codes.Code
	}{
		{"nil", nil, codes.OK},
		{"gRPC error is kept", status.Error(codes.NotFound, "not found"), codes.NotFound},
		{"unexpected error", errors.New("boom"), codes.Internal},
		{"unrelated pq error", &pq.Error{Code: "23505"}, codes.Internal},
		{"database is shutting down", fmt.Errorf("query: %w", &pq.Error{Code: "57P03"}), codes.Unavailable},
		{"admin shutdown", &pq.Error{Code: "57P01"}, codes.Unavailable},
		{"bad connection", fmt.Errorf("query: %w", driver.ErrBadConn), codes.Unavailable},
		{
			"connection refused",
			fmt.Errorf("query: %w", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}),
			codes.Unavailable,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := logRequest(l, "test", func() error { return tc.err })
			assert.Equal(t, tc.code, status.Code(err))
		})
	}
}
