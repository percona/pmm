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
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
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
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

func TestRDSService(t *testing.T) {
	// logrus.SetLevel(logrus.DebugLevel)

	uuid.SetRand(&tests.IDReader{})
	defer uuid.SetRand(nil)

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		assert.NoError(t, sqlDB.Close())
	})
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

	s := NewManagementService(db, ar, state, cc, sib, vmdb, vc, grafanaClient, vmClient, nil, nil, false)

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
			ctx := logger.Set(t.Context(), t.Name())
			accessKey, secretKey := "EXAMPLE_ACCESS_KEY", "EXAMPLE_SECRET_KEY"

			instances, err := s.DiscoverRDS(ctx, &managementv1.DiscoverRDSRequest{
				AwsAccessKey: accessKey,
				AwsSecretKey: secretKey,
			})

			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "The security token included in the request is invalid."), err)
			assert.Empty(t, instances)
		})

		t.Run("DeadlineExceeded", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Nanosecond)
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

		t.Run("RoleARNPartitionNotEnabled", func(t *testing.T) {
			ctx := logger.Set(t.Context(), t.Name())

			// Default settings enable only the aws partition.
			instances, err := s.DiscoverRDS(ctx, &managementv1.DiscoverRDSRequest{
				AwsRoleArn: "arn:aws-cn:iam::123456789012:role/pmm-monitoring",
			})

			tests.AssertGRPCError(t, status.New(codes.FailedPrecondition,
				"Role arn:aws-cn:iam::123456789012:role/pmm-monitoring belongs to AWS partition aws-cn, which is not enabled in PMM settings."), err)
			assert.Nil(t, instances)
		})

		t.Run("RoleARNScansOnlyItsPartition", func(t *testing.T) {
			ctx := logger.Set(t.Context(), t.Name())

			settings, err := models.GetSettings(db.Querier)
			require.NoError(t, err)
			_, err = models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{AWSPartitions: []string{"aws", "aws-cn"}})
			require.NoError(t, err)
			t.Cleanup(func() {
				_, err := models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{AWSPartitions: settings.AWSPartitions})
				assert.NoError(t, err)
			})

			fake := setupRoleDiscovery(t)

			instances, err := s.DiscoverRDS(ctx, &managementv1.DiscoverRDSRequest{
				AwsRoleArn: "arn:aws:iam::123456789012:role/pmm-monitoring",
			})
			require.NoError(t, err)
			assert.Equal(t, []*managementv1.DiscoverRDSInstance{
				{
					Region:        "eu-north-1",
					Az:            "eu-north-1a",
					InstanceId:    "pmm-mysql",
					NodeModel:     "db.t4g.micro",
					Address:       "pmm-mysql.abc.eu-north-1.rds.amazonaws.com",
					Port:          3306,
					Engine:        managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_MYSQL,
					EngineVersion: "8.0.36",
				},
			}, instances.RdsInstances)

			// With no region configured on the server, the role is assumed once, against the
			// partition's default region.
			assert.Equal(t, []fakeAWSCall{{"sts", "us-east-1", "arn:aws:iam::123456789012:role/pmm-monitoring"}}, fake.calls("sts"))

			// Only the role's partition is scanned; aws-cn is enabled in settings but never called.
			assert.Equal(t, listRegions([]string{"aws"}), scannedRegions(fake))
		})

		t.Run("RoleARNUsesConfiguredRegion", func(t *testing.T) {
			ctx := logger.Set(t.Context(), t.Name())

			fake := setupRoleDiscovery(t)
			t.Setenv("AWS_REGION", "eu-west-1")

			instances, err := s.DiscoverRDS(ctx, &managementv1.DiscoverRDSRequest{
				AwsRoleArn: "arn:aws:iam::123456789012:role/pmm-monitoring",
			})
			require.NoError(t, err)
			require.Len(t, instances.RdsInstances, 1)
			assert.Equal(t, "eu-north-1", instances.RdsInstances[0].Region)

			// The role is assumed in the server's configured region, not the partition default.
			assert.Equal(t, []fakeAWSCall{{"sts", "eu-west-1", "arn:aws:iam::123456789012:role/pmm-monitoring"}}, fake.calls("sts"))

			// The configured region moves only the STS call; the whole partition is still scanned.
			assert.Equal(t, listRegions([]string{"aws"}), scannedRegions(fake))
		})

		t.Run("RoleARNUsesDefaultRegionVariable", func(t *testing.T) {
			ctx := logger.Set(t.Context(), t.Name())

			fake := setupRoleDiscovery(t)
			t.Setenv("AWS_DEFAULT_REGION", "eu-west-1")

			instances, err := s.DiscoverRDS(ctx, &managementv1.DiscoverRDSRequest{
				AwsRoleArn: "arn:aws:iam::123456789012:role/pmm-monitoring",
			})
			require.NoError(t, err)
			require.Len(t, instances.RdsInstances, 1)

			assert.Equal(t, []fakeAWSCall{{"sts", "eu-west-1", "arn:aws:iam::123456789012:role/pmm-monitoring"}}, fake.calls("sts"))
		})

		t.Run("RoleARNRejectsRegionOutsidePartition", func(t *testing.T) {
			ctx := logger.Set(t.Context(), t.Name())

			fake := setupRoleDiscovery(t)
			t.Setenv("AWS_REGION", "cn-north-1")

			instances, err := s.DiscoverRDS(ctx, &managementv1.DiscoverRDSRequest{
				AwsRoleArn: "arn:aws:iam::123456789012:role/pmm-monitoring",
			})
			tests.AssertGRPCError(t, status.New(codes.FailedPrecondition,
				"AWS region cn-north-1 configured on PMM Server is not in AWS partition aws of role "+
					"arn:aws:iam::123456789012:role/pmm-monitoring; unset AWS_REGION or set it to a region of that partition."), err)
			assert.Nil(t, instances)

			// Rejected before any network call.
			assert.Empty(t, fake.calls("sts"))
			assert.Empty(t, fake.calls("rds"))
		})

		t.Run("Normal", func(t *testing.T) {
			ctx := logger.Set(t.Context(), t.Name())
			accessKey, secretKey := tests.GetAWSKeys(t)

			instances, err := s.DiscoverRDS(ctx, &managementv1.DiscoverRDSRequest{
				AwsAccessKey: accessKey,
				AwsSecretKey: secretKey,
			})

			require.NoError(t, err)
			assert.Len(t, instances.RdsInstances, 4, "Should have four instances")
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
			t.Run("discoverRDSRegion "+tt.region, func(t *testing.T) {
				ctx := logger.Set(t.Context(), t.Name())
				accessKey, secretKey := tests.GetAWSKeys(t)
				creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
				opts := []func(*config.LoadOptions) error{
					config.WithCredentialsProvider(creds),
					config.WithHTTPClient(&http.Client{}),
					config.WithClientLogMode(aws.LogRetries | aws.LogRequestWithBody | aws.LogResponseWithBody),
				}
				cfg, err := config.LoadDefaultConfig(ctx, opts...)
				require.NoError(t, err)

				// do not break our API if some AWS region is slow or down
				ctx, cancel := context.WithTimeout(ctx, awsDiscoverTimeout)
				defer cancel()

				instances, err := discoverRDSRegion(ctx, cfg, tt.region)

				require.NoError(t, err)
				require.Len(t, instances, len(tt.instances), "Should have two instances")
				// we compare instances this way because there are too much fields that we don't need to compare.
				for i, instance := range tt.instances {
					assert.Equal(t, instance.az, pointer.GetString(instances[i].AvailabilityZone))
					assert.Equal(t, instance.instanceID, pointer.GetString(instances[i].DBInstanceIdentifier))
				}
			})
		}
	})

	t.Run("AddRDS", func(t *testing.T) {
		ctx := logger.Set(t.Context(), t.Name())
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
			MysqlDisableCollectors:    []string{"global_status", "info_schema.innodb_metrics"},
		}

		state.On("RequestStateUpdate", ctx, "pmm-server")
		resp, err := s.addRDS(ctx, req)
		require.NoError(t, err)

		expected := &managementv1.AddServiceResponse{
			Service: &managementv1.AddServiceResponse_Rds{
				Rds: &managementv1.RDSServiceResult{
					Node: &inventoryv1.RemoteRDSNode{
						NodeId:     "00000000-0000-4000-8000-000000000005",
						NodeName:   "rds-mysql57",
						Address:    "rds-mysql57-renaming.xyzzy.us-east-1.rds.amazonaws.com",
						InstanceId: "rds-mysql57",
						NodeModel:  "db.t3.micro",
						Region:     "us-east-1",
						Az:         "us-east-1b",
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
						DisabledCollectors:        []string{"global_status", "info_schema.innodb_metrics"},
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
		ctx := logger.Set(t.Context(), t.Name())
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
			PostgresqlDisableCollectors:      []string{"stat_database", "stat_bgwriter"},
		}

		state.On("RequestStateUpdate", ctx, "pmm-server")
		resp, err := s.addRDS(ctx, req)
		require.NoError(t, err)

		expected := &managementv1.AddServiceResponse{
			Service: &managementv1.AddServiceResponse_Rds{
				Rds: &managementv1.RDSServiceResult{
					Node: &inventoryv1.RemoteRDSNode{
						NodeId:     "00000000-0000-4000-8000-00000000000a",
						NodeName:   "rds-postgresql",
						Address:    "rds-postgresql-renaming.xyzzy.us-east-1.rds.amazonaws.com",
						InstanceId: "rds-postgresql",
						NodeModel:  "db.t3.micro",
						Region:     "us-east-1",
						Az:         "us-east-1b",
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
						DisabledCollectors:     []string{"stat_database", "stat_bgwriter"},
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

func TestAssumeRoleProvider(t *testing.T) {
	t.Parallel()

	provider := assumeRoleProvider(aws.Config{Region: "eu-west-1"}, "arn:aws:iam::123456789012:role/pmm-monitoring")
	require.NotNil(t, provider)
	assert.IsType(t, &aws.CredentialsCache{}, provider)
}

func TestSTSRegionForRoleARN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		roleARN       string
		wantRegion    string
		wantPartition string
		wantErr       bool
	}{
		{
			name:          "aws partition",
			roleARN:       "arn:aws:iam::123456789012:role/pmm-monitoring",
			wantRegion:    "us-east-1",
			wantPartition: "aws",
		},
		{
			name:          "aws-cn partition",
			roleARN:       "arn:aws-cn:iam::123456789012:role/pmm-monitoring",
			wantRegion:    "cn-north-1",
			wantPartition: "aws-cn",
		},
		{
			name:          "aws-us-gov partition",
			roleARN:       "arn:aws-us-gov:iam::123456789012:role/pmm-monitoring",
			wantRegion:    "us-gov-west-1",
			wantPartition: "aws-us-gov",
		},
		{
			name:          "aws-iso partition",
			roleARN:       "arn:aws-iso:iam::123456789012:role/pmm-monitoring",
			wantRegion:    "us-iso-east-1",
			wantPartition: "aws-iso",
		},
		{
			name:    "unsupported partition",
			roleARN: "arn:aws-iso-b:iam::123456789012:role/pmm-monitoring",
			wantErr: true,
		},
		{
			name:    "malformed ARN",
			roleARN: "not-an-arn",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			region, partition, err := stsRegionForRoleARN(tt.roleARN)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRegion, region)
			assert.Equal(t, tt.wantPartition, partition)
		})
	}
}

// fakeAWSCall is one signed request fakeAWS received: the service and region come from the
// SigV4 credential scope, roleARN is set for sts:AssumeRole.
type fakeAWSCall struct {
	service string
	region  string
	roleARN string
}

// fakeAWS stands in for the STS and RDS endpoints DiscoverRDS calls (query protocol), so a
// role-based discovery runs without AWS. It returns one MySQL instance in eu-north-1 and
// nothing elsewhere.
type fakeAWS struct {
	*httptest.Server

	mu       sync.Mutex
	received []fakeAWSCall
}

var sigV4Scope = regexp.MustCompile(`Credential=[^/]+/\d{8}/([^/]+)/([^/]+)/aws4_request`)

const (
	fakeSTSAssumeRoleResponse = `<AssumeRoleResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <AssumeRoleResult>
    <Credentials>
      <AccessKeyId>ASIAIOSFODNN7EXAMPLE</AccessKeyId>
      <SecretAccessKey>assumed-secret</SecretAccessKey>
      <SessionToken>assumed-session-token</SessionToken>
      <Expiration>2099-01-01T00:00:00Z</Expiration>
    </Credentials>
    <AssumedRoleUser>
      <Arn>arn:aws:sts::123456789012:assumed-role/pmm-monitoring/pmm</Arn>
      <AssumedRoleId>AROAEXAMPLE:pmm</AssumedRoleId>
    </AssumedRoleUser>
  </AssumeRoleResult>
  <ResponseMetadata><RequestId>fake</RequestId></ResponseMetadata>
</AssumeRoleResponse>`

	fakeRDSEmptyResponse = `<DescribeDBInstancesResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/">
  <DescribeDBInstancesResult><DBInstances/></DescribeDBInstancesResult>
  <ResponseMetadata><RequestId>fake</RequestId></ResponseMetadata>
</DescribeDBInstancesResponse>`

	fakeRDSOneInstanceResponse = `<DescribeDBInstancesResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/">
  <DescribeDBInstancesResult>
    <DBInstances>
      <DBInstance>
        <DBInstanceIdentifier>pmm-mysql</DBInstanceIdentifier>
        <DBInstanceClass>db.t4g.micro</DBInstanceClass>
        <Engine>mysql</Engine>
        <EngineVersion>8.0.36</EngineVersion>
        <AvailabilityZone>eu-north-1a</AvailabilityZone>
        <Endpoint>
          <Address>pmm-mysql.abc.eu-north-1.rds.amazonaws.com</Address>
          <Port>3306</Port>
        </Endpoint>
      </DBInstance>
    </DBInstances>
  </DescribeDBInstancesResult>
  <ResponseMetadata><RequestId>fake</RequestId></ResponseMetadata>
</DescribeDBInstancesResponse>`
)

func newFakeAWS(t *testing.T) *fakeAWS {
	t.Helper()

	f := &fakeAWS{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope := sigV4Scope.FindStringSubmatch(r.Header.Get("Authorization"))
		if scope == nil {
			http.Error(w, "unsigned request", http.StatusBadRequest)
			return
		}
		region, service := scope[1], scope[2]
		action := r.PostFormValue("Action")

		f.mu.Lock()
		f.received = append(f.received, fakeAWSCall{service: service, region: region, roleARN: r.PostFormValue("RoleArn")})
		f.mu.Unlock()

		w.Header().Set("Content-Type", "text/xml")
		switch {
		case service == "sts" && action == "AssumeRole":
			_, _ = io.WriteString(w, fakeSTSAssumeRoleResponse)
		case service == "rds" && action == "DescribeDBInstances" && region == "eu-north-1":
			_, _ = io.WriteString(w, fakeRDSOneInstanceResponse)
		case service == "rds" && action == "DescribeDBInstances":
			_, _ = io.WriteString(w, fakeRDSEmptyResponse)
		default:
			http.Error(w, "unexpected call "+service+" "+action, http.StatusBadRequest)
		}
	}))
	t.Cleanup(f.Close)

	return f
}

// calls returns the recorded calls to service, in arrival order.
func (f *fakeAWS) calls(service string) []fakeAWSCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	var res []fakeAWSCall
	for _, c := range f.received {
		if c.service == service {
			res = append(res, c)
		}
	}
	return res
}

// scannedRegions returns the regions fakeAWS received RDS calls for, sorted.
func scannedRegions(f *fakeAWS) []string {
	calls := f.calls("rds")
	res := make([]string, 0, len(calls))
	for _, c := range calls {
		res = append(res, c.region)
	}
	sort.Strings(res)
	return res
}

// setupRoleDiscovery points DiscoverRDS at a fakeAWS with ambient credentials and no region
// configured, so a subtest that sets AWS_REGION or AWS_DEFAULT_REGION is the only region source.
// The shared config files are disabled too, so a developer's ~/.aws/config region cannot leak in.
func setupRoleDiscovery(t *testing.T) *fakeAWS {
	t.Helper()

	fake := newFakeAWS(t)
	t.Setenv("AWS_ENDPOINT_URL_STS", fake.URL)
	t.Setenv("AWS_ENDPOINT_URL_RDS", fake.URL)
	// The role is assumed with the server's ambient credentials.
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "ambient-secret")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_CONFIG_FILE", "/nonexistent")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/nonexistent")
	return fake
}
