// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package management

import (
	"context"
	"testing"
	"time"

	"github.com/percona/pmm/api/managementpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestDiscoveryService(t *testing.T) {
	// logrus.SetLevel(logrus.DebugLevel)

	sqlDB := testdb.Open(t, models.SetupFixtures)
	defer sqlDB.Close() //nolint:errcheck
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	s := NewDiscoveryService(db)

	t.Run("RDS", func(t *testing.T) {
		t.Run("ListRegions", func(t *testing.T) {
			expected := []string{
				"ap-east-1",
				"ap-northeast-1",
				"ap-northeast-2",
				"ap-south-1",
				"ap-southeast-1",
				"ap-southeast-2",
				"ca-central-1",
				"cn-north-1",
				"cn-northwest-1",
				"eu-central-1",
				"eu-north-1",
				"eu-west-1",
				"eu-west-2",
				"eu-west-3",
				"me-south-1",
				"sa-east-1",
				"us-east-1",
				"us-east-2",
				"us-gov-east-1",
				"us-gov-west-1",
				"us-iso-east-1",
				"us-isob-east-1",
				"us-west-1",
				"us-west-2",
			}
			actual := listRegions([]string{"aws", "aws-cn", "aws-us-gov", "aws-iso", "aws-iso-b"})
			assert.Equal(t, expected, actual)
		})

		t.Run("InvalidClientTokenId", func(t *testing.T) {
			ctx := logger.Set(context.Background(), t.Name())
			accessKey, secretKey := "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" //nolint:gosec

			instances, err := s.DiscoverRDS(ctx, &managementpb.DiscoverRDSRequest{
				AwsAccessKey: accessKey,
				AwsSecretKey: secretKey,
			})

			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "The security token included in the request is invalid."), err)
			assert.Empty(t, instances)
		})

		t.Run("DeadlineExceeded", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
			defer cancel()
			ctx = logger.Set(ctx, t.Name())
			accessKey, secretKey := "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" //nolint:gosec

			instances, err := s.DiscoverRDS(ctx, &managementpb.DiscoverRDSRequest{
				AwsAccessKey: accessKey,
				AwsSecretKey: secretKey,
			})

			tests.AssertGRPCError(t, status.New(codes.DeadlineExceeded, "Request timeout."), err)
			assert.Empty(t, instances)
		})

		t.Run("Normal", func(t *testing.T) {
			ctx := logger.Set(context.Background(), t.Name())
			accessKey, secretKey := tests.GetAWSKeys(t)

			instances, err := s.DiscoverRDS(ctx, &managementpb.DiscoverRDSRequest{
				AwsAccessKey: accessKey,
				AwsSecretKey: secretKey,
			})

			// TODO: Improve this test. https://jira.percona.com/browse/PMM-4896
			// In our current testing env with current AWS keys, 2 regions are returning errors but we don't know why for sure
			// Also, probably we can have more than 1 instance or none. PLEASE UPDATE THIS TESTS !
			assert.NoError(t, err)
			t.Logf("%+v", instances)
			assert.GreaterOrEqualf(t, len(instances.RdsInstances), 1, "Should have at least one instance")
		})
	})
}
