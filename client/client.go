// pmm-agent
// Copyright 2019 Percona LLC
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
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/utils/tlsconfig"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-agent/actions" // TODO https://jira.percona.com/browse/PMM-7206
	"github.com/percona/pmm-agent/client/channel"
	"github.com/percona/pmm-agent/config"
	"github.com/percona/pmm-agent/jobs"
	"github.com/percona/pmm-agent/utils/backoff"
)

const (
	dialTimeout          = 5 * time.Second
	backoffMinDelay      = 1 * time.Second
	backoffMaxDelay      = 15 * time.Second
	clockDriftWarning    = 5 * time.Second
	defaultActionTimeout = 10 * time.Second // default timeout for compatibility with an older server
)

// Client represents pmm-agent's connection to nginx/pmm-managed.
type Client struct {
	cfg               *config.Config
	supervisor        supervisor
	connectionChecker connectionChecker

	l       *logrus.Entry
	backoff *backoff.Backoff
	done    chan struct{}

	// for unit tests only
	dialTimeout time.Duration

	actionsRunner *actions.ConcurrentRunner
	jobsRunner    *jobs.Runner

	rw      sync.RWMutex
	md      *agentpb.ServerConnectMetadata
	channel *channel.Channel
}

// New creates new client.
//
// Caller should call Run.
func New(cfg *config.Config, supervisor supervisor, connectionChecker connectionChecker) *Client {
	return &Client{
		cfg:               cfg,
		supervisor:        supervisor,
		connectionChecker: connectionChecker,
		l:                 logrus.WithField("component", "client"),
		backoff:           backoff.New(backoffMinDelay, backoffMaxDelay),
		done:              make(chan struct{}),
		dialTimeout:       dialTimeout,
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

	c.actionsRunner = actions.NewConcurrentRunner(ctx)
	c.jobsRunner = jobs.NewRunner()

	// do nothing until ctx is canceled if config misses critical info
	var missing string
	if c.cfg.ID == "" {
		missing = "Agent ID"
	}
	if c.cfg.Server.Address == "" {
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
		dialResult, dialErr = dial(dialCtx, c.cfg, c.l)
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

	oneDone := make(chan struct{}, 5)
	go func() {
		c.jobsRunner.Run(ctx)
		oneDone <- struct{}{}
	}()
	go func() {
		c.processActionResults()
		oneDone <- struct{}{}
	}()
	go func() {
		c.processJobsResults()
		oneDone <- struct{}{}
	}()
	go func() {
		c.processSupervisorRequests()
		oneDone <- struct{}{}
	}()
	go func() {
		c.processChannelRequests(ctx)
		oneDone <- struct{}{}
	}()

	<-oneDone
	go func() {
		<-oneDone
		<-oneDone
		<-oneDone
		<-oneDone
		c.l.Info("Done.")
		close(c.done)
	}()
	return nil
}

// Done is closed when all supervisors's requests are sent (if possible) and connection is closed.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

func (c *Client) processActionResults() {
	for result := range c.actionsRunner.Results() {
		resp := c.channel.SendAndWaitResponse(&agentpb.ActionResultRequest{
			ActionId: result.ID,
			Output:   result.Output,
			Done:     true,
			Error:    result.Error,
		})
		if resp == nil {
			c.l.Warn("Failed to send ActionResult request.")
		}
	}
	c.l.Debugf("Actions runner Results() channel drained.")
}

func (c *Client) processJobsResults() {
	for message := range c.jobsRunner.Messages() {
		c.channel.Send(message)
	}
	c.l.Debugf("Jobs runner Messages() channel drained.")
}

func (c *Client) processSupervisorRequests() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		for state := range c.supervisor.Changes() {
			resp := c.channel.SendAndWaitResponse(state)
			if resp == nil {
				c.l.Warn("Failed to send StateChanged request.")
			}
		}
		c.l.Debugf("Supervisor Changes() channel drained.")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for collect := range c.supervisor.QANRequests() {
			resp := c.channel.SendAndWaitResponse(collect)
			if resp == nil {
				c.l.Warn("Failed to send QanCollect request.")
			}
		}
		c.l.Debugf("Supervisor QANRequests() channel drained.")
	}()

	wg.Wait()
}

func (c *Client) processChannelRequests(ctx context.Context) {
	for req := range c.channel.Requests() {
		var responsePayload agentpb.AgentResponsePayload
		switch p := req.Payload.(type) {
		case *agentpb.Ping:
			responsePayload = &agentpb.Pong{
				CurrentTime: ptypes.TimestampNow(),
			}

		case *agentpb.SetStateRequest:
			c.supervisor.SetState(p)
			responsePayload = new(agentpb.SetStateResponse)

		case *agentpb.StartActionRequest:
			var action actions.Action
			switch params := p.Params.(type) {
			case *agentpb.StartActionRequest_MysqlExplainParams:
				action = actions.NewMySQLExplainAction(p.ActionId, params.MysqlExplainParams)

			case *agentpb.StartActionRequest_MysqlShowCreateTableParams:
				action = actions.NewMySQLShowCreateTableAction(p.ActionId, params.MysqlShowCreateTableParams)

			case *agentpb.StartActionRequest_MysqlShowTableStatusParams:
				action = actions.NewMySQLShowTableStatusAction(p.ActionId, params.MysqlShowTableStatusParams)

			case *agentpb.StartActionRequest_MysqlShowIndexParams:
				action = actions.NewMySQLShowIndexAction(p.ActionId, params.MysqlShowIndexParams)

			case *agentpb.StartActionRequest_PostgresqlShowCreateTableParams:
				action = actions.NewPostgreSQLShowCreateTableAction(p.ActionId, params.PostgresqlShowCreateTableParams)

			case *agentpb.StartActionRequest_PostgresqlShowIndexParams:
				action = actions.NewPostgreSQLShowIndexAction(p.ActionId, params.PostgresqlShowIndexParams)

			case *agentpb.StartActionRequest_MongodbExplainParams:
				action = actions.NewMongoDBExplainAction(p.ActionId, params.MongodbExplainParams, c.cfg.Paths.TempDir)

			case *agentpb.StartActionRequest_MysqlQueryShowParams:
				action = actions.NewMySQLQueryShowAction(p.ActionId, params.MysqlQueryShowParams)

			case *agentpb.StartActionRequest_MysqlQuerySelectParams:
				action = actions.NewMySQLQuerySelectAction(p.ActionId, params.MysqlQuerySelectParams)

			case *agentpb.StartActionRequest_PostgresqlQueryShowParams:
				action = actions.NewPostgreSQLQueryShowAction(p.ActionId, params.PostgresqlQueryShowParams)

			case *agentpb.StartActionRequest_PostgresqlQuerySelectParams:
				action = actions.NewPostgreSQLQuerySelectAction(p.ActionId, params.PostgresqlQuerySelectParams)

			case *agentpb.StartActionRequest_MongodbQueryGetparameterParams:
				action = actions.NewMongoDBQueryAdmincommandAction(actions.MongoDBQueryAdmincommandActionParams{
					ID:      p.ActionId,
					DSN:     params.MongodbQueryGetparameterParams.Dsn,
					Files:   params.MongodbQueryGetparameterParams.TextFiles,
					Command: "getParameter",
					Arg:     "*",
					TempDir: c.cfg.Paths.TempDir,
				})

			case *agentpb.StartActionRequest_MongodbQueryBuildinfoParams:
				action = actions.NewMongoDBQueryAdmincommandAction(actions.MongoDBQueryAdmincommandActionParams{
					ID:      p.ActionId,
					DSN:     params.MongodbQueryBuildinfoParams.Dsn,
					Files:   params.MongodbQueryBuildinfoParams.TextFiles,
					Command: "buildInfo",
					Arg:     1,
					TempDir: c.cfg.Paths.TempDir,
				})

			case *agentpb.StartActionRequest_MongodbQueryGetcmdlineoptsParams:
				action = actions.NewMongoDBQueryAdmincommandAction(actions.MongoDBQueryAdmincommandActionParams{
					ID:      p.ActionId,
					DSN:     params.MongodbQueryGetcmdlineoptsParams.Dsn,
					Files:   params.MongodbQueryGetcmdlineoptsParams.TextFiles,
					Command: "getCmdLineOpts",
					Arg:     1,
					TempDir: c.cfg.Paths.TempDir,
				})

			case *agentpb.StartActionRequest_PtSummaryParams:
				action = actions.NewProcessAction(p.ActionId, c.cfg.Paths.PTSummary, []string{})

			case *agentpb.StartActionRequest_PtPgSummaryParams:
				action = actions.NewProcessAction(p.ActionId, c.cfg.Paths.PTPgSummary, argListFromPgParams(params.PtPgSummaryParams))

			case *agentpb.StartActionRequest_PtMysqlSummaryParams:
				action = actions.NewPTMySQLSummaryAction(p.ActionId, c.cfg.Paths.PTMySqlSummary, params.PtMysqlSummaryParams)

			case *agentpb.StartActionRequest_PtMongodbSummaryParams:
				action = actions.NewProcessAction(p.ActionId, c.cfg.Paths.PTMongoDBSummary, argListFromMongoDBParams(params.PtMongodbSummaryParams))

			case nil:
				// Requests() is not closed, so exit early to break channel
				c.l.Errorf("Unhandled StartAction request: %v.", req)
				return
			}

			c.actionsRunner.Start(action, c.getActionTimeout(p))
			responsePayload = new(agentpb.StartActionResponse)

		case *agentpb.StopActionRequest:
			c.actionsRunner.Stop(p.ActionId)
			responsePayload = new(agentpb.StopActionResponse)

		case *agentpb.CheckConnectionRequest:
			responsePayload = c.connectionChecker.Check(ctx, p, req.ID)

		case *agentpb.StartJobRequest:
			var resp agentpb.StartJobResponse
			if err := c.handleStartJobRequest(p); err != nil {
				resp.Error = err.Error()
			}
			responsePayload = &resp

		case *agentpb.StopJobRequest:
			c.jobsRunner.Stop(p.JobId)
			responsePayload = new(agentpb.StopJobResponse)

		case *agentpb.JobStatusRequest:
			alive := c.jobsRunner.IsRunning(p.JobId)
			responsePayload = &agentpb.JobStatusResponse{Alive: alive}

		case nil:
			// Requests() is not closed, so exit early to break channel
			c.l.Errorf("Unhandled server request: %v.", req)
			return
		}

		c.channel.Send(&channel.AgentResponse{
			ID:      req.ID,
			Payload: responsePayload,
		})
	}

	if err := c.channel.Wait(); err != nil {
		c.l.Debugf("Channel closed: %s.", err)
		return
	}
	c.l.Debug("Channel closed.")
}

func (c *Client) handleStartJobRequest(p *agentpb.StartJobRequest) error {
	timeout, err := ptypes.Duration(p.Timeout)
	if err != nil {
		return err
	}

	var job jobs.Job
	switch j := p.Job.(type) {
	case *agentpb.StartJobRequest_Echo_:
		delay, err := ptypes.Duration(j.Echo.Delay)
		if err != nil {
			return err
		}

		job = jobs.NewEchoJob(p.JobId, timeout, j.Echo.Message, delay)
	case *agentpb.StartJobRequest_MysqlBackup:
		var locationConfig jobs.BackupLocationConfig
		switch cfg := j.MysqlBackup.LocationConfig.(type) {
		case *agentpb.StartJobRequest_MySQLBackup_S3Config:
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

		cfg := jobs.DatabaseConfig{
			User:     j.MysqlBackup.User,
			Password: j.MysqlBackup.Password,
			Address:  j.MysqlBackup.Address,
			Port:     int(j.MysqlBackup.Port),
			Socket:   j.MysqlBackup.Socket,
		}
		job = jobs.NewMySQLBackupJob(p.JobId, timeout, j.MysqlBackup.Name, cfg, locationConfig)
	default:
		return errors.Errorf("unknown job type: %T", j)
	}

	c.jobsRunner.Start(job)

	return nil
}

func (c *Client) getActionTimeout(req *agentpb.StartActionRequest) time.Duration {
	d, err := ptypes.Duration(req.Timeout)
	if err == nil && d == 0 {
		err = errors.New("timeout can't be zero")
	}
	if err != nil {
		c.l.Warnf("Invalid timeout, using default value instead: %s.", err)
		d = defaultActionTimeout
	}
	return d
}

type dialResult struct {
	conn         *grpc.ClientConn
	streamCancel context.CancelFunc
	channel      *channel.Channel
	md           *agentpb.ServerConnectMetadata
}

// dial tries to connect to the server once.
// State changes are logged via l. Returned error is not user-visible.
func dial(dialCtx context.Context, cfg *config.Config, l *logrus.Entry) (*dialResult, error) {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithUserAgent("pmm-agent/" + version.Version),
	}
	if cfg.Server.WithoutTLS {
		opts = append(opts, grpc.WithInsecure())
	} else {
		host, _, _ := net.SplitHostPort(cfg.Server.Address)
		tlsConfig := tlsconfig.Get()
		tlsConfig.ServerName = host
		tlsConfig.InsecureSkipVerify = cfg.Server.InsecureTLS //nolint:gosec
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}

	if cfg.Server.Username != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(&basicAuth{
			username: cfg.Server.Username,
			password: cfg.Server.Password,
		}))
	}

	l.Infof("Connecting to %s ...", cfg.Server.FilteredURL())
	conn, err := grpc.DialContext(dialCtx, cfg.Server.Address, opts...)
	if err != nil {
		msg := err.Error()

		// improve error message in that particular case
		if err == context.DeadlineExceeded {
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
	streamCtx = agentpb.AddAgentConnectMetadata(streamCtx, &agentpb.AgentConnectMetadata{
		ID:      cfg.ID,
		Version: version.Version,
	})
	stream, err := agentpb.NewAgentClient(conn).Connect(streamCtx)
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
		if s, ok := status.FromError(errors.Cause(err)); ok {
			msg = strings.TrimSuffix(s.Message(), ".")
		}

		l.Errorf("Failed to establish two-way communication channel: %s.", msg)
		teardown()
		return nil, err
	}

	// read metadata header after receiving pong
	md, err := agentpb.ReceiveServerConnectMetadata(stream)
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
		md:           md}, nil
}

func getNetworkInformation(channel *channel.Channel) (latency, clockDrift time.Duration, err error) {
	start := time.Now()
	resp := channel.SendAndWaitResponse(new(agentpb.Ping))
	if resp == nil {
		err = channel.Wait()
		return
	}
	roundtrip := time.Since(start)
	serverTime, err := ptypes.Timestamp(resp.(*agentpb.Pong).CurrentTime)
	if err != nil {
		err = errors.Wrap(err, "Failed to decode Ping")
		return
	}
	latency = roundtrip / 2
	clockDrift = serverTime.Sub(start) - latency
	return
}

// GetNetworkInformation sends ping request to the server and returns info about latency and clock drift.
func (c *Client) GetNetworkInformation() (latency, clockDrift time.Duration, err error) {
	c.rw.RLock()
	channel := c.channel
	c.rw.RUnlock()
	if channel == nil {
		err = errors.New("not connected")
		return
	}

	latency, clockDrift, err = getNetworkInformation(c.channel)
	return
}

// GetServerConnectMetadata returns current server's metadata, or nil.
func (c *Client) GetServerConnectMetadata() *agentpb.ServerConnectMetadata {
	c.rw.RLock()
	md := c.md
	c.rw.RUnlock()
	return md
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
}

// argListFromPgParams creates an array of strings from the pointer to the parameters for pt-pg-sumamry
func argListFromPgParams(pParams *agentpb.StartActionRequest_PTPgSummaryParams) []string {
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

// argListFromMongoDBParams creates an array of strings from the pointer to the parameters for pt-mongodb-sumamry
func argListFromMongoDBParams(pParams *agentpb.StartActionRequest_PTMongoDBSummaryParams) []string {
	var args []string

	// Only adds the arguments are valid

	if pParams.Username != "" {
		args = append(args, "--username", pParams.Username)
	}

	if pParams.Password != "" {
		// TODO change this line when pt-mongodb-summary is updated
		args = append(args, fmt.Sprintf("--password=%s", pParams.Password))
	}

	if pParams.Host != "" {
		var hostPortStr string = pParams.Host

		// If valid port attaches ':' and the port number after address
		if pParams.Port > 0 && pParams.Port <= 65535 {
			hostPortStr += ":" + strconv.Itoa(int(pParams.Port))
		}

		args = append(args, hostPortStr)
	}

	return args
}

// check interface
var (
	_ prometheus.Collector = (*Client)(nil)
)
