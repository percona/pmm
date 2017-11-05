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

// Package rds contains business logic of working with AWS RDS.
package rds

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestRDS(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	t.Run("OK", func(t *testing.T) {
		ctx, _ := logger.Set(context.Background(), t.Name())

		accessKey, secretKey := os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY")
		svc, err := NewService()
		require.NoError(t, err)
		res, err := svc.Get(ctx, accessKey, secretKey)
		require.NoError(t, err)
		for _, db := range res {
			t.Logf("%+v", db)
		}
	})

	t.Run("WrongKeys", func(t *testing.T) {
		ctx, _ := logger.Set(context.Background(), t.Name())

		accessKey, secretKey := "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
		svc, err := NewService()
		require.NoError(t, err)
		res, err := svc.Get(ctx, accessKey, secretKey)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `The security token included in the request is invalid.`), err)
		assert.Empty(t, res)
	})
}
