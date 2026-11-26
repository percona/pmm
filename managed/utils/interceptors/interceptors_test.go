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
	"os"
	"syscall"
	"testing"

	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLogRequestErrorCode(t *testing.T) {
	t.Parallel()

	dialErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: os.NewSyscallError("connect", syscall.ECONNREFUSED),
	}

	for name, tc := range map[string]struct {
		err      error
		expected codes.Code
	}{
		"database shutting down": {
			err:      fmt.Errorf("failed to select settings: %w", &pq.Error{Code: pqerror.CannotConnectNow}),
			expected: codes.Unavailable,
		},
		"connection terminated": {
			err:      &pq.Error{Code: pqerror.AdminShutdown},
			expected: codes.Unavailable,
		},
		"connection refused": {
			err:      fmt.Errorf("failed to select settings: %w", dialErr),
			expected: codes.Unavailable,
		},
		"connection reset": {
			err:      &net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("read", syscall.ECONNRESET)},
			expected: codes.Unavailable,
		},
		"bad connection": {
			err:      fmt.Errorf("failed to insert service: %w", driver.ErrBadConn),
			expected: codes.Unavailable,
		},
		"other database error": {
			err:      &pq.Error{Code: pqerror.UniqueViolation},
			expected: codes.Internal,
		},
		"other error": {
			err:      errors.New("boom"),
			expected: codes.Internal,
		},
		"gRPC error": {
			err:      status.Error(codes.NotFound, "not found"),
			expected: codes.NotFound,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := logRequest(logrus.WithField("test", t.Name()), "RPC test", func() error { return tc.err })

			assert.Equal(t, tc.expected, status.Code(err))
		})
	}
}
