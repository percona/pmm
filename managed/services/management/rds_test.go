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

package management

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

func TestRDSService(t *testing.T) {
	// logrus.SetLevel(logrus.DebugLevel)

	uuid.SetRand(&tests.IDReader{})
	defer uuid.SetRand(nil)

	sqlDB := testdb.Open(t, database.SetupFixtures, nil)
	defer sqlDB.Close() //nolint:errcheck
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	cc := &mockConnectionChecker{}
	cc.Test(t)
	sib := &mockServiceInfoBroker{}
	sib.Test(t)
	state := &mockAgentsStateUpdater{}
	state.Test(t)
	ar := &mockAgentsRegistry{}
	ar.Test(t)
	vmdb := &mockPrometheusService{}
	vmdb.Test(t)
	vc := &mockVersionCache{}
	vc.Test(t)
	grafanaClient := &mockGrafanaClient{}
	grafanaClient.Test(t)
	vmClient := &mockVictoriaMetricsClient{}
	vmClient.Test(t)

	defer func() {
		cc.AssertExpectations(t)
		state.AssertExpectations(t)
		ar.AssertExpectations(t)
		vmdb.AssertExpectations(t)
		sib.AssertExpectations(t)
		vc.AssertExpectations(t)
		vmClient.AssertExpectations(t)
	}()

	s := NewManagementService(db, ar, state, cc, sib, vmdb, vc, grafanaClient, vmClient)

	t.Run("DiscoverRDS", func(t *testing.T) {
		t.Run("ListRegions", func(t *testing.T) {
			expected := []string{
				"af-south-1",
				"ap-east-1",
				"ap-northeast-1",
				"ap-northeast-2",
				"ap-northeast-3",
				"ap-south-1",
				"ap-south-2",
				"ap-southeast-1",
				"ap-southeast-2",
				"ap-southeast-3",
				"ap-southeast-4",
				"ca-central-1",
				"ca-west-1",
				"cn-north-1",
				"cn-northwest-1",
				"eu-central-1",
				"eu-central-2",
				"eu-north-1",
				"eu-south-1",
				"eu-south-2",
				"eu-west-1",
				"eu-west-2",
				"eu-west-3",
				"il-central-1",
				"me-central-1",
				"me-south-1",
				"sa-east-1",
				"us-east-1",
				"us-east-2",
				"us-gov-east-1",
				"us-gov-west-1",
				"us-iso-east-1",
				"us-iso-west-1",
				"us-isob-east-1",
				"us-west-1",
				"us-west-2",
			}
			actual := listRegions([]string{"aws", "aws-cn", "aws-us-gov", "aws-iso", "aws-iso-b"})
			assert.Equal(t, expected, actual)
		})

		t.Run("InvalidClientTokenId", func(t *testing.T) {
			ctx := logger.Set(context.Background(), t.Name())
			accessKey, secretKey := "EXAMPLE_ACCESS_KEY", "EXAMPLE_SECRET_KEY"

			instances, err := s.DiscoverRDS(ctx, &managementv1.DiscoverRDSRequest{
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
			accessKey, secretKey := "EXAMPLE_ACCESS_KEY", "EXAMPLE_SECRET_KEY"

			instances, err := s.DiscoverRDS(ctx, &managementv1.DiscoverRDSRequest{
				AwsAccessKey: accessKey,
				AwsSecretKey: secretKey,
			})

			tests.AssertGRPCError(t, status.New(codes.DeadlineExceeded, "Request timeout."), err)
			assert.Empty(t, instances)
		})

		t.Run("Normal", func(t *testing.T) {
			ctx := logger.Set(context.Background(), t.Name())
			accessKey, secretKey := tests.GetAWSKeys(t)

			instances, err := s.DiscoverRDS(ctx, &managementv1.DiscoverRDSRequest{
				AwsAccessKey: accessKey,
				AwsSecretKey: secretKey,
			})

			require.NoError(t, err)
			assert.Equal(t, 4, len(instances.RdsInstances), "Should have four instances")
			assert.Equal(t, []*managementv1.DiscoverRDSInstance{
				{
					Region:        "us-east-1",
					Az:            "us-east-1a",
					InstanceId:    "autotest-aurora-mysql-56",
					NodeModel:     "db.t2.medium",
					Address:       "autotest-aurora-mysql-56.cstdx0tr6tzx.us-east-1.rds.amazonaws.com",
					Port:          3306,
					Engine:        managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_MYSQL,
					EngineVersion: "5.6.mysql_aurora.1.22.2",
				},
				{
					Region:        "us-east-1",
					Az:            "us-east-1d",
					InstanceId:    "autotest-psql-10",
					NodeModel:     "db.t2.micro",
					Address:       "autotest-psql-10.cstdx0tr6tzx.us-east-1.rds.amazonaws.com",
					Port:          5432,
					Engine:        managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_POSTGRESQL,
					EngineVersion: "10.16",
				},
				{
					Region:        "us-west-2",
					Az:            "us-west-2b",
					InstanceId:    "autotest-aurora-psql-11",
					NodeModel:     "db.r4.large",
					Address:       "autotest-aurora-psql-11.c3uoaol27cbb.us-west-2.rds.amazonaws.com",
					Port:          5432,
					Engine:        managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_POSTGRESQL,
					EngineVersion: "11.9",
				},
				{
					Region:        "us-west-2",
					Az:            "us-west-2c",
					InstanceId:    "autotest-mysql-57",
					NodeModel:     "db.t2.micro",
					Address:       "autotest-mysql-57.c3uoaol27cbb.us-west-2.rds.amazonaws.com",
					Port:          3306,
					Engine:        managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_MYSQL,
					EngineVersion: "5.7.22",
				},
			}, instances.RdsInstances)
		})

		type instance struct {
			az         string
			instanceID string
		}

		for _, tt := range []struct {
			region    string
			instances []instance
		}{
			{"us-east-1", []instance{{"us-east-1a", "autotest-aurora-mysql-56"}, {"us-east-1d", "autotest-psql-10"}}},
			{"us-west-2", []instance{{"us-west-2b", "autotest-aurora-psql-11"}, {"us-west-2c", "autotest-mysql-57"}}},
		} {
			t.Run(fmt.Sprintf("discoverRDSRegion %s", tt.region), func(t *testing.T) {
				ctx := logger.Set(context.Background(), t.Name())
				accessKey, secretKey := tests.GetAWSKeys(t)

				creds := credentials.NewStaticCredentials(accessKey, secretKey, "")
				cfg := &aws.Config{
					CredentialsChainVerboseErrors: aws.Bool(true),
					Credentials:                   creds,
					HTTPClient:                    &http.Client{},
				}
				sess, err := session.NewSession(cfg)
				require.NoError(t, err)

				// do not break our API if some AWS region is slow or down
				ctx, cancel := context.WithTimeout(ctx, awsDiscoverTimeout)
				defer cancel()

				instances, err := discoverRDSRegion(ctx, sess, tt.region)

				require.NoError(t, err)
				require.Equal(t, len(tt.instances), len(instances), "Should have two instances")
				// we compare instances this way because there are too much fields that we don't need to compare.
				for i, instance := range tt.instances {
					assert.Equal(t, instance.az, pointer.GetString(instances[i].AvailabilityZone))
					assert.Equal(t, instance.instanceID, pointer.GetString(instances[i].DBInstanceIdentifier))
				}
			})
		}
	})

	t.Run("AddRDS", func(t *testing.T) {
		ctx := logger.Set(context.Background(), t.Name())
		accessKey, secretKey := "EXAMPLE_ACCESS_KEY", "EXAMPLE_SECRET_KEY"

		req := &managementv1.AddRDSServiceParams{
			Region:             "us-east-1",
			Az:                 "us-east-1b",
			InstanceId:         "rds-mysql57",
			NodeModel:          "db.t3.micro",
			Address:            "rds-mysql57-renaming.xyzzy.us-east-1.rds.amazonaws.com",
			Port:               3306,
			Engine:             managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_MYSQL,
			Environment:        "production",
			Cluster:            "c-01",
			ReplicationSet:     "rs-01",
			Username:           "username",
			Password:           "password",
			AwsAccessKey:       accessKey,
			AwsSecretKey:       secretKey,
			RdsExporter:        true,
			QanMysqlPerfschema: true,
			CustomLabels: map[string]string{
				"foo": "bar",
			},
			SkipConnectionCheck:       true,
			Tls:                       false,
			TlsSkipVerify:             false,
			DisableQueryExamples:      true,
			TablestatsGroupTableLimit: 0,
		}

		state.On("RequestStateUpdate", ctx, "pmm-server")
		resp, err := s.addRDS(ctx, req)
		require.NoError(t, err)

		expected := &managementv1.AddServiceResponse{
			Service: &managementv1.AddServiceResponse_Rds{
				Rds: &managementv1.RDSServiceResult{
					Node: &inventoryv1.RemoteRDSNode{
						NodeId:    "00000000-0000-4000-8000-000000000005",
						NodeName:  "rds-mysql57",
						Address:   "rds-mysql57",
						NodeModel: "db.t3.micro",
						Region:    "us-east-1",
						Az:        "us-east-1b",
						CustomLabels: map[string]string{
							"foo": "bar",
						},
					},
					RdsExporter: &inventoryv1.RDSExporter{
						AgentId:      "00000000-0000-4000-8000-000000000006",
						PmmAgentId:   "pmm-server",
						NodeId:       "00000000-0000-4000-8000-000000000005",
						AwsAccessKey: "EXAMPLE_ACCESS_KEY",
						Status:       inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
					},
					Mysql: &inventoryv1.MySQLService{
						ServiceId:      "00000000-0000-4000-8000-000000000007",
						NodeId:         "00000000-0000-4000-8000-000000000005",
						Address:        "rds-mysql57-renaming.xyzzy.us-east-1.rds.amazonaws.com",
						Port:           3306,
						Environment:    "production",
						Cluster:        "c-01",
						ReplicationSet: "rs-01",
						ServiceName:    "rds-mysql57",
						CustomLabels: map[string]string{
							"foo": "bar",
						},
					},
					MysqldExporter: &inventoryv1.MySQLdExporter{
						AgentId:                   "00000000-0000-4000-8000-000000000008",
						PmmAgentId:                "pmm-server",
						ServiceId:                 "00000000-0000-4000-8000-000000000007",
						Username:                  "username",
						TablestatsGroupTableLimit: 1000,
						Status:                    inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
					},
					QanMysqlPerfschema: &inventoryv1.QANMySQLPerfSchemaAgent{
						AgentId:               "00000000-0000-4000-8000-000000000009",
						PmmAgentId:            "pmm-server",
						ServiceId:             "00000000-0000-4000-8000-000000000007",
						Username:              "username",
						QueryExamplesDisabled: true,
						Status:                inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
					},
				},
			},
		}
		assert.Equal(t, prototext.Format(expected), prototext.Format(resp)) // for better diffs
	})

	t.Run("AddRDSPostgreSQL", func(t *testing.T) {
		ctx := logger.Set(context.Background(), t.Name())
		accessKey, secretKey := "EXAMPLE_ACCESS_KEY", "EXAMPLE_SECRET_KEY"

		req := &managementv1.AddRDSServiceParams{
			Region:                    "us-east-1",
			Az:                        "us-east-1b",
			InstanceId:                "rds-postgresql",
			NodeModel:                 "db.t3.micro",
			Address:                   "rds-postgresql-renaming.xyzzy.us-east-1.rds.amazonaws.com",
			Port:                      3306,
			Engine:                    managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_POSTGRESQL,
			Environment:               "production",
			Cluster:                   "c-01",
			ReplicationSet:            "rs-01",
			Username:                  "username",
			Password:                  "password",
			AwsAccessKey:              accessKey,
			AwsSecretKey:              secretKey,
			RdsExporter:               true,
			QanPostgresqlPgstatements: true,
			CustomLabels: map[string]string{
				"foo": "bar",
			},
			SkipConnectionCheck:              true,
			Tls:                              false,
			TlsSkipVerify:                    false,
			DisableQueryExamples:             true,
			TablestatsGroupTableLimit:        0,
			AutoDiscoveryLimit:               10,
			MaxPostgresqlExporterConnections: 15,
		}

		state.On("RequestStateUpdate", ctx, "pmm-server")
		resp, err := s.addRDS(ctx, req)
		require.NoError(t, err)

		expected := &managementv1.AddServiceResponse{
			Service: &managementv1.AddServiceResponse_Rds{
				Rds: &managementv1.RDSServiceResult{
					Node: &inventoryv1.RemoteRDSNode{
						NodeId:    "00000000-0000-4000-8000-00000000000a",
						NodeName:  "rds-postgresql",
						Address:   "rds-postgresql",
						NodeModel: "db.t3.micro",
						Region:    "us-east-1",
						Az:        "us-east-1b",
						CustomLabels: map[string]string{
							"foo": "bar",
						},
					},
					RdsExporter: &inventoryv1.RDSExporter{
						AgentId:      "00000000-0000-4000-8000-00000000000b",
						PmmAgentId:   "pmm-server",
						NodeId:       "00000000-0000-4000-8000-00000000000a",
						AwsAccessKey: "EXAMPLE_ACCESS_KEY",
						Status:       inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
					},
					Postgresql: &inventoryv1.PostgreSQLService{
						ServiceId:      "00000000-0000-4000-8000-00000000000c",
						NodeId:         "00000000-0000-4000-8000-00000000000a",
						Address:        "rds-postgresql-renaming.xyzzy.us-east-1.rds.amazonaws.com",
						Port:           3306,
						Environment:    "production",
						Cluster:        "c-01",
						ReplicationSet: "rs-01",
						ServiceName:    "rds-postgresql",
						DatabaseName:   "postgres",
						CustomLabels: map[string]string{
							"foo": "bar",
						},
					},
					PostgresqlExporter: &inventoryv1.PostgresExporter{
						AgentId:                "00000000-0000-4000-8000-00000000000d",
						PmmAgentId:             "pmm-server",
						ServiceId:              "00000000-0000-4000-8000-00000000000c",
						Username:               "username",
						Status:                 inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
						AutoDiscoveryLimit:     10,
						MaxExporterConnections: 15,
					},
					QanPostgresqlPgstatements: &inventoryv1.QANPostgreSQLPgStatementsAgent{
						AgentId:    "00000000-0000-4000-8000-00000000000e",
						PmmAgentId: "pmm-server",
						ServiceId:  "00000000-0000-4000-8000-00000000000c",
						Username:   "username",
						Status:     inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
					},
				},
			},
		}
		assert.Equal(t, prototext.Format(expected), prototext.Format(resp)) // for better diffs
	})
}
