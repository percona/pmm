// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package client contains business logic of working with pmm-managed.
package client

import (
	"context"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/client/channel"
	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/connectionuptime"
	"github.com/percona/pmm/agent/runner"
	"github.com/percona/pmm/agent/runner/actions" // TODO https://jira.percona.com/browse/PMM-7206
	"github.com/percona/pmm/agent/runner/jobs"
	"github.com/percona/pmm/agent/tailog"
	"github.com/percona/pmm/agent/utils/backoff"
	agenterrors "github.com/percona/pmm/agent/utils/errors"
	"github.com/percona/pmm/agent/utils/templates"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/utils/tlsconfig"
	"github.com/percona/pmm/version"
)

const (
	dialTimeout       = 5 * time.Second
	backoffMinDelay   = 1 * time.Second
	backoffMaxDelay   = 15 * time.Second
	clockDriftWarning = 5 * time.Second
)

// configGetter allows to get a config.
type configGetter interface {
	Get() *config.Config
}

// Client represents pmm-agent's connection to nginx/pmm-managed.
type Client struct {
	cfg               configGetter
	supervisor        supervisor
	connectionChecker connectionChecker
	softwareVersioner softwareVersioner
	serviceInfoBroker serviceInfoBroker

	l       *logrus.Entry
	backoff *backoff.Backoff
	done    chan struct{}

	// for unit tests only
	dialTimeout time.Duration

	runner *runner.Runner

	rw      sync.RWMutex
	md      *agentv1.ServerConnectMetadata
	channel *channel.Channel

	cus      *connectionuptime.Service
	logStore *tailog.Store
}

// New creates new client.
//
// Caller should call Run.
func New(cfg configGetter, supervisor supervisor, r *runner.Runner, connectionChecker connectionChecker, sv softwareVersioner, sib serviceInfoBroker, cus *connectionuptime.Service, logStore *tailog.Store) *Client { //nolint:lll
	return &Client{
		cfg:               cfg,
		supervisor:        supervisor,
		connectionChecker: connectionChecker,
		softwareVersioner: sv,
		serviceInfoBroker: sib,
		l:                 logrus.WithField("component", "client"),
		backoff:           backoff.New(backoffMinDelay, backoffMaxDelay),
		dialTimeout:       dialTimeout,
		runner:            r,
		cus:               cus,
		logStore:          logStore,
	}
}

// Run connects to the server, processes requests and sends responses.
//
// Once Run exits, connection is closed, and caller should cancel supervisor's context.
// Then caller should wait until Done() channel is closed.
// That Client instance can't be reused after that.
//
// Returned error is already logged and should be ignored. It is returned only for unit tests.
func (c *Client) Run(ctx context.Context) error {
	c.l.Info("Starting...")

	c.rw.Lock()
	c.done = make(chan struct{})
	c.rw.Unlock()

	cfg := c.cfg.Get()

	// do nothing until ctx is canceled if config misses critical info
	var missing string
	if cfg.ID == "" {
		missing = "Agent ID"
	}
	if cfg.Server.Address == "" {
		missing = "PMM Server address"
	}
	if missing != "" {
		c.l.Errorf("%s is not provided, halting.", missing)
		<-ctx.Done()
		close(c.done)
		return errors.Wrap(ctx.Err(), "missing "+missing)
	}

	// try to connect until success, or until ctx is canceled
	var dialResult *dialResult
	var dialErr error
	for {
		dialCtx, dialCancel := context.WithTimeout(ctx, c.dialTimeout)
		dialResult, dialErr = dial(dialCtx, cfg, c.l)

		c.cus.RegisterConnectionStatus(time.Now(), dialErr == nil)
		dialCancel()
		if dialResult != nil {
			break
		}

		retryCtx, retryCancel := context.WithTimeout(ctx, c.backoff.Delay())
		<-retryCtx.Done()
		retryCancel()
		if ctx.Err() != nil {
			break
		}
	}
	if ctx.Err() != nil {
		close(c.done)
		if dialErr != nil {
			return dialErr
		}
		return ctx.Err()
	}

	c.backoff.Reset()

	defer func() {
		if err := dialResult.conn.Close(); err != nil {
			c.l.Errorf("Connection closed: %s.", err)
			return
		}
		c.l.Info("Connection closed.")
	}()

	c.rw.Lock()
	c.md = dialResult.md
	c.channel = dialResult.channel
	c.rw.Unlock()

	// Once the client is connected, ctx cancellation is ignored by it.
	//
	// We start goroutines, and terminate the gRPC connection and exit Run when any of them exits:
	//
	// 1. processActionResults reads action results from action runner and sends them to the channel.
	//    It exits when the action runner is stopped by cancelling ctx.
	//
	// 2. processSupervisorRequests reads requests (status changes and QAN data) from the supervisor and sends them to the channel.
	//    It exits when the supervisor is stopped by the caller.
	//    Caller stops supervisor when Run is left and gRPC connection is closed.
	//
	// 3. processChannelRequests reads requests from the channel and processes them.
	//    It exits when an unexpected message is received from the channel, or when can't be received at all.
	//    When Run is left, caller stops supervisor, and that allows processSupervisorRequests to exit.
	//
	// Done() channel is closed when all three goroutines exited.

	// TODO Make 2 and 3 behave more like 1 - that seems to be simpler.
	// https://jira.percona.com/browse/PMM-4245

	c.supervisor.ClearChangesChannel()
	c.SendActualStatuses()

	oneDone := make(chan struct{}, 4)
	go func() {
		c.processActionResults(ctx)
		c.l.Debug("processActionResults is finished")
		oneDone <- struct{}{}
	}()
	go func() {
		c.processJobsResults(ctx)
		c.l.Debug("processJobsResults is finished")
		oneDone <- struct{}{}
	}()
	go func() {
		c.processSupervisorRequests(ctx)
		c.l.Debug("processSupervisorRequests is finished")
		oneDone <- struct{}{}
	}()
	go func() {
		c.processChannelRequests(ctx)
		c.l.Debug("processChannelRequests is finished")
		oneDone <- struct{}{}
	}()

	<-oneDone
	go func() {
		<-oneDone
		<-oneDone
		<-oneDone
		c.l.Info("Done.")
		close(c.done)
	}()
	return nil
}

// SendActualStatuses sends status of running agents to server.
func (c *Client) SendActualStatuses() {
	for _, agent := range c.supervisor.AgentsList() {
		c.l.Infof("Sending status: %s (port %d).", agent.Status, agent.ListenPort)
		resp, err := c.channel.SendAndWaitResponse(
			&agentv1.StateChangedRequest{
				AgentId:         agent.AgentId,
				Status:          agent.Status,
				ListenPort:      agent.ListenPort,
				ProcessExecPath: agent.GetProcessExecPath(),
			})
		if err != nil {
			c.l.Error(err)
			continue
		}
		if resp == nil {
			c.l.Warn("Failed to send StateChanged request.")
		}
	}
}

// Done is closed when all supervisor's requests are sent (if possible) and connection is closed.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

func (c *Client) processActionResults(ctx context.Context) {
	for {
		select {
		case result := <-c.runner.ActionsResults():
			if result == nil {
				continue
			}
			resp, err := c.channel.SendAndWaitResponse(result)
			if err != nil {
				c.l.Error(err)
				continue
			}
			if resp == nil {
				c.l.Warn("Failed to send ActionResult request.")
			}
		case <-ctx.Done():
			c.l.Infof("Actions runner Results() channel drained.")
			return
		}
	}
}

func (c *Client) processJobsResults(ctx context.Context) {
	for {
		select {
		case message := <-c.runner.JobsMessages():
			if message == nil {
				continue
			}
			c.channel.Send(&channel.AgentResponse{
				ID:      0, // Jobs send messages that don't require any responses, so we can leave message ID blank.
				Payload: message,
			})
		case <-ctx.Done():
			c.l.Infof("Jobs runner Messages() channel drained.")
			return
		}
	}
}

func (c *Client) processSupervisorRequests(ctx context.Context) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case state := <-c.supervisor.Changes():
				if state == nil {
					continue
				}
				resp, err := c.channel.SendAndWaitResponse(state)
				if err != nil {
					c.l.Error(err)
					continue
				}
				if resp == nil {
					c.l.Warn("Failed to send StateChanged request.")
				}
			case <-ctx.Done():
				c.l.Infof("Supervisor Changes() channel drained.")
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case collect := <-c.supervisor.QANRequests():
				if collect == nil {
					continue
				}
				resp, err := c.channel.SendAndWaitResponse(collect)
				if err != nil {
					c.l.Error(err)
					continue
				}
				if resp == nil {
					c.l.Warn("Failed to send QanCollect request.")
				}
			case <-ctx.Done():
				c.l.Infof("Supervisor QANRequests() channel drained.")
				return
			}
		}
	}()

	wg.Wait()
}

func (c *Client) processChannelRequests(ctx context.Context) {
LOOP:
	for {
		select {
		case req, more := <-c.channel.Requests():
			if !more {
				break LOOP
			}
			var responsePayload agentv1.AgentResponsePayload
			var status *grpcstatus.Status
			switch p := req.Payload.(type) {
			case *agentv1.Ping:
				responsePayload = &agentv1.Pong{
					CurrentTime: timestamppb.Now(),
				}
			case *agentv1.SetStateRequest:
				c.supervisor.SetState(p)
				responsePayload = &agentv1.SetStateResponse{}

			case *agentv1.StartActionRequest:
				responsePayload = &agentv1.StartActionResponse{}
				if err := c.handleStartActionRequest(p); err != nil {
					status = convertAgentErrorToGrpcStatus(err)
					break
				}

			case *agentv1.StopActionRequest:
				c.runner.Stop(p.ActionId)
				responsePayload = &agentv1.StopActionResponse{}

			case *agentv1.CheckConnectionRequest:
				responsePayload = c.connectionChecker.Check(ctx, p, req.ID)

			case *agentv1.ServiceInfoRequest:
				responsePayload = c.serviceInfoBroker.GetInfoFromService(ctx, p, req.ID)

			case *agentv1.StartJobRequest:
				var resp agentv1.StartJobResponse
				if err := c.handleStartJobRequest(p); err != nil {
					resp.Error = err.Error()
				}
				responsePayload = &resp

			case *agentv1.StopJobRequest:
				c.runner.Stop(p.JobId)
				responsePayload = &agentv1.StopJobResponse{}

			case *agentv1.JobStatusRequest:
				alive := c.runner.IsRunning(p.JobId)
				responsePayload = &agentv1.JobStatusResponse{Alive: alive}

			case *agentv1.GetVersionsRequest:
				responsePayload = &agentv1.GetVersionsResponse{Versions: c.handleVersionsRequest(p)}
			case *agentv1.PBMSwitchPITRRequest:
				var resp agentv1.PBMSwitchPITRResponse
				if err := c.handlePBMSwitchRequest(ctx, p, req.ID); err != nil {
					resp.Error = err.Error()
				}
				responsePayload = &resp
			case *agentv1.AgentLogsRequest:
				logs, configLogLinesCount := c.agentLogByID(p.AgentId, p.Limit)
				responsePayload = &agentv1.AgentLogsResponse{
					Logs:                     logs,
					AgentConfigLogLinesCount: uint32(configLogLinesCount), //nolint:gosec // log lines count is not expected to overflow uint32
				}
			default:
				c.l.Errorf("Unhandled server request: %v.", req)
			}
			c.cus.RegisterConnectionStatus(time.Now(), true)

			response := &channel.AgentResponse{
				ID:      req.ID,
				Payload: responsePayload,
			}
			if status != nil {
				response.Status = status
			}
			c.channel.Send(response)
		case <-ctx.Done():
			break LOOP
		}
	}
	if err := c.channel.Wait(); err != nil {
		c.l.Debugf("Channel closed: %s.", err)
		return
	}
	c.l.Debug("Channel closed.")
}

func (c *Client) handleStartActionRequest(p *agentv1.StartActionRequest) error {
	timeout := p.Timeout.AsDuration()
	if err := p.Timeout.CheckValid(); err != nil {
		timeout = 0
	}

	cfg := c.cfg.Get()
	var action actions.Action
	var err error
	switch params := p.Params.(type) {
	case *agentv1.StartActionRequest_MysqlExplainParams:
		action, err = actions.NewMySQLExplainAction(p.ActionId, timeout, params.MysqlExplainParams)

	case *agentv1.StartActionRequest_MysqlShowCreateTableParams:
		action = actions.NewMySQLShowCreateTableAction(p.ActionId, timeout, params.MysqlShowCreateTableParams)

	case *agentv1.StartActionRequest_MysqlShowTableStatusParams:
		action = actions.NewMySQLShowTableStatusAction(p.ActionId, timeout, params.MysqlShowTableStatusParams)

	case *agentv1.StartActionRequest_MysqlShowIndexParams:
		action = actions.NewMySQLShowIndexAction(p.ActionId, timeout, params.MysqlShowIndexParams)

	case *agentv1.StartActionRequest_PostgresqlShowCreateTableParams:
		action, err = actions.NewPostgreSQLShowCreateTableAction(p.ActionId, timeout, params.PostgresqlShowCreateTableParams, cfg.Paths.TempDir)

	case *agentv1.StartActionRequest_PostgresqlShowIndexParams:
		action, err = actions.NewPostgreSQLShowIndexAction(p.ActionId, timeout, params.PostgresqlShowIndexParams, cfg.Paths.TempDir)

	case *agentv1.StartActionRequest_MongodbExplainParams:
		action, err = actions.NewMongoDBExplainAction(p.ActionId, timeout, params.MongodbExplainParams, cfg.Paths.TempDir)

	case *agentv1.StartActionRequest_MysqlQueryShowParams:
		action = actions.NewMySQLQueryShowAction(p.ActionId, timeout, params.MysqlQueryShowParams)

	case *agentv1.StartActionRequest_MysqlQuerySelectParams:
		action = actions.NewMySQLQuerySelectAction(p.ActionId, timeout, params.MysqlQuerySelectParams)

	case *agentv1.StartActionRequest_PostgresqlQueryShowParams:
		action, err = actions.NewPostgreSQLQueryShowAction(p.ActionId, timeout, params.PostgresqlQueryShowParams, cfg.Paths.TempDir)

	case *agentv1.StartActionRequest_PostgresqlQuerySelectParams:
		action, err = actions.NewPostgreSQLQuerySelectAction(p.ActionId, timeout, params.PostgresqlQuerySelectParams, cfg.Paths.TempDir)

	case *agentv1.StartActionRequest_MongodbQueryGetparameterParams:
		action, err = actions.NewMongoDBQueryAdmincommandAction(
			p.ActionId,
			timeout,
			params.MongodbQueryGetparameterParams.Dsn,
			params.MongodbQueryGetparameterParams.TextFiles,
			"getParameter",
			"*",
			cfg.Paths.TempDir)

	case *agentv1.StartActionRequest_MongodbQueryBuildinfoParams:
		action, err = actions.NewMongoDBQueryAdmincommandAction(
			p.ActionId,
			timeout,
			params.MongodbQueryBuildinfoParams.Dsn,
			params.MongodbQueryBuildinfoParams.TextFiles,
			"buildInfo",
			1,
			cfg.Paths.TempDir)

	case *agentv1.StartActionRequest_MongodbQueryGetcmdlineoptsParams:
		action, err = actions.NewMongoDBQueryAdmincommandAction(
			p.ActionId,
			timeout,
			params.MongodbQueryGetcmdlineoptsParams.Dsn,
			params.MongodbQueryGetcmdlineoptsParams.TextFiles,
			"getCmdLineOpts",
			1,
			cfg.Paths.TempDir)

	case *agentv1.StartActionRequest_MongodbQueryReplsetgetstatusParams:
		action, err = actions.NewMongoDBQueryAdmincommandAction(
			p.ActionId,
			timeout,
			params.MongodbQueryReplsetgetstatusParams.Dsn,
			params.MongodbQueryReplsetgetstatusParams.TextFiles,
			"replSetGetStatus",
			1,
			cfg.Paths.TempDir)

	case *agentv1.StartActionRequest_MongodbQueryGetdiagnosticdataParams:
		action, err = actions.NewMongoDBQueryAdmincommandAction(
			p.ActionId,
			timeout,
			params.MongodbQueryGetdiagnosticdataParams.Dsn,
			params.MongodbQueryGetdiagnosticdataParams.TextFiles,
			"getDiagnosticData",
			1,
			cfg.Paths.TempDir)

	case *agentv1.StartActionRequest_PtSummaryParams:
		action = actions.NewProcessAction(p.ActionId, timeout, cfg.Paths.PTSummary, []string{})

	case *agentv1.StartActionRequest_PtPgSummaryParams:
		action = actions.NewProcessAction(p.ActionId, timeout, cfg.Paths.PTPGSummary, argListFromPgParams(params.PtPgSummaryParams))

	case *agentv1.StartActionRequest_PtMysqlSummaryParams:
		action = actions.NewPTMySQLSummaryAction(p.ActionId, timeout, cfg.Paths.PTMySQLSummary, params.PtMysqlSummaryParams)

	case *agentv1.StartActionRequest_PtMongodbSummaryParams:
		action = actions.NewProcessAction(p.ActionId, timeout, cfg.Paths.PTMongoDBSummary, argListFromMongoDBParams(params.PtMongodbSummaryParams))
	case *agentv1.StartActionRequest_RestartSysServiceParams:
		var service string
		switch params.RestartSysServiceParams.SystemService {
		case agentv1.StartActionRequest_RestartSystemServiceParams_SYSTEM_SERVICE_MONGOD:
			service = "mongod"
		case agentv1.StartActionRequest_RestartSystemServiceParams_SYSTEM_SERVICE_PBM_AGENT:
			service = "pbm-agent"
		default:
			return errors.Wrapf(agenterrors.ErrInvalidArgument, "invalid service '%s' specified in mongod restart request", params.RestartSysServiceParams.SystemService)
		}
		action = actions.NewProcessAction(p.ActionId, timeout, "systemctl", []string{"restart", service})

	default:
		return errors.Wrapf(agenterrors.ErrActionUnimplemented, "invalid action type request: %T", params)
	}

	if err != nil {
		return errors.Wrap(err, "failed to create action")
	}

	return c.runner.StartAction(action)
}

func (c *Client) handleStartJobRequest(p *agentv1.StartJobRequest) error {
	if err := p.Timeout.CheckValid(); err != nil {
		return err
	}
	timeout := p.Timeout.AsDuration()

	var job jobs.Job
	switch j := p.Job.(type) {
	case *agentv1.StartJobRequest_MysqlBackup:
		var locationConfig jobs.BackupLocationConfig
		switch cfg := j.MysqlBackup.LocationConfig.(type) {
		case *agentv1.StartJobRequest_MySQLBackup_S3Config:
			locationConfig.Type = jobs.S3BackupLocationType
			locationConfig.S3Config = &jobs.S3LocationConfig{
				Endpoint:     cfg.S3Config.Endpoint,
				AccessKey:    cfg.S3Config.AccessKey,
				SecretKey:    cfg.S3Config.SecretKey,
				BucketName:   cfg.S3Config.BucketName,
				BucketRegion: cfg.S3Config.BucketRegion,
			}
		default:
			return errors.Errorf("unknown location config: %T", j.MysqlBackup.LocationConfig)
		}

		dbConnCfg := jobs.DBConnConfig{
			User:     j.MysqlBackup.User,
			Password: j.MysqlBackup.Password,
			Address:  j.MysqlBackup.Address,
			Port:     int(j.MysqlBackup.Port),
			Socket:   j.MysqlBackup.Socket,
		}
		job = jobs.NewMySQLBackupJob(p.JobId, timeout, j.MysqlBackup.Name, dbConnCfg, locationConfig, j.MysqlBackup.Folder)

	case *agentv1.StartJobRequest_MysqlRestoreBackup:
		var locationConfig jobs.BackupLocationConfig
		switch cfg := j.MysqlRestoreBackup.LocationConfig.(type) {
		case *agentv1.StartJobRequest_MySQLRestoreBackup_S3Config:
			locationConfig.Type = jobs.S3BackupLocationType
			locationConfig.S3Config = &jobs.S3LocationConfig{
				Endpoint:     cfg.S3Config.Endpoint,
				AccessKey:    cfg.S3Config.AccessKey,
				SecretKey:    cfg.S3Config.SecretKey,
				BucketName:   cfg.S3Config.BucketName,
				BucketRegion: cfg.S3Config.BucketRegion,
			}
		default:
			return errors.Errorf("unknown location config: %T", j.MysqlRestoreBackup.LocationConfig)
		}

		job = jobs.NewMySQLRestoreJob(p.JobId, timeout, j.MysqlRestoreBackup.Name, locationConfig, j.MysqlRestoreBackup.Folder)

	case *agentv1.StartJobRequest_MongodbBackup:
		var locationConfig jobs.BackupLocationConfig
		switch cfg := j.MongodbBackup.LocationConfig.(type) {
		case *agentv1.StartJobRequest_MongoDBBackup_S3Config:
			locationConfig.Type = jobs.S3BackupLocationType
			locationConfig.S3Config = &jobs.S3LocationConfig{
				Endpoint:     cfg.S3Config.Endpoint,
				AccessKey:    cfg.S3Config.AccessKey,
				SecretKey:    cfg.S3Config.SecretKey,
				BucketName:   cfg.S3Config.BucketName,
				BucketRegion: cfg.S3Config.BucketRegion,
			}
		case *agentv1.StartJobRequest_MongoDBBackup_FilesystemConfig:
			locationConfig.Type = jobs.FilesystemBackupLocationType
			locationConfig.FilesystemStorageConfig = &jobs.FilesystemBackupLocationConfig{
				Path: cfg.FilesystemConfig.Path,
			}
		default:
			return errors.Errorf("unknown location config: %T", j.MongodbBackup.LocationConfig)
		}

		dsn, err := c.getMongoDSN(j.MongodbBackup.Dsn, j.MongodbBackup.TextFiles, p.JobId)
		if err != nil {
			return errors.WithStack(err)
		}

		job, err = jobs.NewMongoDBBackupJob(p.JobId, timeout, j.MongodbBackup.Name, dsn, locationConfig,
			j.MongodbBackup.EnablePitr, j.MongodbBackup.DataModel, j.MongodbBackup.Folder)
		if err != nil {
			return err
		}

	case *agentv1.StartJobRequest_MongodbRestoreBackup:
		var locationConfig jobs.BackupLocationConfig
		switch cfg := j.MongodbRestoreBackup.LocationConfig.(type) {
		case *agentv1.StartJobRequest_MongoDBRestoreBackup_S3Config:
			locationConfig.Type = jobs.S3BackupLocationType
			locationConfig.S3Config = &jobs.S3LocationConfig{
				Endpoint:     cfg.S3Config.Endpoint,
				AccessKey:    cfg.S3Config.AccessKey,
				SecretKey:    cfg.S3Config.SecretKey,
				BucketName:   cfg.S3Config.BucketName,
				BucketRegion: cfg.S3Config.BucketRegion,
			}
		case *agentv1.StartJobRequest_MongoDBRestoreBackup_FilesystemConfig:
			locationConfig.Type = jobs.FilesystemBackupLocationType
			locationConfig.FilesystemStorageConfig = &jobs.FilesystemBackupLocationConfig{
				Path: cfg.FilesystemConfig.Path,
			}
		default:
			return errors.Errorf("unknown location config: %T", j.MongodbRestoreBackup.LocationConfig)
		}

		dsn, err := c.getMongoDSN(j.MongodbRestoreBackup.Dsn, j.MongodbRestoreBackup.TextFiles, p.JobId)
		if err != nil {
			return errors.WithStack(err)
		}

		job = jobs.NewMongoDBRestoreJob(p.JobId, timeout, j.MongodbRestoreBackup.Name,
			j.MongodbRestoreBackup.PitrTimestamp.AsTime(), dsn, locationConfig,
			c.supervisor, j.MongodbRestoreBackup.Folder, j.MongodbRestoreBackup.PbmMetadata.Name)
	default:
		return errors.Errorf("unknown job type: %T", j)
	}

	return c.runner.StartJob(job)
}

func (c *Client) getMongoDSN(dsn string, files *agentv1.TextFiles, jobID string) (string, error) {
	tempDir := filepath.Join(c.cfg.Get().Paths.TempDir, "mongodb-backup-restore", strings.ReplaceAll(jobID, "/", "_"))
	res, err := templates.RenderDSN(dsn, files, tempDir)
	defer templates.CleanupTempDir(tempDir, c.l)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// TODO following line is a quick patch. Come up with something better.
	res = strings.Replace(res, "directConnection=true", "directConnection=false", 1)

	return res, nil
}

func (c *Client) agentLogByID(agentID string, limit uint32) ([]string, uint) {
	var (
		logs     []string
		capacity uint
	)

	if c.cfg.Get().ID == agentID {
		logs, capacity = c.logStore.GetLogs()
	} else {
		logs, capacity = c.supervisor.AgentLogByID(agentID)
	}

	if limit > 0 && len(logs) > int(limit) {
		logs = logs[len(logs)-int(limit):]
	}

	for i, log := range logs {
		logs[i] = strings.TrimSuffix(log, "\n")
	}

	return logs, capacity
}

type dialResult struct {
	conn         *grpc.ClientConn
	streamCancel context.CancelFunc
	channel      *channel.Channel
	md           *agentv1.ServerConnectMetadata
}

// dial tries to connect to the server once.
// State changes are logged via l. Returned error is not user-visible.
func dial(dialCtx context.Context, cfg *config.Config, l *logrus.Entry) (*dialResult, error) {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithUserAgent("pmm-agent/" + version.Version),
	}
	if cfg.Server.WithoutTLS {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		host, _, _ := net.SplitHostPort(cfg.Server.Address)
		tlsConfig := tlsconfig.Get()
		tlsConfig.ServerName = host
		tlsConfig.InsecureSkipVerify = cfg.Server.InsecureTLS
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}

	if cfg.Server.Username != "" {
		if cfg.Server.Username == "service_token" || cfg.Server.Username == "api_key" {
			opts = append(opts, grpc.WithPerRPCCredentials(&tokenAuth{
				token: cfg.Server.Password,
			}))
		} else {
			opts = append(opts, grpc.WithPerRPCCredentials(&basicAuth{
				username: cfg.Server.Username,
				password: cfg.Server.Password,
			}))
		}
	}

	l.Infof("Connecting to %s ...", cfg.Server.FilteredURL())
	conn, err := grpc.DialContext(dialCtx, cfg.Server.Address, opts...)
	if err != nil {
		msg := err.Error()

		// improve error message in that particular case
		if errors.Is(err, context.DeadlineExceeded) {
			msg = "timeout"
		}

		l.Errorf("Failed to connect to %s: %s.", cfg.Server.Address, msg)
		return nil, errors.Wrap(err, "failed to dial")
	}
	l.Infof("Connected to %s.", cfg.Server.Address)

	// gRPC stream is created without lifetime timeout.
	// However, we need to cancel it if two-way communication channel can't be established
	// when pmm-managed is down. A separate timer is used for that.
	streamCtx, streamCancel := context.WithCancel(context.Background())
	teardown := func() {
		streamCancel()
		if err := conn.Close(); err != nil {
			l.Debugf("Connection closed: %s.", err)
			return
		}
		l.Debugf("Connection closed.")
	}
	d, ok := dialCtx.Deadline()
	if !ok {
		panic("no deadline in dialCtx")
	}
	streamCancelT := time.AfterFunc(time.Until(d), streamCancel)
	defer streamCancelT.Stop()

	l.Info("Establishing two-way communication channel ...")
	start := time.Now()
	streamCtx = agentv1.AddAgentConnectMetadata(streamCtx, &agentv1.AgentConnectMetadata{
		ID:      cfg.ID,
		Version: version.Version,
	})
	stream, err := agentv1.NewAgentServiceClient(conn).Connect(streamCtx) //nolint:contextcheck
	if err != nil {
		l.Errorf("Failed to establish two-way communication channel: %s.", err)
		teardown()
		return nil, errors.Wrap(err, "failed to connect")
	}

	// So far, nginx can handle all that itself without pmm-managed.
	// We need to exchange one pair of messages (ping/pong) for metadata headers to reach pmm-managed
	// to ensure that pmm-managed is alive and that Agent ID is valid.

	channel := channel.New(stream)
	_, clockDrift, err := getNetworkInformation(channel) // ping/pong
	if err != nil {
		msg := err.Error()

		// improve error message
		if s, ok := grpcstatus.FromError(errors.Cause(err)); ok {
			msg = strings.TrimSuffix(s.Message(), ".")
		}

		l.Errorf("Failed to establish two-way communication channel: %s.", msg)
		teardown()
		return nil, err
	}

	// read metadata header after receiving pong
	md, err := agentv1.ReceiveServerConnectMetadata(stream)
	l.Debugf("Received server metadata: %+v. Error: %+v.", md, err)
	if err != nil {
		l.Errorf("Failed to receive server metadata: %s.", err)
		teardown()
		return nil, errors.Wrap(err, "failed to receive server metadata")
	}
	if md.ServerVersion == "" {
		l.Errorf("Server metadata does not contain server version.")
		teardown()
		return nil, errors.New("empty server version in metadata")
	}

	level := logrus.InfoLevel
	if clockDrift > clockDriftWarning || -clockDrift > clockDriftWarning {
		level = logrus.WarnLevel
	}
	l.Logf(level, "Two-way communication channel established in %s. Estimated clock drift: %s.",
		time.Since(start), clockDrift)

	return &dialResult{
		conn:         conn,
		streamCancel: streamCancel,
		channel:      channel,
		md:           md,
	}, nil
}

func getNetworkInformation(channel *channel.Channel) (latency, clockDrift time.Duration, err error) { //nolint:nonamedreturns
	start := time.Now()
	var resp agentv1.ServerResponsePayload
	resp, err = channel.SendAndWaitResponse(&agentv1.Ping{})
	if err != nil {
		return
	}
	if resp == nil {
		err = channel.Wait()
		return
	}
	roundtrip := time.Since(start)
	currentTime := resp.(*agentv1.Pong).CurrentTime //nolint:forcetypeassert
	serverTime := currentTime.AsTime()
	err = currentTime.CheckValid()
	if err != nil {
		err = errors.Wrap(err, "Failed to decode Ping")
		return
	}
	latency = roundtrip / 2
	clockDrift = serverTime.Sub(start) - latency
	return
}

// GetNetworkInformation sends ping request to the server and returns info about latency and clock drift.
func (c *Client) GetNetworkInformation() (latency, clockDrift time.Duration, err error) { //nolint:nonamedreturns
	c.rw.RLock()
	channel := c.channel
	c.rw.RUnlock()
	if channel == nil {
		err = errors.New("not connected")
		return
	}

	latency, clockDrift, err = getNetworkInformation(channel)
	return
}

// GetServerConnectMetadata returns current server's metadata, or nil.
func (c *Client) GetServerConnectMetadata() *agentv1.ServerConnectMetadata {
	c.rw.RLock()
	md := c.md
	c.rw.RUnlock()
	return md
}

// GetConnectionUpTime returns connection uptime between agent and server in percentage (from 0 to 100).
func (c *Client) GetConnectionUpTime() float32 {
	return c.cus.GetConnectedUpTimeUntil(time.Now())
}

// Describe implements "unchecked" prometheus.Collector.
func (c *Client) Describe(chan<- *prometheus.Desc) {
	// Sending no descriptor at all marks the Collector as “unchecked”,
	// i.e. no checks will be performed at registration time, and the
	// Collector may yield any Metric it sees fit in its Collect method.
}

// Collect implements "unchecked" prometheus.Collector.
func (c *Client) Collect(ch chan<- prometheus.Metric) {
	c.rw.RLock()
	channel := c.channel
	c.rw.RUnlock()

	desc := prometheus.NewDesc("pmm_agent_connected", "Has value 1 if two-way communication channel is established.", nil, nil)
	if channel != nil {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 1)
		channel.Collect(ch)
	} else {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 0)
	}
	c.supervisor.Collect(ch)
}

// argListFromPgParams creates an array of strings from the pointer to the parameters for pt-pg-sumamry.
func argListFromPgParams(pParams *agentv1.StartActionRequest_PTPgSummaryParams) []string {
	var args []string

	if pParams.Host != "" {
		args = append(args, "--host", pParams.Host)
	}

	if pParams.Port > 0 && pParams.Port <= 65535 {
		args = append(args, "--port", strconv.Itoa(int(pParams.Port)))
	}

	if pParams.Username != "" {
		args = append(args, "--username", pParams.Username)
	}

	pswd := strings.TrimSpace(pParams.Password)
	if pswd != "" {
		args = append(args, "--password", pswd)
	}

	return args
}

// argListFromMongoDBParams creates an array of strings from the pointer to the parameters for pt-mongodb-sumamry.
func argListFromMongoDBParams(pParams *agentv1.StartActionRequest_PTMongoDBSummaryParams) []string {
	var args []string

	// Only adds the arguments are valid

	if pParams.Username != "" {
		args = append(args, "--username", pParams.Username)
	}

	if pParams.Password != "" {
		// TODO change this line when pt-mongodb-summary is updated
		args = append(args, "--password="+pParams.Password)
	}

	if pParams.Host != "" {
		hostPortStr := pParams.Host

		// If valid port attaches ':' and the port number after address
		if pParams.Port > 0 && pParams.Port <= 65535 {
			hostPortStr += ":" + strconv.Itoa(int(pParams.Port))
		}

		args = append(args, hostPortStr)
	}

	return args
}

func convertAgentErrorToGrpcStatus(agentErr error) *grpcstatus.Status {
	var status *grpcstatus.Status
	switch {
	case errors.Is(agentErr, agenterrors.ErrInvalidArgument):
		status = grpcstatus.New(codes.InvalidArgument, agentErr.Error())
	case errors.Is(agentErr, agenterrors.ErrActionQueueOverflow):
		status = grpcstatus.New(codes.ResourceExhausted, agentErr.Error())
	case errors.Is(agentErr, agenterrors.ErrActionUnimplemented):
		status = grpcstatus.New(codes.Unimplemented, agentErr.Error())
	default:
		status = grpcstatus.New(codes.Internal, agentErr.Error())
	}
	return status
}

// check interface.
var (
	_ prometheus.Collector = (*Client)(nil)
)
