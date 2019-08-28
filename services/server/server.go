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

// Package server implements pmm-managed Server API.
package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// Server represents service for checking PMM Server status and changing settings.
type Server struct {
	db                 *reform.DB
	prometheus         prometheusService
	supervisord        supervisordService
	l                  *logrus.Entry
	pmmUpdateAuthFile  string
	pmmUpdateAuthFileM sync.Mutex

	envRW                sync.RWMutex
	envDisableUpdates    bool
	envDisableTelemetry  bool
	envMetricsResolution time.Duration
	envDataRetention     time.Duration
}

type pmmUpdateAuth struct {
	AuthToken string `json:"auth_token"`
}

// NewServer returns new server for Server service.
func NewServer(db *reform.DB, prometheus prometheusService, supervisord supervisordService) (*Server, error) {
	path := os.TempDir()
	if _, err := os.Stat(path); err != nil {
		return nil, errors.WithStack(err)
	}
	path = filepath.Join(path, "pmm-update.json")

	s := &Server{
		db:                db,
		prometheus:        prometheus,
		supervisord:       supervisord,
		l:                 logrus.WithField("component", "server"),
		pmmUpdateAuthFile: path,
	}
	return s, nil
}

// UpdateSettingsFromEnv updates settings in the database with environment variables values.
// It returns only validation or database errors; invalid environment variables are logged and skipped.
func (s *Server) UpdateSettingsFromEnv(env []string) error {
	s.envRW.Lock()
	defer s.envRW.Unlock()

	return s.db.InTransaction(func(tx *reform.TX) error {
		settings, err := models.GetSettings(tx.Querier)
		if err != nil {
			return err
		}

		for _, e := range env {
			p := strings.SplitN(e, "=", 2)
			if len(p) != 2 {
				s.l.Warnf("Failed to parse environment variable %q.", e)
				continue
			}

			k, v := strings.ToUpper(p[0]), strings.ToLower(p[1])
			var err error
			var value interface{}
			switch k {
			case "DISABLE_UPDATES":
				var b bool
				b, err = strconv.ParseBool(v)
				if err == nil {
					value = b
					s.envDisableUpdates = b
				}

			case "DISABLE_TELEMETRY":
				var b bool
				b, err = strconv.ParseBool(v)
				if err == nil {
					value = b
					s.envDisableTelemetry = b
					settings.Telemetry.Disabled = b
				}

			case "METRICS_RESOLUTION":
				var d time.Duration
				d, err = time.ParseDuration(v)
				if err == nil {
					value = d
					s.envMetricsResolution = d
					settings.MetricsResolutions.HR = d
				}

			case "DATA_RETENTION":
				var d time.Duration
				d, err = time.ParseDuration(v)
				if err == nil {
					value = d
					s.envDataRetention = d
					settings.DataRetention = d
				}

			default:
				s.l.Tracef("Skipping %q.", e)
				continue
			}

			if err == nil {
				s.l.Infof("Environment variable %q parsed: %v.", e, value)
			} else {
				s.l.Warnf("Failed to parse environment variable %q: %s.", e, err)
			}
		}

		return models.SaveSettings(tx.Querier, settings)
	})
}

// Version returns PMM Server version.
func (s *Server) Version(ctx context.Context, req *serverpb.VersionRequest) (*serverpb.VersionResponse, error) {
	// for API testing of authentication, panic handling, etc.
	if req.Dummy != "" {
		switch {
		case strings.HasPrefix(req.Dummy, "panic-"):
			switch req.Dummy {
			case "panic-error":
				panic(errors.New("panic-error"))
			case "panic-fmterror":
				panic(fmt.Errorf("panic-fmterror"))
			default:
				panic(req.Dummy)
			}

		case strings.HasPrefix(req.Dummy, "grpccode-"):
			code, err := strconv.Atoi(strings.TrimPrefix(req.Dummy, "grpccode-"))
			if err != nil {
				return nil, err
			}
			grpcCode := codes.Code(code)
			return nil, status.Errorf(grpcCode, "gRPC code %d (%s)", grpcCode, grpcCode)
		}
	}

	res := &serverpb.VersionResponse{
		// always return something in this field:
		// it is used by PMM 1.x's pmm-client for compatibility checking
		Version: version.Version,

		Managed: &serverpb.VersionInfo{
			Version:     version.Version,
			FullVersion: version.FullCommit,
		},
	}
	if t, err := version.Time(); err == nil {
		ts, _ := ptypes.TimestampProto(t)
		res.Managed.Timestamp = ts
	}

	if v := s.supervisord.InstalledPMMVersion(); v != nil {
		res.Version = v.Version
		res.Server = &serverpb.VersionInfo{
			Version:     v.Version,
			FullVersion: v.FullVersion,
		}
		if v.BuildTime != nil {
			res.Server.Timestamp, _ = ptypes.TimestampProto(*v.BuildTime)
		}
	}

	return res, nil
}

// Readiness returns an error when some PMM Server component is not ready yet or is being restarted.
// It can be used as for Docker health check or Kubernetes readiness probe.
func (s *Server) Readiness(ctx context.Context, req *serverpb.ReadinessRequest) (*serverpb.ReadinessResponse, error) {
	// TODO https://jira.percona.com/browse/PMM-1962

	if err := s.prometheus.Check(ctx); err != nil {
		return nil, err
	}
	return &serverpb.ReadinessResponse{}, nil
}

// CheckUpdates checks PMM Server updates availability.
func (s *Server) CheckUpdates(ctx context.Context, req *serverpb.CheckUpdatesRequest) (*serverpb.CheckUpdatesResponse, error) {
	if req.Force {
		if err := s.supervisord.ForceCheckUpdates(); err != nil {
			return nil, err
		}
	}

	v, lastCheck := s.supervisord.LastCheckUpdatesResult()
	if v == nil {
		return nil, status.Error(codes.Unavailable, "failed to check for updates")
	}

	res := &serverpb.CheckUpdatesResponse{
		Installed: &serverpb.VersionInfo{
			Version:     v.Installed.Version,
			FullVersion: v.Installed.FullVersion,
		},
		Latest: &serverpb.VersionInfo{
			Version:     v.Latest.Version,
			FullVersion: v.Latest.FullVersion,
		},
		UpdateAvailable: v.UpdateAvailable,
		LatestNewsUrl:   "", // TODO https://jira.percona.com/browse/PMM-4444
	}
	res.LastCheck, _ = ptypes.TimestampProto(lastCheck)
	if v.Installed.BuildTime != nil {
		t := v.Installed.BuildTime.UTC().Truncate(24 * time.Hour) // return only date
		res.Installed.Timestamp, _ = ptypes.TimestampProto(t)
	}
	if v.Latest.BuildTime != nil {
		t := v.Latest.BuildTime.UTC().Truncate(24 * time.Hour) // return only date
		res.Latest.Timestamp, _ = ptypes.TimestampProto(t)
	}
	return res, nil
}

// StartUpdate starts PMM Server update.
func (s *Server) StartUpdate(ctx context.Context, req *serverpb.StartUpdateRequest) (*serverpb.StartUpdateResponse, error) {
	s.envRW.RLock()
	updatesDisabled := s.envDisableUpdates
	s.envRW.RUnlock()

	if updatesDisabled {
		return nil, status.Error(codes.FailedPrecondition, "Updates are disabled via DISABLE_UPDATES environment variable.")
	}

	offset, err := s.supervisord.StartUpdate()
	if err != nil {
		return nil, err
	}

	authToken := uuid.New().String()
	if err = s.writeUpdateAuthToken(authToken); err != nil {
		return nil, err
	}

	return &serverpb.StartUpdateResponse{
		AuthToken: authToken,
		LogOffset: offset,
	}, nil
}

// UpdateStatus returns PMM Server update status.
func (s *Server) UpdateStatus(ctx context.Context, req *serverpb.UpdateStatusRequest) (*serverpb.UpdateStatusResponse, error) {
	token, err := s.readUpdateAuthToken()
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(req.AuthToken), []byte(token)) == 0 {
		return nil, status.Error(codes.PermissionDenied, "Invalid authentication token.")
	}

	// wait up to 30 seconds for new log lines
	var lines []string
	var newOffset uint32
	var done bool
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for ctx.Err() == nil {
		lines, newOffset, err = s.supervisord.UpdateLog(req.LogOffset)
		if err != nil {
			s.l.Warn(err)
		}
		done = !s.supervisord.UpdateRunning()

		if len(lines) != 0 || done {
			break
		}

		time.Sleep(200 * time.Millisecond)
	}

	return &serverpb.UpdateStatusResponse{
		LogLines:  lines,
		LogOffset: newOffset,
		Done:      done,
	}, nil
}

// writeUpdateAuthToken writes authentication token for getting update status and logs to the file.
//
// We can't rely on Grafana for authentication or on PostgreSQL for storage as their configuration
// is being changed during update.
func (s *Server) writeUpdateAuthToken(token string) error {
	s.pmmUpdateAuthFileM.Lock()
	defer s.pmmUpdateAuthFileM.Unlock()

	a := &pmmUpdateAuth{
		AuthToken: token,
	}
	f, err := os.OpenFile(s.pmmUpdateAuthFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600|os.ModeExclusive)
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			s.l.Error(err)
		}
	}()

	return errors.WithStack(json.NewEncoder(f).Encode(a))
}

// readUpdateAuthToken reads authentication token for getting update status and logs from the file.
func (s *Server) readUpdateAuthToken() (string, error) {
	s.pmmUpdateAuthFileM.Lock()
	defer s.pmmUpdateAuthFileM.Unlock()

	f, err := os.OpenFile(s.pmmUpdateAuthFile, os.O_RDONLY, os.ModeExclusive)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			s.l.Error(err)
		}
	}()

	var a pmmUpdateAuth
	err = json.NewDecoder(f).Decode(&a)
	return a.AuthToken, errors.WithStack(err)
}

func convertSettings(s *models.Settings) *serverpb.Settings {
	return &serverpb.Settings{
		TelemetryEnabled: !s.Telemetry.Disabled,
		MetricsResolutions: &serverpb.MetricsResolutions{
			Hr: ptypes.DurationProto(s.MetricsResolutions.HR),
			Mr: ptypes.DurationProto(s.MetricsResolutions.MR),
			Lr: ptypes.DurationProto(s.MetricsResolutions.LR),
		},
		DataRetention: ptypes.DurationProto(s.DataRetention),
	}
}

// GetSettings returns current PMM Server settings.
func (s *Server) GetSettings(ctx context.Context, req *serverpb.GetSettingsRequest) (*serverpb.GetSettingsResponse, error) {
	s.envRW.RLock()
	defer s.envRW.RUnlock()

	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}
	res := &serverpb.GetSettingsResponse{
		Settings: convertSettings(settings),
	}
	res.Settings.UpdatesDisabled = s.envDisableUpdates
	return res, nil
}

// ChangeSettings changes PMM Server settings.
func (s *Server) ChangeSettings(ctx context.Context, req *serverpb.ChangeSettingsRequest) (*serverpb.ChangeSettingsResponse, error) {
	s.envRW.RLock()
	defer s.envRW.RUnlock()

	if req.EnableTelemetry && req.DisableTelemetry {
		return nil, status.Error(codes.InvalidArgument, "Both enable_telemetry and disable_telemetry are present.")
	}

	var settings *models.Settings
	err := s.db.InTransaction(func(tx *reform.TX) error {
		var e error
		if settings, e = models.GetSettings(tx); e != nil {
			return e
		}

		if s.envDisableTelemetry && (req.EnableTelemetry || req.DisableTelemetry) {
			return status.Error(codes.FailedPrecondition, "Telemetry is disabled via DISABLE_TELEMETRY environment variable.")
		}
		if req.EnableTelemetry {
			settings.Telemetry.Disabled = false
		}
		if req.DisableTelemetry {
			settings.Telemetry.Disabled = true
		}

		// absent or zero value means "do not change"
		res := req.GetMetricsResolutions()
		if hr, e := ptypes.Duration(res.GetHr()); e == nil && hr != 0 {
			if s.envMetricsResolution != 0 {
				return status.Error(codes.FailedPrecondition, "High resolution for metrics is set via METRICS_RESOLUTION environment variable.")
			}
			settings.MetricsResolutions.HR = hr
		}
		if mr, e := ptypes.Duration(res.GetMr()); e == nil && mr != 0 {
			settings.MetricsResolutions.MR = mr
		}
		if lr, e := ptypes.Duration(res.GetLr()); e == nil && lr != 0 {
			settings.MetricsResolutions.LR = lr
		}

		// absent or zero value means "do not change"
		if dr, e := ptypes.Duration(req.GetDataRetention()); e == nil && dr != 0 {
			if s.envDataRetention != 0 {
				return status.Error(codes.FailedPrecondition, "Data retention for queries is set via DATA_RETENTION environment variable.")
			}
			settings.DataRetention = dr
		}

		return models.SaveSettings(tx, settings)
	})
	if err != nil {
		return nil, err
	}

	err = s.supervisord.UpdateConfiguration(settings)
	s.prometheus.RequestConfigurationUpdate()
	if err != nil {
		return nil, err
	}

	res := &serverpb.ChangeSettingsResponse{
		Settings: convertSettings(settings),
	}
	res.Settings.UpdatesDisabled = s.envDisableUpdates
	return res, nil
}

// check interfaces
var (
	_ serverpb.ServerServer = (*Server)(nil)
)
