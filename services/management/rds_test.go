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

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestRDSService(t *testing.T) {
	// logrus.SetLevel(logrus.DebugLevel)

	uuid.SetRand(new(tests.IDReader))
	defer uuid.SetRand(nil)

	sqlDB := testdb.Open(t, models.SetupFixtures)
	defer sqlDB.Close() //nolint:errcheck
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	r := new(mockAgentsRegistry)
	r.Test(t)
	defer r.AssertExpectations(t)
	s := NewRDSService(db, r)

	t.Run("DiscoverRDS", func(t *testing.T) {
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

	t.Run("AddMySQL", func(t *testing.T) {
		ctx := logger.Set(context.Background(), t.Name())
		accessKey, secretKey := "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" //nolint:gosec

		req := &managementpb.AddRDSRequest{
			Region:                    "us-east-1",
			Az:                        "us-east-1b",
			InstanceId:                "rds-mysql57",
			NodeModel:                 "db.t3.micro",
			Address:                   "rds-mysql57-renaming.xyzzy.us-east-1.rds.amazonaws.com",
			Port:                      3306,
			Engine:                    managementpb.DiscoverRDSEngine_DISCOVER_RDS_MYSQL,
			Environment:               "production",
			Cluster:                   "c-01",
			ReplicationSet:            "rs-01",
			Username:                  "username",
			Password:                  "password",
			AwsAccessKey:              accessKey,
			AwsSecretKey:              secretKey,
			RdsExporter:               true,
			QanMysqlPerfschema:        true,
			SkipConnectionCheck:       true,
			Tls:                       false,
			TlsSkipVerify:             false,
			DisableQueryExamples:      true,
			TablestatsGroupTableLimit: 0,
		}

		r.On("SendSetStateRequest", ctx, "pmm-server")
		resp, err := s.AddRDS(ctx, req)
		require.NoError(t, err)

		expected := &managementpb.AddRDSResponse{
			Node: &inventorypb.RemoteRDSNode{
				NodeId:    "/node_id/00000000-0000-4000-8000-000000000005",
				NodeName:  "rds-mysql57",
				Address:   "rds-mysql57",
				NodeModel: "db.t3.micro",
				Region:    "us-east-1",
				Az:        "us-east-1b",
			},
			RdsExporter: &inventorypb.RDSExporter{
				AgentId:      "/agent_id/00000000-0000-4000-8000-000000000006",
				PmmAgentId:   "pmm-server",
				NodeId:       "/node_id/00000000-0000-4000-8000-000000000005",
				AwsAccessKey: "AKIAIOSFODNN7EXAMPLE",
			},
			Mysql: &inventorypb.MySQLService{
				ServiceId:      "/service_id/00000000-0000-4000-8000-000000000007",
				NodeId:         "/node_id/00000000-0000-4000-8000-000000000005",
				Address:        "rds-mysql57-renaming.xyzzy.us-east-1.rds.amazonaws.com",
				Port:           3306,
				Environment:    "production",
				Cluster:        "c-01",
				ReplicationSet: "rs-01",
				ServiceName:    "rds-mysql57",
			},
			MysqldExporter: &inventorypb.MySQLdExporter{
				AgentId:                   "/agent_id/00000000-0000-4000-8000-000000000008",
				PmmAgentId:                "pmm-server",
				ServiceId:                 "/service_id/00000000-0000-4000-8000-000000000007",
				Username:                  "username",
				TablestatsGroupTableLimit: 1000,
			},
			QanMysqlPerfschema: &inventorypb.QANMySQLPerfSchemaAgent{
				AgentId:               "/agent_id/00000000-0000-4000-8000-000000000009",
				PmmAgentId:            "pmm-server",
				ServiceId:             "/service_id/00000000-0000-4000-8000-000000000007",
				Username:              "username",
				QueryExamplesDisabled: true,
			},
		}
		assert.Equal(t, proto.MarshalTextString(expected), proto.MarshalTextString(resp)) // for better diffs
	})
}
