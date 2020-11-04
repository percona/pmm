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
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/google/uuid"
	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/utils/pdeathsig"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/envvars"
)

const platformAPITimeout = 10 * time.Second

// Server represents service for checking PMM Server status and changing settings.
type Server struct {
	db                      *reform.DB
	vmdb                    prometheusService
	vmalert                 prometheusService
	prometheusAlertingRules prometheusAlertingRules
	alertmanager            alertmanagerService
	supervisord             supervisordService
	telemetryService        telemetryService
	platformService         platformService
	awsInstanceChecker      *AWSInstanceChecker
	grafanaClient           grafanaClient
	l                       *logrus.Entry

	pmmUpdateAuthFileM sync.Mutex
	pmmUpdateAuthFile  string

	envRW       sync.RWMutex
	envSettings *models.ChangeSettingsParams

	sshKeyM sync.Mutex
}

type pmmUpdateAuth struct {
	AuthToken string `json:"auth_token"`
}

// Params holds the parameters needed to create a new service.
type Params struct {
	DB                      *reform.DB
	VMDB                    prometheusService
	VMAlert                 prometheusService
	Alertmanager            alertmanagerService
	PrometheusAlertingRules prometheusAlertingRules
	Supervisord             supervisordService
	TelemetryService        telemetryService
	PlatformService         platformService
	AwsInstanceChecker      *AWSInstanceChecker
	GrafanaClient           grafanaClient
}

// NewServer returns new server for Server service.
func NewServer(params *Params) (*Server, error) {
	path := os.TempDir()
	if _, err := os.Stat(path); err != nil {
		return nil, errors.WithStack(err)
	}
	path = filepath.Join(path, "pmm-update.json")

	s := &Server{
		db:                      params.DB,
		vmdb:                    params.VMDB,
		vmalert:                 params.VMAlert,
		alertmanager:            params.Alertmanager,
		prometheusAlertingRules: params.PrometheusAlertingRules,
		supervisord:             params.Supervisord,
		telemetryService:        params.TelemetryService,
		platformService:         params.PlatformService,
		awsInstanceChecker:      params.AwsInstanceChecker,
		grafanaClient:           params.GrafanaClient,
		l:                       logrus.WithField("component", "server"),
		pmmUpdateAuthFile:       path,
		envSettings:             new(models.ChangeSettingsParams),
	}
	return s, nil
}

// UpdateSettingsFromEnv updates settings in the database with environment variables values.
// It returns only validation or database errors; invalid environment variables are logged and skipped.
func (s *Server) UpdateSettingsFromEnv(env []string) []error {
	s.envRW.Lock()
	defer s.envRW.Unlock()

	envSettings, errs, warns := envvars.ParseEnvVars(env)
	for _, w := range warns {
		s.l.Warnln(w)
	}
	if len(errs) > 0 {
		return errs
	}

	err := s.db.InTransaction(func(tx *reform.TX) error {
		_, err := models.UpdateSettings(tx.Querier, envSettings)
		return err
	})
	if err != nil {
		return []error{err}
	}
	s.envSettings = envSettings
	return nil
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

		DistributionMethod: s.telemetryService.DistributionMethod(),
	}
	if t, err := version.Time(); err == nil {
		ts, _ := ptypes.TimestampProto(t)
		res.Managed.Timestamp = ts
	}

	if v := s.supervisord.InstalledPMMVersion(ctx); v != nil {
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
	var notReady bool
	for n, svc := range map[string]healthChecker{
		"alertmanager":    s.alertmanager,
		"grafana":         s.grafanaClient,
		"vmalert":         s.vmalert,
		"victoriametrics": s.vmdb,
	} {
		if err := svc.IsReady(ctx); err != nil {
			s.l.Errorf("%s readiness check failed: %+v", n, err)
			notReady = true
		}
	}

	if notReady {
		return nil, status.Error(codes.Internal, "PMM Server is not ready yet.")
	}

	return &serverpb.ReadinessResponse{}, nil
}

// CheckUpdates checks PMM Server updates availability.
func (s *Server) CheckUpdates(ctx context.Context, req *serverpb.CheckUpdatesRequest) (*serverpb.CheckUpdatesResponse, error) {
	s.envRW.RLock()
	updatesDisabled := s.envSettings.DisableUpdates
	s.envRW.RUnlock()

	if req.Force {
		if err := s.supervisord.ForceCheckUpdates(ctx); err != nil {
			return nil, err
		}
	}

	v, lastCheck := s.supervisord.LastCheckUpdatesResult(ctx)
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
		LatestNewsUrl:   v.LatestNewsURL,
	}

	if updatesDisabled {
		res.UpdateAvailable = false
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
	updatesDisabled := s.envSettings.DisableUpdates
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
		done = !s.supervisord.UpdateRunning()
		if done {
			// give supervisord a second to flush logs to file
			time.Sleep(time.Second)
		}

		lines, newOffset, err = s.supervisord.UpdateLog(req.LogOffset)
		if err != nil {
			s.l.Warn(err)
		}

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

// convertSettings merges database settings and settings from environment variables into API response.
func (s *Server) convertSettings(settings *models.Settings) *serverpb.Settings {
	res := &serverpb.Settings{
		UpdatesDisabled:  s.envSettings.DisableUpdates,
		TelemetryEnabled: !settings.Telemetry.Disabled,
		MetricsResolutions: &serverpb.MetricsResolutions{
			Hr: ptypes.DurationProto(settings.MetricsResolutions.HR),
			Mr: ptypes.DurationProto(settings.MetricsResolutions.MR),
			Lr: ptypes.DurationProto(settings.MetricsResolutions.LR),
		},
		DataRetention:   ptypes.DurationProto(settings.DataRetention),
		SshKey:          settings.SSHKey,
		AwsPartitions:   settings.AWSPartitions,
		AlertManagerUrl: settings.AlertManagerURL,
		SttEnabled:      settings.SaaS.STTEnabled,
		PlatformEmail:   settings.SaaS.Email,
		DbaasEnabled:    settings.DBaaS.Enabled,
	}

	b, err := s.prometheusAlertingRules.ReadRules()
	if err != nil {
		s.l.Warnf("Cannot load Alert Manager rules: %s", err)
	}
	res.AlertManagerRules = b

	return res
}

// GetSettings returns current PMM Server settings.
func (s *Server) GetSettings(ctx context.Context, req *serverpb.GetSettingsRequest) (*serverpb.GetSettingsResponse, error) {
	s.envRW.RLock()
	defer s.envRW.RUnlock()

	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	return &serverpb.GetSettingsResponse{
		Settings: s.convertSettings(settings),
	}, nil
}

func getDuration(p *duration.Duration) time.Duration {
	d, _ := ptypes.Duration(p)
	return d
}

func (s *Server) validateChangeSettingsRequest(ctx context.Context, req *serverpb.ChangeSettingsRequest) error {
	metricsRes := req.MetricsResolutions

	// check request parameters

	if req.AlertManagerRules != "" && req.RemoveAlertManagerRules {
		return status.Error(codes.InvalidArgument, "Both alert_manager_rules and remove_alert_manager_rules are present.")
	}
	if req.SshKey != "" {
		if err := s.validateSSHKey(ctx, req.SshKey); err != nil {
			return err
		}
	}

	if req.AlertManagerRules != "" {
		if err := s.prometheusAlertingRules.ValidateRules(ctx, req.AlertManagerRules); err != nil {
			return err
		}
	}

	// check request parameters compatibility with environment variables

	// ignore req.DisableTelemetry and req.DisableStt even if they are present since that will not change anything
	if req.EnableTelemetry && s.envSettings.DisableTelemetry {
		return status.Error(codes.FailedPrecondition, "Telemetry is disabled via DISABLE_TELEMETRY environment variable.")
	}
	if req.EnableStt && s.envSettings.DisableTelemetry {
		return status.Error(codes.FailedPrecondition, "STT cannot be enabled because telemetry is disabled via DISABLE_TELEMETRY environment variable.")
	}

	if getDuration(metricsRes.GetHr()) != 0 && s.envSettings.MetricsResolutions.HR != 0 {
		return status.Error(codes.FailedPrecondition, "High resolution for metrics is set via METRICS_RESOLUTION_HR (or METRICS_RESOLUTION) environment variable.")
	}
	if getDuration(metricsRes.GetMr()) != 0 && s.envSettings.MetricsResolutions.MR != 0 {
		return status.Error(codes.FailedPrecondition, "Medium resolution for metrics is set via METRICS_RESOLUTION_MR environment variable.")
	}
	if getDuration(metricsRes.GetLr()) != 0 && s.envSettings.MetricsResolutions.LR != 0 {
		return status.Error(codes.FailedPrecondition, "Low resolution for metrics is set via METRICS_RESOLUTION_LR environment variable.")
	}

	if getDuration(req.DataRetention) != 0 && s.envSettings.DataRetention != 0 {
		return status.Error(codes.FailedPrecondition, "Data retention for queries is set via DATA_RETENTION environment variable.")
	}

	return nil
}

// ChangeSettings changes PMM Server settings.
func (s *Server) ChangeSettings(ctx context.Context, req *serverpb.ChangeSettingsRequest) (*serverpb.ChangeSettingsResponse, error) {
	s.envRW.RLock()
	defer s.envRW.RUnlock()

	if err := s.validateChangeSettingsRequest(ctx, req); err != nil {
		return nil, err
	}

	var settings *models.Settings
	err := s.db.InTransaction(func(tx *reform.TX) error {
		metricsRes := req.MetricsResolutions
		settingsParams := &models.ChangeSettingsParams{
			DisableTelemetry: req.DisableTelemetry,
			EnableTelemetry:  req.EnableTelemetry,
			MetricsResolutions: models.MetricsResolutions{
				HR: getDuration(metricsRes.GetHr()),
				MR: getDuration(metricsRes.GetMr()),
				LR: getDuration(metricsRes.GetLr()),
			},
			DataRetention:         getDuration(req.DataRetention),
			AWSPartitions:         req.AwsPartitions,
			AlertManagerURL:       req.AlertManagerUrl,
			RemoveAlertManagerURL: req.RemoveAlertManagerUrl,
			SSHKey:                req.SshKey,
			EnableSTT:             req.EnableStt,
			DisableSTT:            req.DisableStt,
		}

		var e error
		if settings, e = models.UpdateSettings(tx, settingsParams); e != nil {
			return status.Error(codes.InvalidArgument, e.Error())
		}

		// absent value means "do not change"
		if req.SshKey != "" {
			if e = s.writeSSHKey(req.SshKey); e != nil {
				return errors.WithStack(e)
			}
		}

		// absent value means "do not change"
		if req.AlertManagerRules != "" {
			if e = s.prometheusAlertingRules.WriteRules(req.AlertManagerRules); e != nil {
				return errors.WithStack(e)
			}
		}
		if req.RemoveAlertManagerRules {
			if e = s.prometheusAlertingRules.RemoveRulesFile(); e != nil && !os.IsNotExist(e) {
				return errors.WithStack(e)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = s.supervisord.UpdateConfiguration(settings)
	s.vmdb.RequestConfigurationUpdate()
	s.vmalert.RequestConfigurationUpdate()
	if err != nil {
		return nil, err
	}

	return &serverpb.ChangeSettingsResponse{
		Settings: s.convertSettings(settings),
	}, nil
}

func (s *Server) validateSSHKey(ctx context.Context, sshKey string) error {
	tempFile, err := ioutil.TempFile("", "temp_ssh_keys_*")
	if err != nil {
		return errors.WithStack(err)
	}
	tempFile.Close()                 //nolint:errcheck
	defer os.Remove(tempFile.Name()) //nolint:errcheck

	if err = ioutil.WriteFile(tempFile.Name(), []byte(sshKey), 0600); err != nil {
		return errors.WithStack(err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "ssh-keygen", "-l", "-f", tempFile.Name()) //nolint:gosec
	pdeathsig.Set(cmd, unix.SIGKILL)

	if err = cmd.Run(); err != nil {
		if e, ok := err.(*exec.ExitError); ok && e.ExitCode() != 0 {
			return status.Errorf(codes.InvalidArgument, "Invalid SSH key.")
		}
		return errors.WithStack(err)
	}

	return nil
}

func (s *Server) writeSSHKey(sshKey string) error {
	s.sshKeyM.Lock()
	defer s.sshKeyM.Unlock()

	const username = "admin"
	usr, err := user.Lookup(username)
	if err != nil {
		return errors.WithStack(err)
	}
	sshDirPath := path.Join(usr.HomeDir, ".ssh")
	if err = os.MkdirAll(sshDirPath, 0700); err != nil {
		return errors.WithStack(err)
	}

	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		return errors.WithStack(err)
	}
	gid, err := strconv.Atoi(usr.Gid)
	if err != nil {
		return errors.WithStack(err)
	}
	if err = os.Chown(sshDirPath, uid, gid); err != nil {
		return errors.WithStack(err)
	}
	keysPath := path.Join(sshDirPath, "authorized_keys")
	if err = ioutil.WriteFile(keysPath, []byte(sshKey), 0600); err != nil {
		return errors.WithStack(err)
	}
	if err = os.Chown(keysPath, uid, gid); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// AWSInstanceCheck checks AWS EC2 instance ID.
func (s *Server) AWSInstanceCheck(ctx context.Context, req *serverpb.AWSInstanceCheckRequest) (*serverpb.AWSInstanceCheckResponse, error) {
	if err := s.awsInstanceChecker.check(req.InstanceId); err != nil {
		return nil, err
	}
	return &serverpb.AWSInstanceCheckResponse{}, nil
}

// PlatformSignUp creates new Percona Platform user with given email and password.
func (s *Server) PlatformSignUp(ctx context.Context, req *serverpb.PlatformSignUpRequest) (*serverpb.PlatformSignUpResponse, error) {
	nCtx, cancel := context.WithTimeout(ctx, platformAPITimeout)
	defer cancel()
	if err := s.platformService.SignUp(nCtx, req.Email, req.Password); err != nil {
		return nil, err
	}

	return &serverpb.PlatformSignUpResponse{}, nil
}

// PlatformSignIn links that PMM instance to Percona Platform user and created new session.
func (s *Server) PlatformSignIn(ctx context.Context, req *serverpb.PlatformSignInRequest) (*serverpb.PlatformSignInResponse, error) {
	nCtx, cancel := context.WithTimeout(ctx, platformAPITimeout)
	defer cancel()
	if err := s.platformService.SignIn(nCtx, req.Email, req.Password); err != nil {
		return nil, err
	}

	return &serverpb.PlatformSignInResponse{}, nil
}

// PlatformSignOut logouts that PMM instance from Percona Platform account.
func (s *Server) PlatformSignOut(ctx context.Context, _ *serverpb.PlatformSignOutRequest) (*serverpb.PlatformSignOutResponse, error) {
	nCtx, cancel := context.WithTimeout(ctx, platformAPITimeout)
	defer cancel()
	if err := s.platformService.SignOut(nCtx); err != nil {
		return nil, err
	}

	return &serverpb.PlatformSignOutResponse{}, nil
}

// check interfaces
var (
	_ serverpb.ServerServer = (*Server)(nil)
)
