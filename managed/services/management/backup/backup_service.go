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

package backup

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	backupv1 "github.com/percona/pmm/api/backup/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/services/backup"
	"github.com/percona/pmm/managed/services/scheduler"
)

// BackupService represents backups API.
type BackupService struct { //nolint:revive
	db                   *reform.DB
	backupService        backupService
	compatibilityService compatibilityService
	scheduleService      scheduleService
	removalSVC           removalService
	pbmPITRService       pbmPITRService
	l                    *logrus.Entry

	backupv1.UnimplementedBackupServiceServer
}

const (
	maxRetriesAttempts = 10
	maxRetryInterval   = 8 * time.Hour
)

var (
	folderRe = regexp.MustCompile(`^[\.:\/\w-]*$`) // Dots, colons, slashes, letters, digits, underscores, dashes.
	nameRe   = regexp.MustCompile(`^[\.:\w-]*$`)   // Dots, colons, letters, digits, underscores, dashes.
)

// NewBackupsService creates new backups API service.
func NewBackupsService(
	db *reform.DB,
	backupService backupService,
	cSvc compatibilityService,
	scheduleService scheduleService,
	removalSVC removalService,
	pbmPITRService pbmPITRService,
) *BackupService {
	return &BackupService{
		l:                    logrus.WithField("component", "management/backup"),
		db:                   db,
		backupService:        backupService,
		compatibilityService: cSvc,
		scheduleService:      scheduleService,
		removalSVC:           removalSVC,
		pbmPITRService:       pbmPITRService,
	}
}

// StartBackup starts on-demand backup.
func (s *BackupService) StartBackup(ctx context.Context, req *backupv1.StartBackupRequest) (*backupv1.StartBackupResponse, error) {
	if req.Retries > maxRetriesAttempts {
		return nil, status.Errorf(codes.InvalidArgument, "Exceeded max retries %d.", maxRetriesAttempts)
	}

	if req.RetryInterval.AsDuration() > maxRetryInterval {
		return nil, status.Errorf(codes.InvalidArgument, "Exceeded max retry interval %s.", maxRetryInterval)
	}

	if err := isFolderSafe(req.Folder); err != nil {
		return nil, err
	}

	if err := isNameSafe(req.Name); err != nil {
		return nil, err
	}

	dataModel, err := convertModelToBackupModel(req.DataModel)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid data model: %s", req.DataModel.String())
	}

	compression, err := convertCompressionToBackupCompression(req.Compression)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid compression: %s", req.Compression.String())
	}

	svc, err := models.FindServiceByID(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
	}

	if err := compression.ValidateForServiceType(svc.ServiceType); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Compression validation failed: %v", err)
	}

	if svc.ServiceType == models.MongoDBServiceType {
		if svc.Cluster == "" {
			return nil, status.Errorf(codes.FailedPrecondition, "Service %s must be a member of a cluster", svc.ServiceName)
		}
	}

	artifactID, err := s.backupService.PerformBackup(ctx, backup.PerformBackupParams{
		ServiceID:     req.ServiceId,
		LocationID:    req.LocationId,
		Name:          req.Name,
		DataModel:     dataModel,
		Mode:          models.Snapshot,
		Retries:       req.Retries,
		RetryInterval: req.RetryInterval.AsDuration(),
		Folder:        req.Folder,
		Compression:   compression,
	})
	if err != nil {
		return nil, convertError(err)
	}

	return &backupv1.StartBackupResponse{
		ArtifactId: artifactID,
	}, nil
}

// ScheduleBackup add new backup task to scheduler.
func (s *BackupService) ScheduleBackup(ctx context.Context, req *backupv1.ScheduleBackupRequest) (*backupv1.ScheduleBackupResponse, error) {
	var id string

	if req.Retries > maxRetriesAttempts {
		return nil, status.Errorf(codes.InvalidArgument, "Exceeded max retries %d.", maxRetriesAttempts)
	}

	if req.RetryInterval.AsDuration() > maxRetryInterval {
		return nil, status.Errorf(codes.InvalidArgument, "Exceeded max retry interval %s.", maxRetryInterval)
	}

	if err := isFolderSafe(req.Folder); err != nil {
		return nil, err
	}

	if err := isNameSafe(req.Name); err != nil {
		return nil, err
	}

	mode, err := convertBackupModeToModel(req.Mode)
	if err != nil {
		return nil, err
	}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		svc, err := models.FindServiceByID(tx.Querier, req.ServiceId)
		if err != nil {
			return err
		}

		_, err = models.FindBackupLocationByID(tx.Querier, req.LocationId)
		if err != nil {
			return err
		}

		dataModel, err := convertModelToBackupModel(req.DataModel)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "Invalid data model: %s", req.DataModel.String())
		}

		compression, err := convertCompressionToBackupCompression(req.Compression)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "Invalid compression: %s", req.Compression.String())
		}

		if err := compression.ValidateForServiceType(svc.ServiceType); err != nil {
			return status.Errorf(codes.InvalidArgument, "Compression validation failed: %v", err)
		}

		backupParams := &scheduler.BackupTaskParams{
			ServiceID:     svc.ServiceID,
			ClusterName:   svc.Cluster,
			LocationID:    req.LocationId,
			Name:          req.Name,
			Description:   req.Description,
			DataModel:     dataModel,
			Mode:          mode,
			Retention:     req.Retention,
			Retries:       req.Retries,
			RetryInterval: req.RetryInterval.AsDuration(),
			Folder:        req.Folder,
			Compression:   compression,
		}

		var task scheduler.Task
		switch svc.ServiceType {
		case models.MySQLServiceType:
			task, err = scheduler.NewMySQLBackupTask(backupParams)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Can't create mySQL backup task: %v", err)
			}
		case models.MongoDBServiceType:
			if svc.Cluster == "" {
				return status.Errorf(codes.FailedPrecondition, "Service %s must be a member of a cluster", svc.ServiceName)
			}

			task, err = scheduler.NewMongoDBBackupTask(backupParams)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Can't create mongoDB backup task: %v", err)
			}
		case models.PostgreSQLServiceType,
			models.ProxySQLServiceType,
			models.HAProxyServiceType,
			models.ExternalServiceType:
			return status.Errorf(codes.Unimplemented, "Unimplemented service: %s.", svc.ServiceType)
		default:
			return status.Errorf(codes.Unknown, "Unknown service: %s.", svc.ServiceType)
		}

		t := req.StartTime.AsTime()
		if t.Unix() == 0 {
			t = time.Time{}
		}

		scheduledTask, err := s.scheduleService.Add(task, scheduler.AddParams{
			CronExpression: req.CronExpression,
			Disabled:       !req.Enabled,
			StartAt:        t,
		})
		if err != nil {
			return convertError(err)
		}

		id = scheduledTask.ID
		return nil
	})
	if errTx != nil {
		return nil, errTx
	}
	return &backupv1.ScheduleBackupResponse{ScheduledBackupId: id}, nil
}

// ListScheduledBackups lists all tasks related to a backup.
func (s *BackupService) ListScheduledBackups(ctx context.Context, req *backupv1.ListScheduledBackupsRequest) (*backupv1.ListScheduledBackupsResponse, error) { //nolint:revive,lll
	tasks, err := models.FindScheduledTasks(s.db.Querier, models.ScheduledTasksFilter{
		Types: []models.ScheduledTaskType{
			models.ScheduledMySQLBackupTask,
			models.ScheduledMongoDBBackupTask,
		},
	})
	if err != nil {
		return nil, err
	}

	locationIDs := make([]string, 0, len(tasks))
	serviceIDs := make([]string, 0, len(tasks))
	for _, t := range tasks {
		var serviceID string
		var locationID string
		switch t.Type {
		case models.ScheduledMySQLBackupTask:
			serviceID = t.Data.MySQLBackupTask.ServiceID
			locationID = t.Data.MySQLBackupTask.LocationID
		case models.ScheduledMongoDBBackupTask:
			serviceID = t.Data.MongoDBBackupTask.ServiceID
			locationID = t.Data.MongoDBBackupTask.LocationID
		default:
			continue
		}
		serviceIDs = append(serviceIDs, serviceID)
		locationIDs = append(locationIDs, locationID)
	}
	locations, err := models.FindBackupLocationsByIDs(s.db.Querier, locationIDs)
	if err != nil {
		return nil, err
	}

	svcs, err := models.FindServicesByIDs(s.db.Querier, serviceIDs)
	if err != nil {
		return nil, err
	}

	scheduledBackups := make([]*backupv1.ScheduledBackup, 0, len(tasks))
	for _, task := range tasks {
		scheduledBackup, err := convertTaskToScheduledBackup(task, svcs, locations)
		if err != nil {
			s.l.WithError(err).Warnf("convert task to scheduledBackup")
			continue
		}
		scheduledBackups = append(scheduledBackups, scheduledBackup)
	}

	return &backupv1.ListScheduledBackupsResponse{
		ScheduledBackups: scheduledBackups,
	}, nil
}

// ChangeScheduledBackup changes existing scheduled backup task.
func (s *BackupService) ChangeScheduledBackup(ctx context.Context, req *backupv1.ChangeScheduledBackupRequest) (*backupv1.ChangeScheduledBackupResponse, error) {
	var disablePITR bool
	var serviceID string

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		scheduledTask, err := models.FindScheduledTaskByID(tx.Querier, req.ScheduledBackupId)
		if err != nil {
			return convertError(err)
		}

		var data *models.CommonBackupTaskData
		switch scheduledTask.Type {
		case models.ScheduledMySQLBackupTask:
			data = &scheduledTask.Data.MySQLBackupTask.CommonBackupTaskData
		case models.ScheduledMongoDBBackupTask:
			data = &scheduledTask.Data.MongoDBBackupTask.CommonBackupTaskData
		default:
			return status.Errorf(codes.InvalidArgument, "Unknown type: %s", scheduledTask.Type)
		}

		if req.Name != nil {
			data.Name = *req.Name
		}
		if req.Description != nil {
			data.Description = *req.Description
		}
		if req.Retention != nil {
			data.Retention = *req.Retention
		}
		if req.Retries != nil {
			if *req.Retries > maxRetriesAttempts {
				return status.Errorf(codes.InvalidArgument, "exceeded max retries %d", maxRetriesAttempts)
			}
			data.Retries = *req.Retries
		}
		if req.RetryInterval != nil {
			if req.RetryInterval.AsDuration() > maxRetryInterval {
				return status.Errorf(codes.InvalidArgument, "exceeded max retry interval %s", maxRetryInterval)
			}
			data.RetryInterval = req.RetryInterval.AsDuration()
		}

		serviceID = data.ServiceID
		params := models.ChangeScheduledTaskParams{
			Data:           scheduledTask.Data,
			CronExpression: req.CronExpression,
		}

		if req.Enabled != nil {
			params.Disable = pointer.ToBool(!*req.Enabled)
			if scheduledTask.Type == models.ScheduledMongoDBBackupTask && !*req.Enabled {
				disablePITR = data.Mode == models.PITR
			}
		}

		err = s.scheduleService.Update(req.ScheduledBackupId, params)

		return convertError(err)
	})
	if errTx != nil {
		return nil, errTx
	}

	if disablePITR {
		if err := s.backupService.SwitchMongoPITR(ctx, serviceID, false); err != nil {
			s.l.WithError(err).Error("failed to disable PITR")
		}
	}

	return &backupv1.ChangeScheduledBackupResponse{}, nil
}

// RemoveScheduledBackup stops and removes existing scheduled backup task.
func (s *BackupService) RemoveScheduledBackup(ctx context.Context, req *backupv1.RemoveScheduledBackupRequest) (*backupv1.RemoveScheduledBackupResponse, error) {
	task, err := models.FindScheduledTaskByID(s.db.Querier, req.ScheduledBackupId)
	if err != nil {
		return nil, err
	}

	var disablePITR bool
	switch task.Type {
	case models.ScheduledMySQLBackupTask:
		// nothing
	case models.ScheduledMongoDBBackupTask:
		// for enabled incremental mongoDB backups switch-off PITR
		disablePITR = task.Data.MongoDBBackupTask.Mode == models.PITR && !task.Disabled
	default:
		return nil, errors.Errorf("non-backup task: %s", task.Type)
	}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		artifacts, err := models.FindArtifacts(tx.Querier, models.ArtifactFilters{
			ScheduleID: req.ScheduledBackupId,
		})
		if err != nil {
			return err
		}

		for _, artifact := range artifacts {
			_, err := models.UpdateArtifact(tx.Querier, artifact.ID, models.UpdateArtifactParams{
				ScheduleID: pointer.ToString(""),
			})
			if err != nil {
				return err
			}
		}

		return s.scheduleService.Remove(req.ScheduledBackupId)
	})
	if errTx != nil {
		return nil, errTx
	}

	if disablePITR {
		if err = s.backupService.SwitchMongoPITR(ctx, task.Data.MongoDBBackupTask.ServiceID, false); err != nil {
			s.l.WithError(err).Error("failed to disable PITR")
		}
	}

	return &backupv1.RemoveScheduledBackupResponse{}, nil
}

// GetLogs returns logs from the underlying tools for a backup/restore job.
func (s *BackupService) GetLogs(_ context.Context, req *backupv1.GetLogsRequest) (*backupv1.GetLogsResponse, error) {
	jobsFilter := models.JobsFilter{
		Types: []models.JobType{
			models.MySQLBackupJob,
			models.MongoDBBackupJob,
			models.MongoDBRestoreBackupJob,
		},
	}

	jobsFilter.ArtifactID = req.ArtifactId

	jobs, err := models.FindJobs(s.db.Querier, jobsFilter)
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, status.Error(codes.NotFound, "Job related to artifact was not found.")
	}
	if len(jobs) > 1 {
		s.l.Warn("provided ID appears in more than one job")
	}

	filter := models.JobLogsFilter{
		JobID:  jobs[0].ID,
		Offset: int(req.Offset),
	}
	if req.Limit > 0 {
		filter.Limit = pointer.ToInt(int(req.Limit))
	}

	jobLogs, err := models.FindJobLogs(s.db.Querier, filter)
	if err != nil {
		return nil, err
	}

	res := &backupv1.GetLogsResponse{
		Logs: make([]*backupv1.LogChunk, 0, len(jobLogs)),
	}
	for _, log := range jobLogs {
		if log.LastChunk {
			res.End = true
			break
		}
		res.Logs = append(res.Logs, &backupv1.LogChunk{
			ChunkId: uint32(log.ChunkID),
			Data:    log.Data,
		})
	}

	return res, nil
}

// ListArtifactCompatibleServices lists compatible service for restoring given artifact.
func (s *BackupService) ListArtifactCompatibleServices(
	ctx context.Context,
	req *backupv1.ListArtifactCompatibleServicesRequest,
) (*backupv1.ListArtifactCompatibleServicesResponse, error) {
	compatibleServices, err := s.compatibilityService.FindArtifactCompatibleServices(ctx, req.ArtifactId)
	switch {
	case err == nil:
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, err.Error())
	default:
		return nil, err
	}

	res := &backupv1.ListArtifactCompatibleServicesResponse{}
	for _, service := range compatibleServices {
		apiService, err := services.ToAPIService(service)
		if err != nil {
			return nil, err
		}

		switch s := apiService.(type) {
		case *inventoryv1.MySQLService:
			res.Mysql = append(res.Mysql, s)
		case *inventoryv1.MongoDBService:
			res.Mongodb = append(res.Mongodb, s)
		case *inventoryv1.PostgreSQLService,
			*inventoryv1.ProxySQLService,
			*inventoryv1.HAProxyService,
			*inventoryv1.ExternalService:
			return nil, status.Errorf(codes.Unimplemented, "unimplemented service type %T", service)
		default:
			return nil, status.Errorf(codes.Internal, "unhandled inventory service type %T", service)
		}
	}

	return res, nil
}

// ListArtifacts returns a list of all artifacts.
func (s *BackupService) ListArtifacts(context.Context, *backupv1.ListArtifactsRequest) (*backupv1.ListArtifactsResponse, error) {
	q := s.db.Querier

	artifacts, err := models.FindArtifacts(q, models.ArtifactFilters{})
	if err != nil {
		return nil, err
	}

	locationIDs := make([]string, 0, len(artifacts))
	for _, b := range artifacts {
		locationIDs = append(locationIDs, b.LocationID)
	}
	locations, err := models.FindBackupLocationsByIDs(q, locationIDs)
	if err != nil {
		return nil, err
	}

	serviceIDs := make([]string, 0, len(artifacts))
	for _, a := range artifacts {
		if a.ServiceID != "" {
			serviceIDs = append(serviceIDs, a.ServiceID)
		}
	}

	services, err := models.FindServicesByIDs(q, serviceIDs)
	if err != nil {
		return nil, err
	}

	artifactsResponse := make([]*backupv1.Artifact, 0, len(artifacts))
	for _, b := range artifacts {
		convertedArtifact, err := convertArtifact(b, services, locations)
		if err != nil {
			return nil, err
		}
		artifactsResponse = append(artifactsResponse, convertedArtifact)
	}
	return &backupv1.ListArtifactsResponse{
		Artifacts: artifactsResponse,
	}, nil
}

// Enabled returns if service is enabled and can be used.
func (s *BackupService) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.IsBackupManagementEnabled()
}

// DeleteArtifact deletes specified artifact and its files.
func (s *BackupService) DeleteArtifact(ctx context.Context, req *backupv1.DeleteArtifactRequest) (*backupv1.DeleteArtifactResponse, error) { //nolint:revive
	artifact, err := models.FindArtifactByID(s.db.Querier, req.ArtifactId)
	if err != nil {
		return nil, err
	}

	location, err := models.FindBackupLocationByID(s.db.Querier, artifact.LocationID)
	if err != nil {
		return nil, err
	}

	storage := backup.GetStorageForLocation(location)

	if err := s.removalSVC.DeleteArtifact(storage, req.ArtifactId, req.RemoveFiles); err != nil {
		return nil, err
	}
	return &backupv1.DeleteArtifactResponse{}, nil
}

// ListPitrTimeranges lists available PITR timelines/time-ranges (for MongoDB).
func (s *BackupService) ListPitrTimeranges(ctx context.Context, req *backupv1.ListPitrTimerangesRequest) (*backupv1.ListPitrTimerangesResponse, error) {
	var artifact *models.Artifact
	var err error

	artifact, err = models.FindArtifactByID(s.db.Querier, req.ArtifactId)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "Artifact with ID '%s' not found.", req.ArtifactId)
		}
		return nil, err
	}

	if artifact.Mode != models.PITR {
		return nil, status.Errorf(codes.FailedPrecondition, "Artifact is not a PITR artifact.")
	}

	if artifact.IsShardedCluster {
		return nil, status.Errorf(codes.FailedPrecondition, "Getting PITR timeranges is not supported for sharded cluster artifacts.")
	}

	location, err := models.FindBackupLocationByID(s.db.Querier, artifact.LocationID)
	if err != nil {
		return nil, err
	}

	storage := backup.GetStorageForLocation(location)

	timelines, err := s.pbmPITRService.ListPITRTimeranges(ctx, storage, location, artifact)
	if err != nil {
		return nil, err
	}
	result := make([]*backupv1.PitrTimerange, 0, len(timelines))
	for _, tl := range timelines {
		result = append(result, &backupv1.PitrTimerange{
			StartTimestamp: timestamppb.New(time.Unix(int64(tl.Start), 0)),
			EndTimestamp:   timestamppb.New(time.Unix(int64(tl.End), 0)),
		})
	}
	return &backupv1.ListPitrTimerangesResponse{
		Timeranges: result,
	}, nil
}

// ListServiceCompression returns available compression methods for a service.
func (s *BackupService) ListServiceCompression(ctx context.Context, req *backupv1.ListServiceCompressionRequest) (*backupv1.ListServiceCompressionResponse, error) {
	svc, err := models.FindServiceByID(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
	}

	supportedCompressions := models.GetSupportedCompressions(svc.ServiceType)
	if supportedCompressions == nil {
		return nil, status.Errorf(codes.Unimplemented, "backup compression is not yet supported for service type: %s", svc.ServiceType)
	}

	compressionMethods := make([]backupv1.BackupCompression, 0, len(supportedCompressions))
	for _, compression := range supportedCompressions {
		protoCompression, err := convertBackupCompression(compression)
		if err != nil {
			return nil, err
		}
		compressionMethods = append(compressionMethods, protoCompression)
	}

	return &backupv1.ListServiceCompressionResponse{
		CompressionMethods: compressionMethods,
	}, nil
}

func convertTaskToScheduledBackup(task *models.ScheduledTask,
	services map[string]*models.Service,
	locationModels map[string]*models.BackupLocation,
) (*backupv1.ScheduledBackup, error) {
	scheduledBackup := &backupv1.ScheduledBackup{
		ScheduledBackupId: task.ID,
		CronExpression:    task.CronExpression,
		Enabled:           !task.Disabled,
	}

	if !task.LastRun.IsZero() {
		scheduledBackup.LastRun = timestamppb.New(task.LastRun)
	}

	if !task.NextRun.IsZero() {
		scheduledBackup.NextRun = timestamppb.New(task.NextRun)
	}

	if !task.StartAt.IsZero() {
		scheduledBackup.StartTime = timestamppb.New(task.StartAt)
	}

	var commonBackupData models.CommonBackupTaskData
	switch task.Type {
	case models.ScheduledMySQLBackupTask:
		commonBackupData = task.Data.MySQLBackupTask.CommonBackupTaskData
	case models.ScheduledMongoDBBackupTask:
		commonBackupData = task.Data.MongoDBBackupTask.CommonBackupTaskData
	default:
		return nil, errors.Errorf("unknown task type: %s", task.Type)
	}

	scheduledBackup.ServiceId = commonBackupData.ServiceID
	scheduledBackup.LocationId = commonBackupData.LocationID
	scheduledBackup.Name = commonBackupData.Name
	scheduledBackup.Description = commonBackupData.Description
	scheduledBackup.Retention = commonBackupData.Retention
	scheduledBackup.Retries = commonBackupData.Retries
	scheduledBackup.Folder = commonBackupData.Folder

	var err error
	if scheduledBackup.DataModel, err = convertDataModel(commonBackupData.DataModel); err != nil {
		return nil, err
	}

	if scheduledBackup.Mode, err = convertModelToBackupMode(commonBackupData.Mode); err != nil {
		return nil, err
	}

	if scheduledBackup.Compression, err = convertBackupCompression(commonBackupData.Compression); err != nil {
		return nil, err
	}

	if commonBackupData.RetryInterval > 0 {
		scheduledBackup.RetryInterval = durationpb.New(commonBackupData.RetryInterval)
	}

	scheduledBackup.ServiceName = services[scheduledBackup.ServiceId].ServiceName
	scheduledBackup.Vendor = string(services[scheduledBackup.ServiceId].ServiceType)
	scheduledBackup.LocationName = locationModels[scheduledBackup.LocationId].Name

	return scheduledBackup, nil
}

func convertBackupModeToModel(mode backupv1.BackupMode) (models.BackupMode, error) {
	switch mode {
	case backupv1.BackupMode_BACKUP_MODE_SNAPSHOT:
		return models.Snapshot, nil
	case backupv1.BackupMode_BACKUP_MODE_INCREMENTAL:
		return models.Incremental, nil
	case backupv1.BackupMode_BACKUP_MODE_PITR:
		return models.PITR, nil
	case backupv1.BackupMode_BACKUP_MODE_UNSPECIFIED:
		return "", status.Errorf(codes.InvalidArgument, "invalid backup mode: %s", mode.String())
	default:
		return "", status.Errorf(codes.InvalidArgument, "Unknown backup mode: %s", mode.String())
	}
}

func convertModelToBackupMode(mode models.BackupMode) (backupv1.BackupMode, error) {
	switch mode {
	case models.Snapshot:
		return backupv1.BackupMode_BACKUP_MODE_SNAPSHOT, nil
	case models.Incremental:
		return backupv1.BackupMode_BACKUP_MODE_INCREMENTAL, nil
	case models.PITR:
		return backupv1.BackupMode_BACKUP_MODE_PITR, nil
	default:
		return 0, errors.Errorf("unknown backup mode: %s", mode)
	}
}

func convertModelToBackupModel(dataModel backupv1.DataModel) (models.DataModel, error) {
	switch dataModel {
	case backupv1.DataModel_DATA_MODEL_LOGICAL:
		return models.LogicalDataModel, nil
	case backupv1.DataModel_DATA_MODEL_PHYSICAL:
		return models.PhysicalDataModel, nil
	default:
		return "", errors.Errorf("unknown backup mode: %s", dataModel)
	}
}

func convertCompressionToBackupCompression(compression backupv1.BackupCompression) (models.BackupCompression, error) {
	switch compression {
	case backupv1.BackupCompression_BACKUP_COMPRESSION_QUICKLZ:
		return models.QuickLZ, nil
	case backupv1.BackupCompression_BACKUP_COMPRESSION_ZSTD:
		return models.ZSTD, nil
	case backupv1.BackupCompression_BACKUP_COMPRESSION_LZ4:
		return models.LZ4, nil
	case backupv1.BackupCompression_BACKUP_COMPRESSION_S2:
		return models.S2, nil
	case backupv1.BackupCompression_BACKUP_COMPRESSION_GZIP:
		return models.GZIP, nil
	case backupv1.BackupCompression_BACKUP_COMPRESSION_SNAPPY:
		return models.Snappy, nil
	case backupv1.BackupCompression_BACKUP_COMPRESSION_PGZIP:
		return models.PGZIP, nil
	case backupv1.BackupCompression_BACKUP_COMPRESSION_NONE:
		return models.None, nil
	case backupv1.BackupCompression_BACKUP_COMPRESSION_DEFAULT:
		return models.Default, nil
	default:
		return "", errors.Errorf("unknown backup compression: %s", compression)
	}
}

// convertError converts error from Go to API.
func convertError(e error) error {
	if e == nil {
		return nil
	}

	var unsupportedAgentErr models.AgentNotSupportedError
	if errors.As(e, &unsupportedAgentErr) {
		return status.Error(codes.FailedPrecondition, e.Error())
	}

	var code backupv1.ErrorCode
	switch {
	case errors.Is(e, backup.ErrXtrabackupNotInstalled):
		code = backupv1.ErrorCode_ERROR_CODE_XTRABACKUP_NOT_INSTALLED
	case errors.Is(e, backup.ErrInvalidXtrabackup):
		code = backupv1.ErrorCode_ERROR_CODE_INVALID_XTRABACKUP
	case errors.Is(e, backup.ErrIncompatibleXtrabackup):
		code = backupv1.ErrorCode_ERROR_CODE_INCOMPATIBLE_XTRABACKUP
	case errors.Is(e, backup.ErrIncompatibleTargetMySQL):
		code = backupv1.ErrorCode_ERROR_CODE_INCOMPATIBLE_TARGET_MYSQL
	case errors.Is(e, backup.ErrIncompatibleTargetMongoDB):
		code = backupv1.ErrorCode_ERROR_CODE_INCOMPATIBLE_TARGET_MONGODB
	case errors.Is(e, backup.ErrTimestampOutOfRange):
		return status.Error(codes.OutOfRange, e.Error())
	case errors.Is(e, models.ErrNotFound):
		return status.Error(codes.NotFound, e.Error())
	case errors.Is(e, models.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, e.Error())
	case errors.Is(e, backup.ErrAnotherOperationInProgress),
		errors.Is(e, backup.ErrArtifactNotReady),
		errors.Is(e, backup.ErrIncompatiblePBM),
		errors.Is(e, backup.ErrIncompatibleLocationType),
		errors.Is(e, backup.ErrIncompatibleService),
		errors.Is(e, backup.ErrIncompatibleArtifactMode),
		errors.Is(e, services.ErrLocationFolderPairAlreadyUsed):
		return status.Error(codes.FailedPrecondition, e.Error())

	default:
		return e
	}

	st, err := status.New(codes.FailedPrecondition, e.Error()).WithDetails(&backupv1.Error{
		Code: code,
	})
	if err != nil {
		return fmt.Errorf("failed to construct status error: %w, original error: %w", err, e)
	}

	return st.Err()
}

// isFolderSafe checks if specified path is safe against traversal attacks.
func isFolderSafe(path string) error {
	if path == "" {
		return nil
	}

	canonical := filepath.Clean(path)
	if canonical != path {
		return status.Errorf(codes.InvalidArgument, "Specified folder in non-canonical format, canonical would be: %q.", canonical)
	}

	if strings.HasPrefix(path, "/") {
		return status.Error(codes.InvalidArgument, "Folder should be a relative path (shouldn't contain leading slashes).")
	}

	if path == ".." || strings.HasPrefix(path, "../") {
		return status.Error(codes.InvalidArgument, "Specified folder refers to a parent directory.")
	}

	if !folderRe.MatchString(path) {
		return status.Error(codes.InvalidArgument, "Folder name can contain only dots, colons, slashes, letters, digits, underscores and dashes.")
	}

	return nil
}

func isNameSafe(name string) error {
	if !nameRe.MatchString(name) {
		return status.Error(codes.InvalidArgument, "Backup name can contain only dots, colons, letters, digits, underscores and dashes.")
	}
	return nil
}

func convertDataModel(model models.DataModel) (backupv1.DataModel, error) {
	switch model {
	case models.PhysicalDataModel:
		return backupv1.DataModel_DATA_MODEL_PHYSICAL, nil
	case models.LogicalDataModel:
		return backupv1.DataModel_DATA_MODEL_LOGICAL, nil
	default:
		return 0, errors.Errorf("unknown data model: %s", model)
	}
}

func convertBackupStatus(status models.BackupStatus) (backupv1.BackupStatus, error) {
	switch status {
	case models.PendingBackupStatus:
		return backupv1.BackupStatus_BACKUP_STATUS_PENDING, nil
	case models.InProgressBackupStatus:
		return backupv1.BackupStatus_BACKUP_STATUS_IN_PROGRESS, nil
	case models.PausedBackupStatus:
		return backupv1.BackupStatus_BACKUP_STATUS_PAUSED, nil
	case models.SuccessBackupStatus:
		return backupv1.BackupStatus_BACKUP_STATUS_SUCCESS, nil
	case models.ErrorBackupStatus:
		return backupv1.BackupStatus_BACKUP_STATUS_ERROR, nil
	case models.DeletingBackupStatus:
		return backupv1.BackupStatus_BACKUP_STATUS_DELETING, nil
	case models.FailedToDeleteBackupStatus:
		return backupv1.BackupStatus_BACKUP_STATUS_FAILED_TO_DELETE, nil
	case models.CleanupInProgressStatus:
		return backupv1.BackupStatus_BACKUP_STATUS_CLEANUP_IN_PROGRESS, nil
	default:
		return 0, errors.Errorf("invalid status '%s'", status)
	}
}

func convertBackupCompression(compression models.BackupCompression) (backupv1.BackupCompression, error) {
	switch compression {
	case models.QuickLZ:
		return backupv1.BackupCompression_BACKUP_COMPRESSION_QUICKLZ, nil
	case models.ZSTD:
		return backupv1.BackupCompression_BACKUP_COMPRESSION_ZSTD, nil
	case models.LZ4:
		return backupv1.BackupCompression_BACKUP_COMPRESSION_LZ4, nil
	case models.S2:
		return backupv1.BackupCompression_BACKUP_COMPRESSION_S2, nil
	case models.GZIP:
		return backupv1.BackupCompression_BACKUP_COMPRESSION_GZIP, nil
	case models.Snappy:
		return backupv1.BackupCompression_BACKUP_COMPRESSION_SNAPPY, nil
	case models.PGZIP:
		return backupv1.BackupCompression_BACKUP_COMPRESSION_PGZIP, nil
	case models.Default:
		return backupv1.BackupCompression_BACKUP_COMPRESSION_DEFAULT, nil
	case models.None:
		return backupv1.BackupCompression_BACKUP_COMPRESSION_NONE, nil
	default:
		return 0, errors.Errorf("invalid compression '%s'", compression)
	}
}

func convertArtifact(
	a *models.Artifact,
	services map[string]*models.Service,
	locationModels map[string]*models.BackupLocation,
) (*backupv1.Artifact, error) {
	createdAt := timestamppb.New(a.CreatedAt)
	if err := createdAt.CheckValid(); err != nil {
		return nil, errors.Wrap(err, "failed to convert timestamp")
	}

	l, ok := locationModels[a.LocationID]
	if !ok {
		return nil, errors.Errorf(
			"failed to convert artifact with id '%s': no location id '%s' in the map", a.ID, a.LocationID)
	}

	var serviceName string
	if s, ok := services[a.ServiceID]; ok {
		serviceName = s.ServiceName
	}

	dataModel, err := convertDataModel(a.DataModel)
	if err != nil {
		return nil, errors.Wrapf(err, "artifact id '%s'", a.ID)
	}

	backupStatus, err := convertBackupStatus(a.Status)
	if err != nil {
		return nil, errors.Wrapf(err, "artifact id '%s'", a.ID)
	}

	backupMode, err := convertModelToBackupMode(a.Mode)
	if err != nil {
		return nil, errors.Wrapf(err, "artifact id '%s'", a.ID)
	}

	compression, err := convertBackupCompression(a.Compression)
	if err != nil {
		return nil, errors.Wrapf(err, "artifact id '%s'", a.ID)
	}

	return &backupv1.Artifact{
		ArtifactId:       a.ID,
		Name:             a.Name,
		Vendor:           a.Vendor,
		LocationId:       a.LocationID,
		LocationName:     l.Name,
		ServiceId:        a.ServiceID,
		ServiceName:      serviceName,
		DataModel:        dataModel,
		Mode:             backupMode,
		Status:           backupStatus,
		CreatedAt:        createdAt,
		IsShardedCluster: a.IsShardedCluster,
		Folder:           a.Folder,
		MetadataList:     artifactMetadataListToProto(a),
		Compression:      compression,
	}, nil
}

// artifactMetadataListToProto returns artifact metadata list in protobuf format.
func artifactMetadataListToProto(artifact *models.Artifact) []*backupv1.Metadata {
	res := make([]*backupv1.Metadata, len(artifact.MetadataList))
	for i, metadata := range artifact.MetadataList {
		res[i] = &backupv1.Metadata{}
		res[i].FileList = make([]*backupv1.File, len(metadata.FileList))

		for j, file := range metadata.FileList {
			res[i].FileList[j] = &backupv1.File{
				Name:        file.Name,
				IsDirectory: file.IsDirectory,
			}
		}

		if metadata.RestoreTo != nil {
			res[i].RestoreTo = timestamppb.New(*metadata.RestoreTo)
		}

		if metadata.BackupToolData != nil {
			if metadata.BackupToolData.PbmMetadata != nil {
				res[i].BackupToolMetadata = &backupv1.Metadata_PbmMetadata{
					PbmMetadata: &backupv1.PbmMetadata{Name: metadata.BackupToolData.PbmMetadata.Name},
				}
			}
		}
	}
	return res
}

// Check interfaces.
var (
	_ backupv1.BackupServiceServer = (*BackupService)(nil)
)
