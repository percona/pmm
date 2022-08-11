package agents

import (
	"fmt"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func setup(t *testing.T, d *reform.DB, serviceType models.ServiceType, serviceName string, agentVersion string) *models.Agent {
	t.Helper()
	require.Contains(t, []models.ServiceType{models.MySQLServiceType, models.MongoDBServiceType}, serviceType)

	node, err := models.CreateNode(d.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node-" + serviceName,
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(d.Querier, node.NodeID, nil)
	require.NoError(t, err)

	pmmAgent.Version = pointer.ToString(agentVersion)
	err = d.Update(pmmAgent)
	require.NoError(t, err)

	var service *models.Service
	service, err = models.AddNewService(d.Querier, serviceType, &models.AddDBMSServiceParams{
		ServiceName: serviceName,
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(60000),
	})
	require.NoError(t, err)

	agentType := models.MySQLdExporterType
	if serviceType == models.MongoDBServiceType {
		agentType = models.MongoDBExporterType
	}

	agent, err := models.CreateAgent(d.Querier, agentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service.ServiceID,
		Username:   "user",
		Password:   "password",
	})
	require.NoError(t, err)
	return agent
}

func TestStartMongoDBRestoreBackupJob(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	registry := NewRegistry(db)
	j := NewJobsService(db, registry, nil)
	testJobTimeout := 1 * time.Second

	t.Run("check pmm-agent version", func(t *testing.T) {
		locationConfig := models.BackupLocationConfig{
			S3Config: &models.S3LocationConfig{
				Endpoint:     "https://example.com/",
				AccessKey:    "access_key",
				SecretKey:    "secret_key",
				BucketName:   "example_bucket",
				BucketRegion: "us-east-2",
			},
		}

		pmmAgentVersions := map[string]*models.Agent{
			"2.18.1": setup(t, db, models.MongoDBServiceType, "mongo-test-1", "2.18.1"),
			"2.19.1": setup(t, db, models.MongoDBServiceType, "mongo-test-2", "2.19.1"),
			"2.30.1": setup(t, db, models.MongoDBServiceType, "mongo-test-3", "2.30.1"),
			"2.31.1": setup(t, db, models.MongoDBServiceType, "mongo-test-4", "2.31.1"),
		}

		tests := []struct {
			name         string
			agentVersion string
			dataModel    models.DataModel
			errMsg       string
		}{
			{
				name:         "physical backup on invalid agent",
				agentVersion: "2.30.1",
				dataModel:    models.PhysicalDataModel,
				errMsg:       "mongodb physical restore is not supported on pmm-agent",
			},
			{
				name:         "physical backup on valid agent",
				agentVersion: "2.31.1",
				dataModel:    models.PhysicalDataModel,
				errMsg:       fmt.Sprintf("pmm-agent with ID \"%s\" is not currently connected", *pmmAgentVersions["2.31.1"].PMMAgentID),
			},
			{
				name:         "logical backup on invalid agent",
				agentVersion: "2.18.1",
				dataModel:    models.LogicalDataModel,
				errMsg:       "mongodb logical restore is not supported on pmm-agent",
			},
			{
				name:         "logical backup on valid agent",
				agentVersion: "2.19.1",
				dataModel:    models.LogicalDataModel,
				errMsg:       fmt.Sprintf("pmm-agent with ID \"%s\" is not currently connected", *pmmAgentVersions["2.19.1"].PMMAgentID),
			},
		}
		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				agent := pmmAgentVersions[tc.agentVersion]
				svc, err := models.FindServiceByID(db.Querier, *agent.ServiceID)
				require.NoError(t, err)

				err = j.StartMongoDBRestoreBackupJob(t.Name(), *agent.PMMAgentID, testJobTimeout, "restore",
					agent.DBConfig(svc), tc.dataModel, &locationConfig)
				require.ErrorContains(t, err, tc.errMsg)
			})
		}
	})
}
