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
	"errors"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/models"
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
		{"database unavailable", fmt.Errorf("query: %w", models.ErrDatabaseUnavailable), codes.Unavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := logRequest(l, "test", func() error { return tc.err })
			assert.Equal(t, tc.code, status.Code(err))
		})
	}
}
