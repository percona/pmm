// pmm-agent
// Copyright (C) 2018 Percona LLC
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

// Package client contains business logic of working with pmm-managed.
package client

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-agent/client/channel"
	"github.com/percona/pmm-agent/config"
	"github.com/percona/pmm-agent/utils/backoff"
)

var (
	dialTimeout = 5 * time.Second // changed by unit tests
)

const (
	backoffMinDelay   = 1 * time.Second
	backoffMaxDelay   = 15 * time.Second
	clockDriftWarning = 5 * time.Second
)

// Client represents pmm-agent's connection to nginx/pmm-managed.
type Client struct {
	cfg        *config.Config
	supervisor supervisor
	withoutTLS bool // only for unit tests

	l       *logrus.Entry
	backoff *backoff.Backoff
	done    chan struct{}

	rw      sync.RWMutex
	md      *agentpb.AgentServerMetadata
	channel *channel.Channel
}

// New creates new client.
//
// Caller should call Run.
func New(cfg *config.Config, supervisor supervisor) *Client {
	return &Client{
		cfg:        cfg,
		supervisor: supervisor,
		l:          logrus.WithField("component", "client"),
		backoff:    backoff.New(backoffMinDelay, backoffMaxDelay),
		done:       make(chan struct{}),
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
		dialCtx, dialCancel := context.WithTimeout(ctx, dialTimeout)
		dialResult, dialErr = dial(dialCtx, c.cfg, c.withoutTLS, c.l)
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
	c.md = &dialResult.md
	c.channel = dialResult.channel
	c.rw.Unlock()

	// Once the client is connected, ctx cancellation is ignored.
	// We start two goroutines, and terminate the gRPC connection and exit Run when any of them exits:
	// 1. processSupervisorRequests reads requests (status changes and QAN data) from the supervisor and sends them to the channel.
	//    It exits when the supervisor is stopped.
	//    When the gRPC connection is terminated on exiting Run, processChannelRequests exits too.
	// 2. processChannelRequests reads requests from the channel and processes them.
	//    It exits when an unexpected message is received from the channel, or when can't be received at all.
	//    When Run is left, caller stops supervisor, and that allows processSupervisorRequests to exit.
	// Done() channel is closed when both goroutines exited.
	oneDone := make(chan struct{}, 2)
	go func() {
		c.processSupervisorRequests()
		oneDone <- struct{}{}
	}()
	go func() {
		c.processChannelRequests()
		oneDone <- struct{}{}
	}()
	<-oneDone
	go func() {
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

func (c *Client) processSupervisorRequests() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		for state := range c.supervisor.Changes() {
			resp := c.channel.SendRequest(&state)
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
			resp := c.channel.SendRequest(&collect)
			if resp == nil {
				c.l.Warn("Failed to send QanCollect request.")
			}
		}
		c.l.Debugf("Supervisor QANRequests() channel drained.")
	}()

	wg.Wait()
}

func (c *Client) processChannelRequests() {
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
			panic("TODO")

		case *agentpb.StopActionRequest:
			panic("TODO")

		case nil:
			// Requests() is not closed, so exit early to break channel
			c.l.Errorf("Unhandled server request: %v.", req)
			return
		}

		c.channel.SendResponse(&channel.AgentResponse{
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

type dialResult struct {
	conn         *grpc.ClientConn
	streamCancel context.CancelFunc
	channel      *channel.Channel
	md           agentpb.AgentServerMetadata
}

// dial tries to connect to the server once.
// State changes are logged via l. Returned error is not user-visible.
func dial(dialCtx context.Context, cfg *config.Config, withoutTLS bool, l *logrus.Entry) (*dialResult, error) {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithUserAgent("pmm-agent/" + version.Version),
	}
	if withoutTLS {
		opts = append(opts, grpc.WithInsecure())
	} else {
		host, _, _ := net.SplitHostPort(cfg.Server.Address)
		tlsConfig := &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: cfg.Server.InsecureTLS, //nolint:gosec
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}

	// FIXME https://jira.percona.com/browse/PMM-3867
	// https://github.com/grpc/grpc-go/issues/106#issuecomment-246978683
	// https://jbrandhorst.com/post/grpc-auth/
	if cfg.Server.Username != "" {
		logrus.Panic("PMM Server authentication is not implemented yet.")
	}

	l.Infof("Connecting to %s ...", cfg.Server.Address)
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

	// So far nginx can handle all that itself without pmm-managed.
	// We need to exchange metadata and one pair of messages (ping/pong)
	// to ensure that pmm-managed is alive and that Agent ID is valid.

	md, err := agentpb.GetAgentServerMetadata(stream)
	if err != nil {
		msg := err.Error()

		// improve error message in that particular case
		if code := status.Code(err); code == codes.DeadlineExceeded || code == codes.Canceled {
			msg = "timeout"
		}

		l.Errorf("Can't get server metadata: %s.", msg)
		teardown()
		return nil, errors.Wrap(err, "failed to get server metadata")
	}

	channel := channel.New(stream)
	_, clockDrift, err := getNetworkInformation(channel)
	if err != nil {
		l.Errorf("Failed to get network information: %s.", err)
		teardown()
		return nil, err
	}
	l.Infof("Two-way communication channel established in %s.", time.Since(start))
	streamCancelT.Stop()

	if clockDrift > clockDriftWarning || -clockDrift > clockDriftWarning {
		l.Warnf("Estimated clock drift: %s.", clockDrift)
	}

	return &dialResult{conn, streamCancel, channel, md}, nil
}

func getNetworkInformation(channel *channel.Channel) (latency, clockDrift time.Duration, err error) {
	start := time.Now()
	resp := channel.SendRequest(new(agentpb.Ping))
	if resp == nil {
		err = errors.Wrap(channel.Wait(), "Failed to send Ping")
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

// GetAgentServerMetadata returns current server's metadata, or nil.
func (c *Client) GetAgentServerMetadata() *agentpb.AgentServerMetadata {
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

// check interface
var (
	_ prometheus.Collector = (*Client)(nil)
)
