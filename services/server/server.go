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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

const updateCheckInterval = 24 * time.Hour

// Server represents service for checking PMM Server status and changing settings.
type Server struct {
	db                    *reform.DB
	prometheus            prometheusService
	supervisord           supervisordService
	l                     *logrus.Entry
	pmmUpdate             *pmmUpdate
	pmmUpdateProgressFile string

	pmmUpdateProgressFileM sync.Mutex

	envMetricsResolution time.Duration
	envDisableTelemetry  bool
}

type pmmUpdateProgress struct {
	AuthToken string `json:"auth_token"`
}

// NewServer returns new server for Server service.
func NewServer(db *reform.DB, prometheus prometheusService, supervisord supervisordService, env []string) (*Server, error) {
	path := os.TempDir()
	if _, err := os.Stat(path); err != nil {
		return nil, errors.WithStack(err)
	}
	path = filepath.Join(path, "pmm-update.json")

	s := &Server{
		db:                    db,
		prometheus:            prometheus,
		l:                     logrus.WithField("component", "server"),
		pmmUpdate:             newPMMUpdate(logrus.WithField("component", "server/pmm-update")),
		pmmUpdateProgressFile: path,
	}
	s.parseEnv(env)
	return s, nil
}

func (s *Server) parseEnv(env []string) {
	for _, e := range env {
		p := strings.SplitN(e, "=", 2)
		if len(p) != 2 {
			s.l.Warnf("Failed to parse environment variable %s.", e)
			continue
		}

		k, v := strings.ToUpper(p[0]), strings.ToLower(p[1])
		var err error
		switch k {
		case "METRICS_RESOLUTION":
			var d time.Duration
			d, err = time.ParseDuration(v)
			if err != nil {
				i, _ := strconv.ParseInt(v, 10, 64)
				if i != 0 {
					d = time.Duration(i) * time.Second
					err = nil
				}
			}
			if d != 0 && d < time.Second {
				s.l.Warnf("Failed to parse environment variable %s: minimal resolution is 1s.", e)
				continue
			}
			if err == nil {
				s.envMetricsResolution = d
			}

		case "DISABLE_TELEMETRY":
			var b bool
			b, err = strconv.ParseBool(v)
			if err == nil {
				s.envDisableTelemetry = b
			}
		}

		if err != nil {
			s.l.Warnf("Failed to parse environment variable %s: %s", e, err)
		}
	}
}

// Run runs check for updates loop until ctx is canceled.
func (s *Server) Run(ctx context.Context) {
	s.l.Info("Starting...")
	ticker := time.NewTicker(updateCheckInterval)
	defer ticker.Stop()

	for {
		_ = s.pmmUpdate.check()

		select {
		case <-ticker.C:
			// continue with next loop iteration
		case <-ctx.Done():
			s.l.Info("Done.")
			return
		}
	}
}

// UpdateSettings updates settings in the database with environment variables values.
func (s *Server) UpdateSettings() error {
	if s.envMetricsResolution == 0 && !s.envDisableTelemetry {
		return nil
	}

	return s.db.InTransaction(func(tx *reform.TX) error {
		settings, err := models.GetSettings(tx.Querier)
		if err != nil {
			return err
		}

		if s.envMetricsResolution != 0 {
			settings.MetricsResolutions.HR = s.envMetricsResolution
		}
		if s.envDisableTelemetry {
			settings.Telemetry.Disabled = true
		}

		return models.SaveSettings(tx.Querier, settings)
	})
}

func convertSettings(s *models.Settings) *serverpb.Settings {
	return &serverpb.Settings{
		MetricsResolutions: &serverpb.MetricsResolutions{
			Hr: ptypes.DurationProto(s.MetricsResolutions.HR),
			Mr: ptypes.DurationProto(s.MetricsResolutions.MR),
			Lr: ptypes.DurationProto(s.MetricsResolutions.LR),
		},
		Telemetry: !s.Telemetry.Disabled,
	}
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

	if v, _ := s.pmmUpdate.checkResult(); v != nil {
		res.Version = v.InstalledRPMNiceVersion
		res.Server = &serverpb.VersionInfo{
			Version:     v.InstalledRPMNiceVersion,
			FullVersion: v.InstalledRPMVersion,
		}
		if v.InstalledTime != nil {
			res.Server.Timestamp, _ = ptypes.TimestampProto(*v.InstalledTime)
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
		if err := s.pmmUpdate.check(); err != nil {
			return nil, err
		}
	}

	v, lastCheck := s.pmmUpdate.checkResult()
	if v == nil {
		return nil, status.Error(codes.Unavailable, "failed to check for updates")
	}

	res := &serverpb.CheckUpdatesResponse{
		Installed: &serverpb.VersionInfo{
			Version:     v.InstalledRPMNiceVersion,
			FullVersion: v.InstalledRPMVersion,
		},
		Latest: &serverpb.VersionInfo{
			Version:     v.LatestRPMNiceVersion,
			FullVersion: v.LatestRPMVersion,
		},
		UpdateAvailable: v.UpdateAvailable,
		LatestNewsUrl:   "", // TODO https://jira.percona.com/browse/PMM-4444
	}
	res.LastCheck, _ = ptypes.TimestampProto(lastCheck)
	if v.InstalledTime != nil {
		t := v.InstalledTime.UTC().Truncate(24 * time.Hour) // return only date
		res.Installed.Timestamp, _ = ptypes.TimestampProto(t)
	}
	if v.LatestTime != nil {
		t := v.LatestTime.UTC().Truncate(24 * time.Hour) // return only date
		res.Latest.Timestamp, _ = ptypes.TimestampProto(t)
	}
	return res, nil
}

// StartUpdate starts PMM Server update.
func (s *Server) StartUpdate(ctx context.Context, req *serverpb.StartUpdateRequest) (*serverpb.StartUpdateResponse, error) {
	var authToken string

	// TODO https://jira.percona.com/browse/PMM-4448
	// e := s.db.InTransaction(func(tx *reform.TX) error {
	// 	settings, err := models.GetSettings(tx.Querier)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if settings.Updates.AuthToken != "" {
	// 		return status.Error(codes.AlreadyExists, "Update is already underway.")
	// 	}
	// 	authToken = "/update_auth_token/" + uuid.New().String()
	// 	settings.Updates.AuthToken = authToken
	// 	return models.SaveSettings(tx.Querier, settings)
	// })
	// if e != nil {
	// 	return nil, e
	// }

	return &serverpb.StartUpdateResponse{
		AuthToken: authToken,
	}, nil
}

// UpdateStatus returns PMM Server update status.
func (s *Server) UpdateStatus(ctx context.Context, req *serverpb.UpdateStatusRequest) (*serverpb.UpdateStatusResponse, error) {
	// TODO https://jira.percona.com/browse/PMM-4448

	// settings, err := models.GetSettings(s.db.Querier)
	// if err != nil {
	// 	return nil, err
	// }

	// if subtle.ConstantTimeCompare([]byte(req.AuthToken), []byte(settings.Updates.AuthToken)) == 0 {
	// 	return nil, status.Error(codes.PermissionDenied, "Invalid authentication token.")
	// }

	// e := s.db.InTransaction(func(tx *reform.TX) error {
	// 	settings, err = models.GetSettings(tx.Querier)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	settings.Updates.AuthToken = ""
	// 	return models.SaveSettings(tx.Querier, settings)
	// })
	// if e != nil {
	// 	return nil, err
	// }

	return &serverpb.UpdateStatusResponse{
		LogLines: []string{
			"TODO",
		},
		Done: true,
	}, nil
}

// GetSettings returns current PMM Server settings.
func (s *Server) GetSettings(ctx context.Context, req *serverpb.GetSettingsRequest) (*serverpb.GetSettingsResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}
	res := &serverpb.GetSettingsResponse{
		Settings: convertSettings(settings),
	}
	return res, nil
}

// ChangeSettings changes PMM Server settings.
func (s *Server) ChangeSettings(ctx context.Context, req *serverpb.ChangeSettingsRequest) (*serverpb.ChangeSettingsResponse, error) {
	if req.EnableTelemetry && req.DisableTelemetry {
		return nil, status.Error(codes.InvalidArgument, "Both enable_telemetry and disable_telemetry are present.")
	}

	var settings *models.Settings
	err := s.db.InTransaction(func(tx *reform.TX) error {
		var e error
		if settings, e = models.GetSettings(tx); e != nil {
			return e
		}

		// absent or zero resolution value means "do not change"
		if res := req.MetricsResolutions; res != nil {
			if hr, e := ptypes.Duration(res.Hr); e == nil && hr != 0 {
				if s.envMetricsResolution != 0 {
					return status.Error(codes.FailedPrecondition, "High resolution for metrics is set via METRICS_RESOLUTION environment variable.")
				}
				if hr < time.Second {
					return status.Error(codes.FailedPrecondition, "Minimal resolution is 1s.")
				}
				settings.MetricsResolutions.HR = hr
			}
			if mr, e := ptypes.Duration(res.Mr); e == nil && mr != 0 {
				if mr < time.Second {
					return status.Error(codes.FailedPrecondition, "Minimal resolution is 1s.")
				}
				settings.MetricsResolutions.MR = mr
			}
			if lr, e := ptypes.Duration(res.Lr); e == nil && lr != 0 {
				if lr < time.Second {
					return status.Error(codes.FailedPrecondition, "Minimal resolution is 1s.")
				}
				settings.MetricsResolutions.LR = lr
			}
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

		return models.SaveSettings(tx, settings)
	})
	if err != nil {
		return nil, err
	}

	s.prometheus.UpdateConfiguration()

	res := &serverpb.ChangeSettingsResponse{
		Settings: convertSettings(settings),
	}
	return res, nil
}

// check interfaces
var (
	_ serverpb.ServerServer = (*Server)(nil)
)
