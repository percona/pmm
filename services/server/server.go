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
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/version"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
)

// Server represents service for checking PMM Server status and changing settings.
type Server struct {
	db         *reform.DB
	prometheus prometheusService
	l          *logrus.Entry

	envMetricsResolution time.Duration
	envDisableTelemetry  bool
}

// NewServer returns new server for Server service.
func NewServer(db *reform.DB, prometheus prometheusService, env []string) *Server {
	s := &Server{
		db:         db,
		prometheus: prometheus,
		l:          logrus.WithField("component", "server"),
	}
	s.parseEnv(env)
	return s
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

// Version returns PMM Server version.
func (s *Server) Version(ctx context.Context, req *serverpb.VersionRequest) (*serverpb.VersionResponse, error) {
	res := &serverpb.VersionResponse{
		Version:          version.Version,
		PmmManagedCommit: version.FullCommit,
	}

	sec, err := strconv.ParseInt(version.Timestamp, 10, 64)
	if err == nil {
		res.Timestamp, err = ptypes.TimestampProto(time.Unix(sec, 0))
	}
	if err != nil {
		logger.Get(ctx).Warn(err)
	}

	return res, nil
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
