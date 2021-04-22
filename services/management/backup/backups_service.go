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

package backup

import (
	"context"

	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// BackupsService represents backups API.
type BackupsService struct {
	db          *reform.DB
	jobsService jobsService
}

// NewBackupsService creates new backups API service.
func NewBackupsService(db *reform.DB, jobsService jobsService) *BackupsService {
	return &BackupsService{
		db:          db,
		jobsService: jobsService,
	}
}

// StartBackup starts on-demand backup.
func (s *BackupsService) StartBackup(ctx context.Context, req *backupv1beta1.StartBackupRequest) (*backupv1beta1.StartBackupResponse, error) {
	var err error
	var artifact *models.Artifact
	var location *models.BackupLocation
	var svc *models.Service
	var job *models.JobResult
	var config models.DBConfig

	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		svc, err = models.FindServiceByID(tx.Querier, req.ServiceId)
		if err != nil {
			return err
		}

		location, err = models.FindBackupLocationByID(tx.Querier, req.LocationId)
		if err != nil {
			return err
		}

		var dataModel models.DataModel
		switch svc.ServiceType {
		case models.MySQLServiceType:
			dataModel = models.PhysicalDataModel
		case models.PostgreSQLServiceType:
			fallthrough
		case models.MongoDBServiceType:
			fallthrough
		case models.ProxySQLServiceType:
			fallthrough
		case models.HAProxyServiceType:
			fallthrough
		case models.ExternalServiceType:
			return status.Errorf(codes.Unimplemented, "unimplemented service: %s", svc.ServiceType)
		default:
			return status.Errorf(codes.Unknown, "unknown service: %s", svc.ServiceType)
		}

		artifact, err = models.CreateArtifact(tx.Querier, models.CreateArtifactParams{
			Name:       req.Name,
			Vendor:     string(svc.ServiceType),
			LocationID: location.ID,
			ServiceID:  svc.ServiceID,
			DataModel:  dataModel,
			Status:     models.PendingBackupStatus,
		})
		if err != nil {
			return err
		}

		job, config, err = s.prepareBackupJob(svc, artifact.ID, models.MySQLBackupJob)
		if err != nil {
			return err
		}
		return nil
	})

	if errTX != nil {
		return nil, errTX
	}

	locationConfig := models.BackupLocationConfig{
		PMMServerConfig: location.PMMServerConfig,
		PMMClientConfig: location.PMMClientConfig,
		S3Config:        location.S3Config,
	}

	switch svc.ServiceType {
	case models.MySQLServiceType:
		err = s.jobsService.StartMySQLBackupJob(job.ID, job.PMMAgentID, 0, req.Name, config, locationConfig)
	case models.PostgreSQLServiceType:
		fallthrough
	case models.MongoDBServiceType:
		fallthrough
	case models.ProxySQLServiceType:
		fallthrough
	case models.HAProxyServiceType:
		fallthrough
	case models.ExternalServiceType:
		return nil, status.Errorf(codes.Unimplemented, "unimplemented service: %s", svc.ServiceType)
	default:
		return nil, status.Errorf(codes.Unknown, "unknown service: %s", svc.ServiceType)
	}
	if err != nil {
		return nil, err
	}

	return &backupv1beta1.StartBackupResponse{
		ArtifactId: artifact.ID,
	}, nil
}

func (s *BackupsService) prepareBackupJob(service *models.Service, artifactID string, jobType models.JobType) (*models.JobResult, models.DBConfig, error) {
	var res *models.JobResult
	var dbConfig models.DBConfig
	txErr := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		dbConfig, err = models.FindDBConfigForService(tx.Querier, service.ServiceID)
		if err != nil {
			return err
		}

		pmmAgents, err := models.FindPMMAgentsForService(tx.Querier, service.ServiceID)
		if err != nil {
			return err
		}

		if len(pmmAgents) == 0 {
			return errors.Errorf("pmmAgent not found for service")
		}

		res, err = models.CreateJobResult(tx.Querier, pmmAgents[0].AgentID, jobType, &models.JobResultData{
			MySQLBackup: &models.MySQLBackupJobResult{
				ArtifactID: artifactID,
			},
		})
		return err
	})

	if txErr != nil {
		return nil, models.DBConfig{}, txErr
	}
	return res, dbConfig, nil
}

// Check interfaces.
var (
	_ backupv1beta1.BackupsServer = (*BackupsService)(nil)
)
