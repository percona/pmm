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
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	backuppb "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/services/backup"
	"github.com/percona/pmm/managed/services/scheduler"
)

// BackupsService represents backups API.
type BackupsService struct {
	db                   *reform.DB
	backupService        backupService
	compatibilityService compatibilityService
	scheduleService      scheduleService
	l                    *logrus.Entry

	backuppb.UnimplementedBackupsServer
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
) *BackupsService {
	return &BackupsService{
		l:                    logrus.WithField("component", "management/backup/backups"),
		db:                   db,
		backupService:        backupService,
		compatibilityService: cSvc,
		scheduleService:      scheduleService,
	}
}

// StartBackup starts on-demand backup.
func (s *BackupsService) StartBackup(ctx context.Context, req *backuppb.StartBackupRequest) (*backuppb.StartBackupResponse, error) {
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

	svc, err := models.FindServiceByID(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
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
	})
	if err != nil {
		return nil, convertError(err)
	}

	return &backuppb.StartBackupResponse{
		ArtifactId: artifactID,
	}, nil
}

// RestoreBackup starts restore backup job.
func (s *BackupsService) RestoreBackup(
	ctx context.Context,
	req *backuppb.RestoreBackupRequest,
) (*backuppb.RestoreBackupResponse, error) {
	// Disable all related scheduled backups before restoring
	tasks, err := models.FindScheduledTasks(s.db.Querier, models.ScheduledTasksFilter{ServiceID: req.ServiceId})
	if err != nil {
		return nil, err
	}

	for _, t := range tasks {
		if _, err := s.ChangeScheduledBackup(ctx, &backuppb.ChangeScheduledBackupRequest{
			ScheduledBackupId: t.ID,
			Enabled:           &wrapperspb.BoolValue{Value: false},
		}); err != nil {
			return nil, err
		}
	}

	id, err := s.backupService.RestoreBackup(ctx, req.ServiceId, req.ArtifactId, req.PitrTimestamp.AsTime())
	if err != nil {
		return nil, convertError(err)
	}

	return &backuppb.RestoreBackupResponse{
		RestoreId: id,
	}, nil
}

// ScheduleBackup add new backup task to scheduler.
func (s *BackupsService) ScheduleBackup(ctx context.Context, req *backuppb.ScheduleBackupRequest) (*backuppb.ScheduleBackupResponse, error) {
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
	return &backuppb.ScheduleBackupResponse{ScheduledBackupId: id}, nil
}

// ListScheduledBackups lists all tasks related to a backup.
func (s *BackupsService) ListScheduledBackups(ctx context.Context, req *backuppb.ListScheduledBackupsRequest) (*backuppb.ListScheduledBackupsResponse, error) { //nolint:revive,lll
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

	scheduledBackups := make([]*backuppb.ScheduledBackup, 0, len(tasks))
	for _, task := range tasks {
		scheduledBackup, err := convertTaskToScheduledBackup(task, svcs, locations)
		if err != nil {
			s.l.WithError(err).Warnf("convert task to scheduledBackup")
			continue
		}
		scheduledBackups = append(scheduledBackups, scheduledBackup)
	}

	return &backuppb.ListScheduledBackupsResponse{
		ScheduledBackups: scheduledBackups,
	}, nil
}

// ChangeScheduledBackup changes existing scheduled backup task.
func (s *BackupsService) ChangeScheduledBackup(ctx context.Context, req *backuppb.ChangeScheduledBackupRequest) (*backuppb.ChangeScheduledBackupResponse, error) {
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
			data.Name = req.Name.Value
		}
		if req.Description != nil {
			data.Description = req.Description.Value
		}
		if req.Retention != nil {
			data.Retention = req.Retention.Value
		}
		if req.Retries != nil {
			if req.Retries.Value > maxRetriesAttempts {
				return status.Errorf(codes.InvalidArgument, "exceeded max retries %d", maxRetriesAttempts)
			}
			data.Retries = req.Retries.Value
		}
		if req.RetryInterval != nil {
			if req.RetryInterval.AsDuration() > maxRetryInterval {
				return status.Errorf(codes.InvalidArgument, "exceeded max retry interval %s", maxRetryInterval)
			}
			data.RetryInterval = req.RetryInterval.AsDuration()
		}

		serviceID = data.ServiceID
		params := models.ChangeScheduledTaskParams{
			Data: scheduledTask.Data,
		}

		if req.Enabled != nil {
			params.Disable = pointer.ToBool(!req.Enabled.Value)
			if scheduledTask.Type == models.ScheduledMongoDBBackupTask && !req.Enabled.Value {
				disablePITR = data.Mode == models.PITR
			}
		}

		if req.CronExpression != nil {
			params.CronExpression = pointer.ToString(req.CronExpression.Value)
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

	return &backuppb.ChangeScheduledBackupResponse{}, nil
}

// RemoveScheduledBackup stops and removes existing scheduled backup task.
func (s *BackupsService) RemoveScheduledBackup(ctx context.Context, req *backuppb.RemoveScheduledBackupRequest) (*backuppb.RemoveScheduledBackupResponse, error) {
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

	errTx := s.db.InTransaction(func(tx *reform.TX) error {
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

	return &backuppb.RemoveScheduledBackupResponse{}, nil
}

// GetLogs returns logs from the underlying tools for a backup/restore job.
func (s *BackupsService) GetLogs(_ context.Context, req *backuppb.GetLogsRequest) (*backuppb.GetLogsResponse, error) {
	jobsFilter := models.JobsFilter{
		Types: []models.JobType{
			models.MySQLBackupJob,
			models.MongoDBBackupJob,
			models.MongoDBRestoreBackupJob,
		},
	}
	if req.ArtifactId != "" && req.RestoreId != "" {
		return nil, status.Error(codes.InvalidArgument, "Only one of artifact ID or restore ID is required")
	}

	switch {
	case req.ArtifactId != "":
		jobsFilter.ArtifactID = req.ArtifactId
	case req.RestoreId != "":
		jobsFilter.RestoreID = req.RestoreId
	default:
		return nil, status.Error(codes.InvalidArgument, "One of artifact ID or restore ID is required")
	}

	jobs, err := models.FindJobs(s.db.Querier, jobsFilter)
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, status.Error(codes.NotFound, "Job related to artifact was not found.")
	}
	if len(jobs) > 1 {
		s.l.Warn("provided ID appear in more than one job")
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

	res := &backuppb.GetLogsResponse{
		Logs: make([]*backuppb.LogChunk, 0, len(jobLogs)),
	}
	for _, log := range jobLogs {
		if log.LastChunk {
			res.End = true
			break
		}
		res.Logs = append(res.Logs, &backuppb.LogChunk{
			ChunkId: uint32(log.ChunkID),
			Data:    log.Data,
		})
	}

	return res, nil
}

// ListArtifactCompatibleServices lists compatible service for restoring given artifact.
func (s *BackupsService) ListArtifactCompatibleServices(
	ctx context.Context,
	req *backuppb.ListArtifactCompatibleServicesRequest,
) (*backuppb.ListArtifactCompatibleServicesResponse, error) {
	compatibleServices, err := s.compatibilityService.FindArtifactCompatibleServices(ctx, req.ArtifactId)
	switch {
	case err == nil:
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, err.Error())
	default:
		return nil, err
	}

	res := &backuppb.ListArtifactCompatibleServicesResponse{}
	for _, service := range compatibleServices {
		apiService, err := services.ToAPIService(service)
		if err != nil {
			return nil, err
		}

		switch s := apiService.(type) {
		case *inventorypb.MySQLService:
			res.Mysql = append(res.Mysql, s)
		case *inventorypb.MongoDBService:
			res.Mongodb = append(res.Mongodb, s)
		case *inventorypb.PostgreSQLService,
			*inventorypb.ProxySQLService,
			*inventorypb.HAProxyService,
			*inventorypb.ExternalService:
			return nil, status.Errorf(codes.Unimplemented, "unimplemented service type %T", service)
		default:
			return nil, status.Errorf(codes.Internal, "unhandled inventory service type %T", service)
		}
	}

	return res, nil
}

func convertTaskToScheduledBackup(task *models.ScheduledTask,
	services map[string]*models.Service,
	locationModels map[string]*models.BackupLocation,
) (*backuppb.ScheduledBackup, error) {
	scheduledBackup := &backuppb.ScheduledBackup{
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

	if commonBackupData.RetryInterval > 0 {
		scheduledBackup.RetryInterval = durationpb.New(commonBackupData.RetryInterval)
	}

	scheduledBackup.ServiceName = services[scheduledBackup.ServiceId].ServiceName
	scheduledBackup.Vendor = string(services[scheduledBackup.ServiceId].ServiceType)
	scheduledBackup.LocationName = locationModels[scheduledBackup.LocationId].Name

	return scheduledBackup, nil
}

func convertBackupModeToModel(mode backuppb.BackupMode) (models.BackupMode, error) {
	switch mode {
	case backuppb.BackupMode_SNAPSHOT:
		return models.Snapshot, nil
	case backuppb.BackupMode_INCREMENTAL:
		return models.Incremental, nil
	case backuppb.BackupMode_PITR:
		return models.PITR, nil
	case backuppb.BackupMode_BACKUP_MODE_INVALID:
		return "", status.Errorf(codes.InvalidArgument, "invalid backup mode: %s", mode.String())
	default:
		return "", status.Errorf(codes.InvalidArgument, "Unknown backup mode: %s", mode.String())
	}
}

func convertModelToBackupMode(mode models.BackupMode) (backuppb.BackupMode, error) {
	switch mode {
	case models.Snapshot:
		return backuppb.BackupMode_SNAPSHOT, nil
	case models.Incremental:
		return backuppb.BackupMode_INCREMENTAL, nil
	case models.PITR:
		return backuppb.BackupMode_PITR, nil
	default:
		return 0, errors.Errorf("unknown backup mode: %s", mode)
	}
}

func convertModelToBackupModel(dataModel backuppb.DataModel) (models.DataModel, error) {
	switch dataModel {
	case backuppb.DataModel_LOGICAL:
		return models.LogicalDataModel, nil
	case backuppb.DataModel_PHYSICAL:
		return models.PhysicalDataModel, nil
	default:
		return "", errors.Errorf("unknown backup mode: %s", dataModel)
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

	var code backuppb.ErrorCode
	switch {
	case errors.Is(e, backup.ErrXtrabackupNotInstalled):
		code = backuppb.ErrorCode_ERROR_CODE_XTRABACKUP_NOT_INSTALLED
	case errors.Is(e, backup.ErrInvalidXtrabackup):
		code = backuppb.ErrorCode_ERROR_CODE_INVALID_XTRABACKUP
	case errors.Is(e, backup.ErrIncompatibleXtrabackup):
		code = backuppb.ErrorCode_ERROR_CODE_INCOMPATIBLE_XTRABACKUP
	case errors.Is(e, backup.ErrIncompatibleTargetMySQL):
		code = backuppb.ErrorCode_ERROR_CODE_INCOMPATIBLE_TARGET_MYSQL
	case errors.Is(e, backup.ErrIncompatibleTargetMongoDB):
		code = backuppb.ErrorCode_ERROR_CODE_INCOMPATIBLE_TARGET_MONGODB
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

	st, err := status.New(codes.FailedPrecondition, e.Error()).WithDetails(&backuppb.Error{
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

// Check interfaces.
var (
	_ backuppb.BackupsServer = (*BackupsService)(nil)
)
