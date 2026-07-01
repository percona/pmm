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

package backupscript

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	backupv1 "github.com/percona/pmm/api/backup/v1"
	"github.com/percona/pmm/managed/models"
)

// manifestMarker prefixes the one-line JSON result the payload prints on stdout;
// pmm-managed ingests it to catalog the backup (size, location, versions).
const manifestMarker = "PMM_MANIFEST_JSON:"

// Service implements backupv1.BackupScriptServiceServer. It renders the
// XtraBackup payload config, dispatches a Nomad batch job to the target node,
// and catalogs the resulting runs.
type Service struct {
	db    *reform.DB
	nomad *nomadClient
	l     *logrus.Entry

	backupv1.UnimplementedBackupScriptServiceServer
}

// New creates a new BackupScript service.
func New(db *reform.DB, l *logrus.Entry) *Service {
	return &Service{
		db:    db,
		nomad: newNomadClient(),
		l:     l,
	}
}

// CreateConfig persists a new versioned config and its rendered YAML.
func (s *Service) CreateConfig(_ context.Context, req *backupv1.CreateBackupScriptConfigRequest) (*backupv1.CreateBackupScriptConfigResponse, error) {
	rendered, err := RenderConfig(ConfigParams{
		Alias:                req.ServiceId,
		Host:                 "localhost",
		Port:                 3306,
		BackupDir:            req.BackupDir,
		Compress:             req.Compress,
		CompressionAlgorithm: req.CompressionAlgorithm,
		Copies:               int32(req.Copies), //nolint:gosec
		ReplicaInfo:          req.ReplicaInfo,
		XtrabackupBinary:     req.XtrabackupBinary,
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to render config: %v", err)
	}

	var cfg *models.BackupScriptConfig
	err = s.db.InTransaction(func(tx *reform.TX) error {
		var e error
		cfg, e = models.CreateBackupScriptConfig(tx.Querier, models.CreateBackupScriptConfigParams{
			Name:                 req.Name,
			ServiceID:            req.ServiceId,
			NodeName:             req.NodeName,
			BackupDir:            req.BackupDir,
			Compress:             req.Compress,
			CompressionAlgorithm: req.CompressionAlgorithm,
			Copies:               int32(req.Copies), //nolint:gosec
			ReplicaInfo:          req.ReplicaInfo,
			XtrabackupBinary:     req.XtrabackupBinary,
			RenderedYAML:         rendered,
		})
		return e
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create config: %v", err)
	}
	return &backupv1.CreateBackupScriptConfigResponse{Config: convertConfig(cfg)}, nil
}

// ListConfigs returns all stored configs.
func (s *Service) ListConfigs(_ context.Context, _ *backupv1.ListBackupScriptConfigsRequest) (*backupv1.ListBackupScriptConfigsResponse, error) {
	configs, err := models.FindBackupScriptConfigs(s.db.Querier)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list configs: %v", err)
	}
	res := &backupv1.ListBackupScriptConfigsResponse{Configs: make([]*backupv1.BackupScriptConfig, len(configs))}
	for i, c := range configs {
		res.Configs[i] = convertConfig(c)
	}
	return res, nil
}

// GetConfig returns a single config.
func (s *Service) GetConfig(_ context.Context, req *backupv1.GetBackupScriptConfigRequest) (*backupv1.GetBackupScriptConfigResponse, error) {
	cfg, err := models.FindBackupScriptConfigByID(s.db.Querier, req.ConfigId)
	if err != nil {
		return nil, convertNotFound(err, "config")
	}
	return &backupv1.GetBackupScriptConfigResponse{Config: convertConfig(cfg)}, nil
}

// DeleteConfig removes a config.
func (s *Service) DeleteConfig(_ context.Context, req *backupv1.DeleteBackupScriptConfigRequest) (*backupv1.DeleteBackupScriptConfigResponse, error) {
	err := s.db.InTransaction(func(tx *reform.TX) error {
		return models.RemoveBackupScriptConfig(tx.Querier, req.ConfigId)
	})
	if err != nil {
		return nil, convertNotFound(err, "config")
	}
	return &backupv1.DeleteBackupScriptConfigResponse{}, nil
}

// RunNow renders the job for a config, registers it with Nomad, catalogs the run
// and starts an asynchronous monitor.
func (s *Service) RunNow(ctx context.Context, req *backupv1.RunNowRequest) (*backupv1.RunNowResponse, error) {
	cfg, err := models.FindBackupScriptConfigByID(s.db.Querier, req.ConfigId)
	if err != nil {
		return nil, convertNotFound(err, "config")
	}

	runID := uuid.New().String()
	job := buildBackupJob(buildJobParams{
		RunID:         runID,
		ServiceID:     cfg.ServiceID,
		ConfigVersion: cfg.ConfigVersion,
		NodeName:      cfg.NodeName,
		RenderedYAML:  cfg.RenderedYAML,
	})

	if _, err := s.nomad.registerJob(ctx, job); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to dispatch backup: %v", err)
	}

	njobID := jobID(runID)
	err = s.db.InTransaction(func(tx *reform.TX) error {
		_, e := models.CreateBackupScriptRun(tx.Querier, models.CreateBackupScriptRunParams{
			RunID:      runID,
			ConfigID:   cfg.ID,
			ServiceID:  cfg.ServiceID,
			NodeName:   cfg.NodeName,
			NomadJobID: njobID,
			StartedAt:  time.Now(),
		})
		return e
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record run: %v", err)
	}

	go s.monitorRun(runID, njobID)

	s.l.WithFields(logrus.Fields{"run_id": runID, "job_id": njobID, "node": cfg.NodeName}).Info("dispatched script backup")
	return &backupv1.RunNowResponse{RunId: runID, NomadJobId: njobID}, nil
}

// ListRuns returns the run catalog.
func (s *Service) ListRuns(_ context.Context, _ *backupv1.ListBackupScriptRunsRequest) (*backupv1.ListBackupScriptRunsResponse, error) {
	runs, err := models.FindBackupScriptRuns(s.db.Querier)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list runs: %v", err)
	}
	res := &backupv1.ListBackupScriptRunsResponse{Runs: make([]*backupv1.BackupScriptRun, len(runs))}
	for i, r := range runs {
		res.Runs[i] = convertRun(r)
	}
	return res, nil
}

// GetRun returns a single run.
func (s *Service) GetRun(_ context.Context, req *backupv1.GetBackupScriptRunRequest) (*backupv1.GetBackupScriptRunResponse, error) {
	run, err := models.FindBackupScriptRunByID(s.db.Querier, req.RunId)
	if err != nil {
		return nil, convertNotFound(err, "run")
	}
	return &backupv1.GetBackupScriptRunResponse{Run: convertRun(run)}, nil
}

// GetRunLogs streams the live allocation stdout for a run.
func (s *Service) GetRunLogs(ctx context.Context, req *backupv1.GetBackupScriptRunLogsRequest) (*backupv1.GetBackupScriptRunLogsResponse, error) {
	run, err := models.FindBackupScriptRunByID(s.db.Querier, req.RunId)
	if err != nil {
		return nil, convertNotFound(err, "run")
	}
	logs, err := s.latestAllocLogs(ctx, run.NomadJobID, "stdout")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read live logs: %v", err)
	}
	return &backupv1.GetBackupScriptRunLogsResponse{Logs: logs}, nil
}

// latestAllocLogs fetches stdout/stderr from the newest allocation of a job.
func (s *Service) latestAllocLogs(ctx context.Context, njobID, logType string) (string, error) {
	allocs, err := s.nomad.jobAllocations(ctx, njobID)
	if err != nil {
		return "", err
	}
	if len(allocs) == 0 {
		return "", nil
	}
	return s.nomad.allocLogs(ctx, allocs[len(allocs)-1].ID, backupTaskName, logType)
}

// monitorRun polls the job's allocation until it reaches a terminal state, then
// ingests the result manifest from stdout and updates the run row.
func (s *Service) monitorRun(runID, njobID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.setRunError(runID, "backup monitor timed out")
			return
		case <-ticker.C:
		}

		allocs, err := s.nomad.jobAllocations(ctx, njobID)
		if err != nil || len(allocs) == 0 {
			continue
		}
		alloc := allocs[len(allocs)-1]

		switch alloc.ClientStatus {
		case "running":
			s.setRunStatus(runID, models.ScriptBackupRunning)
		case "complete":
			s.finishRun(ctx, runID, alloc.ID, models.ScriptBackupSuccess, "")
			return
		case "failed", "lost":
			s.finishRun(ctx, runID, alloc.ID, models.ScriptBackupError, "allocation "+alloc.ClientStatus)
			return
		}
	}
}

// finishRun reads the manifest from stdout and persists the terminal state.
func (s *Service) finishRun(ctx context.Context, runID, allocID string, st models.ScriptBackupStatus, errMsg string) {
	stdout, _ := s.nomad.allocLogs(ctx, allocID, backupTaskName, "stdout")
	manifest := parseManifest(stdout)

	run, err := models.FindBackupScriptRunByID(s.db.Querier, runID)
	if err != nil {
		s.l.WithField("run_id", runID).WithError(err).Warn("run vanished before finishing")
		return
	}

	now := time.Now()
	run.Status = st
	run.Error = errMsg
	run.FinishedAt = &now
	if manifest != nil {
		run.Manifest = manifest
		run.BackupDir = manifest.BackupDir
		run.SizeBytes = manifest.SizeBytes
		if manifest.Status == "error" || manifest.Status == "failed" {
			run.Status = models.ScriptBackupError
			if run.Error == "" {
				run.Error = "payload reported failure"
			}
		}
	}
	if err := models.UpdateBackupScriptRun(s.db.Querier, run); err != nil {
		s.l.WithField("run_id", runID).WithError(err).Error("failed to persist run result")
	}
}

func (s *Service) setRunStatus(runID string, st models.ScriptBackupStatus) {
	run, err := models.FindBackupScriptRunByID(s.db.Querier, runID)
	if err != nil || run.Status == st {
		return
	}
	run.Status = st
	if err := models.UpdateBackupScriptRun(s.db.Querier, run); err != nil {
		s.l.WithField("run_id", runID).WithError(err).Warn("failed to update run status")
	}
}

func (s *Service) setRunError(runID, msg string) {
	run, err := models.FindBackupScriptRunByID(s.db.Querier, runID)
	if err != nil {
		return
	}
	now := time.Now()
	run.Status = models.ScriptBackupError
	run.Error = msg
	run.FinishedAt = &now
	_ = models.UpdateBackupScriptRun(s.db.Querier, run) //nolint:errcheck
}

// parseManifest extracts the payload result manifest from stdout.
func parseManifest(stdout string) *models.ScriptRunManifest {
	for _, line := range strings.Split(stdout, "\n") {
		idx := strings.Index(line, manifestMarker)
		if idx < 0 {
			continue
		}
		raw := strings.TrimSpace(line[idx+len(manifestMarker):])
		var m models.ScriptRunManifest
		if err := json.Unmarshal([]byte(raw), &m); err == nil {
			return &m
		}
	}
	return nil
}

func convertConfig(c *models.BackupScriptConfig) *backupv1.BackupScriptConfig {
	return &backupv1.BackupScriptConfig{
		ConfigId:             c.ID,
		Name:                 c.Name,
		ServiceId:            c.ServiceID,
		NodeName:             c.NodeName,
		BackupDir:            c.BackupDir,
		Compress:             c.Compress,
		CompressionAlgorithm: c.CompressionAlgorithm,
		Copies:               uint32(c.Copies), //nolint:gosec
		ReplicaInfo:          c.ReplicaInfo,
		XtrabackupBinary:     c.XtrabackupBinary,
		RenderedYaml:         c.RenderedYAML,
		ConfigVersion:        uint32(c.ConfigVersion), //nolint:gosec
		CreatedAt:            timestamppb.New(c.CreatedAt),
		UpdatedAt:            timestamppb.New(c.UpdatedAt),
	}
}

func convertRun(r *models.BackupScriptRun) *backupv1.BackupScriptRun {
	out := &backupv1.BackupScriptRun{
		RunId:      r.ID,
		ConfigId:   r.ConfigID,
		ServiceId:  r.ServiceID,
		NodeName:   r.NodeName,
		NomadJobId: r.NomadJobID,
		Status:     convertStatus(r.Status),
		BackupDir:  r.BackupDir,
		SizeBytes:  r.SizeBytes,
		Error:      r.Error,
		StartedAt:  timestamppb.New(r.StartedAt),
	}
	if r.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(*r.FinishedAt)
	}
	return out
}

func convertStatus(st models.ScriptBackupStatus) backupv1.ScriptBackupStatus {
	switch st {
	case models.ScriptBackupPending:
		return backupv1.ScriptBackupStatus_SCRIPT_BACKUP_STATUS_PENDING
	case models.ScriptBackupRunning:
		return backupv1.ScriptBackupStatus_SCRIPT_BACKUP_STATUS_RUNNING
	case models.ScriptBackupSuccess:
		return backupv1.ScriptBackupStatus_SCRIPT_BACKUP_STATUS_SUCCESS
	case models.ScriptBackupError:
		return backupv1.ScriptBackupStatus_SCRIPT_BACKUP_STATUS_ERROR
	default:
		return backupv1.ScriptBackupStatus_SCRIPT_BACKUP_STATUS_UNSPECIFIED
	}
}

func convertNotFound(err error, kind string) error {
	if errors.Is(err, models.ErrNotFound) {
		return status.Errorf(codes.NotFound, "backup script %s not found", kind)
	}
	return status.Errorf(codes.Internal, "%v", err)
}
