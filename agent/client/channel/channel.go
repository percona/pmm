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

// Package channel contains protocol details of two-way communication channel between pmm-managed and pmm-agent.
package channel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	protostatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/utils/logger"
)

const (
	serverRequestsCap = 32

	// One slot is enough for the Ping queue: a ping carries no state and a new one arrives
	// every few seconds, so a ping that finds the slot taken is stale. See PMM-15431.
	serverPingsCap = 1

	prometheusNamespace = "pmm_agent"
	prometheusSubsystem = "channel"
)

// How long runReceiver waits for room in the request queue before giving up on the connection.
// Reaching it means the consumer of Requests() has stopped draining, which no amount of further
// waiting fixes - see runReceiver. Overridden in tests.
var requestQueueStuckTimeout = 2 * time.Minute

// ServerRequest represents a request from server.
// It is similar to agentv1.ServerMessage except it can contain only requests,
// and the payload is already unwrapped (XXX instead of ServerMessage_XXX).
type ServerRequest struct {
	ID      uint32
	Payload agentv1.ServerRequestPayload
}

// AgentResponse represents agent's response.
// It is similar to agentv1.AgentMessage except it can contain only responses,
// and the payload is already unwrapped (XXX instead of AgentMessage_XXX).
type AgentResponse struct {
	ID      uint32
	Status  *grpcstatus.Status
	Payload agentv1.AgentResponsePayload
}

// Response is a type used to pass response from pmm-server to the subscriber.
type Response struct {
	Payload agentv1.ServerResponsePayload
	Error   error
}

// Channel encapsulates two-way communication channel between pmm-managed and pmm-agent.
//
// All exported methods are thread-safe.
type Channel struct {
	s agentv1.AgentService_ConnectClient
	l *logrus.Entry

	mRecv, mSend prometheus.Counter

	lastSentRequestID atomic.Uint32

	sendM sync.Mutex

	m         sync.Mutex
	responses map[uint32]chan Response
	requests  chan *ServerRequest
	pings     chan *ServerRequest

	closeOnce sync.Once
	closeWait chan struct{}
	closeErr  error
}

// New creates new two-way communication channel with given stream.
//
// Stream should not be used by the caller after channel is created.
func New(stream agentv1.AgentService_ConnectClient) *Channel {
	s := &Channel{
		s: stream,
		l: logrus.WithField("component", "channel"), // only for debug logging

		mRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "messages_received_total",
			Help:      "A total number of received messages from pmm-managed.",
		}),
		mSend: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "messages_sent_total",
			Help:      "A total number of sent messages to pmm-managed.",
		}),

		responses: make(map[uint32]chan Response),
		requests:  make(chan *ServerRequest, serverRequestsCap),
		pings:     make(chan *ServerRequest, serverPingsCap),

		closeWait: make(chan struct{}),
	}

	go s.runReceiver()
	return s
}

// close marks channel as closed with given error - only once.
func (c *Channel) close(err error) {
	c.closeOnce.Do(func() {
		c.l.Debugf("Closing with error: %+v", err)
		c.closeErr = err

		c.m.Lock()
		for _, ch := range c.responses { // unblock all subscribers
			close(ch)
		}
		c.responses = nil // prevent future subscriptions
		c.m.Unlock()

		c.sendM.Lock()
		_ = c.s.CloseSend()
		close(c.closeWait)
		c.sendM.Unlock()
	})
}

// Wait blocks until channel is closed and returns the reason why it was closed.
//
// When Wait returns, underlying gRPC connection should be terminated to prevent goroutine leak.
func (c *Channel) Wait() error {
	<-c.closeWait
	return c.closeErr
}

// Requests returns a channel for incoming requests. It must be read. It is closed on any error (see Wait).
func (c *Channel) Requests() <-chan *ServerRequest {
	return c.requests
}

// Pings returns a channel for incoming Ping requests. It is closed on any error (see Wait).
//
// Unlike Requests() this queue is dropped on overflow rather than waited on, which is the whole
// point of keeping pings out of Requests(): a pong cannot end up queued behind a request the
// consumer has not got to yet. See PMM-15431.
func (c *Channel) Pings() <-chan *ServerRequest {
	return c.pings
}

// Send sends message to pmm-managed. It is no-op once channel is closed (see Wait).
func (c *Channel) Send(resp *AgentResponse) {
	msg := &agentv1.AgentMessage{
		Id: resp.ID,
	}
	if resp.Payload != nil {
		msg.Payload = resp.Payload.AgentMessageResponsePayload()
	}
	if resp.Status != nil {
		msg.Status = resp.Status.Proto()
	}
	c.send(msg)
}

// SendAndWaitResponse sends request to pmm-managed and blocks until the response is available,
// the channel is closed, or ctx is done, whichever happens first.
// If error occurred - subscription got canceled - returned payload is nil and error contains reason for cancelation.
// Response and error will be both nil if channel is closed.
// It is no-op once channel is closed (see Wait).
func (c *Channel) SendAndWaitResponse(ctx context.Context, payload agentv1.AgentRequestPayload) (agentv1.ServerResponsePayload, error) { //nolint:ireturn,nolintlint
	id := c.lastSentRequestID.Add(1)
	ch := c.subscribe(id)

	c.send(&agentv1.AgentMessage{
		Id:      id,
		Payload: payload.AgentMessageRequestPayload(),
	})

	select {
	case resp := <-ch:
		return resp.Payload, resp.Error

	case <-ctx.Done():
		// This is what bounds the callers that have a deadline of their own: the ping/pong
		// of the dial handshake, and pmm-admin status asking for network info. Both used to
		// wait for a response that a wedged connection would never deliver.
		// The subscription goes so that a response arriving later is not left waiting for a
		// reader; it is buffered, so a publisher that took the entry first still completes.
		c.removeResponseChannel(id)
		return nil, ctx.Err()
	}
}

func (c *Channel) send(msg *agentv1.AgentMessage) {
	c.sendM.Lock()
	select {
	case <-c.closeWait:
		c.sendM.Unlock()
		return
	default:
	}

	// Check log level before calling formatting function.
	// Do not waste resources in case debug level is not enabled.
	if c.l.Logger.IsLevelEnabled(logrus.DebugLevel) {
		// do not use default compact representation for large/complex messages
		if size := proto.Size(msg); size < 100 { //nolint:mnd
			c.l.Debugf("Sending message (%d bytes): %s.", size, logger.RedactMessage(msg))
		} else {
			c.l.Debugf("Sending message (%d bytes):\n%s\n", size, prototext.Format(logger.RedactMessage(msg)))
		}
	}

	err := c.s.Send(msg)
	c.sendM.Unlock()
	if err != nil {
		c.close(fmt.Errorf("failed to send message: %w", err))
		return
	}
	c.mSend.Inc()
}

// runReader receives messages from server.
func (c *Channel) runReceiver() {
	defer func() {
		close(c.requests)
		close(c.pings)
		c.l.Debug("Exiting receiver goroutine.")
	}()

	for {
		msg, err := c.s.Recv()
		if err != nil {
			c.close(fmt.Errorf("failed to receive message: %w", err))
			return
		}
		c.mRecv.Inc()

		// Check log level before calling formatting function.
		// Do not waste resources in case debug level is not enabled.
		if c.l.Logger.IsLevelEnabled(logrus.DebugLevel) {
			// do not use default compact representation for large/complex messages
			if size := proto.Size(msg); size < 100 { //nolint:mnd
				c.l.Debugf("Received message (%d bytes): %s.", size, logger.RedactMessage(msg))
			} else {
				c.l.Debugf("Received message (%d bytes):\n%s\n", size, prototext.Format(logger.RedactMessage(msg)))
			}
		}

		var req *ServerRequest
		switch p := msg.Payload.(type) {
		// requests
		case *agentv1.ServerMessage_Ping:
			// Answered from its own goroutine, see Pings. Dropping a ping that finds the
			// single slot taken costs only a warning on the server side, which is a much
			// smaller price than blocking here.
			select {
			case c.pings <- &ServerRequest{ID: msg.Id, Payload: p.Ping}:
			default:
				c.l.Warnf("Dropping ping %d: previous one is not answered yet.", msg.Id)
			}
		case *agentv1.ServerMessage_SetState:
			req = &ServerRequest{ID: msg.Id, Payload: p.SetState}
		case *agentv1.ServerMessage_StartAction:
			req = &ServerRequest{ID: msg.Id, Payload: p.StartAction}
		case *agentv1.ServerMessage_StopAction:
			req = &ServerRequest{ID: msg.Id, Payload: p.StopAction}
		case *agentv1.ServerMessage_CheckConnection:
			req = &ServerRequest{ID: msg.Id, Payload: p.CheckConnection}
		case *agentv1.ServerMessage_StartJob:
			req = &ServerRequest{ID: msg.Id, Payload: p.StartJob}
		case *agentv1.ServerMessage_StopJob:
			req = &ServerRequest{ID: msg.Id, Payload: p.StopJob}
		case *agentv1.ServerMessage_JobStatus:
			req = &ServerRequest{ID: msg.Id, Payload: p.JobStatus}
		case *agentv1.ServerMessage_GetVersions:
			req = &ServerRequest{ID: msg.Id, Payload: p.GetVersions}
		case *agentv1.ServerMessage_PbmSwitchPitr:
			req = &ServerRequest{ID: msg.Id, Payload: p.PbmSwitchPitr}
		case *agentv1.ServerMessage_AgentLogs:
			req = &ServerRequest{ID: msg.Id, Payload: p.AgentLogs}
		case *agentv1.ServerMessage_ServiceInfo:
			req = &ServerRequest{ID: msg.Id, Payload: p.ServiceInfo}

		// responses
		case *agentv1.ServerMessage_Pong:
			c.publish(msg.Id, msg.Status, p.Pong)
		case *agentv1.ServerMessage_StateChanged:
			c.publish(msg.Id, msg.Status, p.StateChanged)
		case *agentv1.ServerMessage_QanCollect:
			c.publish(msg.Id, msg.Status, p.QanCollect)
		case *agentv1.ServerMessage_ActionResult:
			c.publish(msg.Id, msg.Status, p.ActionResult)

		default:
			c.cancel(msg.Id, fmt.Errorf("unimplemented: failed to handle received message %s", msg))
			if msg.Status != nil && grpcstatus.FromProto(msg.Status).Code() == codes.Unimplemented {
				// This means pmm-managed does not know the message payload type we just sent.
				// We continue here to stop endless cycle of Unimplemented messages between pmm-agent and pmm-managed.
				c.l.Warnf("pmm-managed was not able to process message with id: %d, handling of that payload type is unimplemented", msg.Id)
				continue
			}
			c.Send(&AgentResponse{
				ID:     msg.Id,
				Status: grpcstatus.New(codes.Unimplemented, "can't handle message type sent, it is not implemented"),
			})
		}

		if req == nil {
			continue
		}

		// Never park here without a way out: this goroutine also delivers every response the
		// consumer of Requests() may be waiting for, so a full queue stops both directions at
		// once. Giving up on the connection is what breaks that cycle - the caller of Run
		// redials, and the fresh connection starts with an empty queue. See PMM-15431.
		select {
		case c.requests <- req:
		case <-c.closeWait:
			return
		case <-time.After(requestQueueStuckTimeout):
			// Loudly: this is a monitoring outage that recovers itself, and the whole
			// point of PMM-15431 is that it used to be invisible. Closing on its own is
			// only reported at debug level.
			c.l.Errorf("Request queue full for %s, giving up on the connection.", requestQueueStuckTimeout)
			c.close(fmt.Errorf("request queue full for %s", requestQueueStuckTimeout))
			return
		}
	}
}

func (c *Channel) removeResponseChannel(id uint32) chan Response {
	c.m.Lock()
	defer c.m.Unlock()
	if c.responses == nil { // Channel is closed
		return nil
	}

	ch := c.responses[id]
	if ch == nil {
		return nil
	}
	delete(c.responses, id)
	return ch
}

// cancel sends an error to the subscriber and closes the subscription channel.
func (c *Channel) cancel(id uint32, err error) {
	if ch := c.removeResponseChannel(id); ch != nil {
		ch <- Response{Error: err}
		close(ch)
	}
}

func (c *Channel) subscribe(id uint32) chan Response {
	ch := make(chan Response, 1)

	c.m.Lock()
	if c.responses == nil { // Channel is closed, no more subscriptions
		c.m.Unlock()
		close(ch)
		return ch
	}

	if _, ok := c.responses[id]; ok {
		// it is possible only on lastSentRequestID wrap around, and we can't recover from that
		c.l.Panicf("Already have subscriber for ID %d.", id)
	}

	c.responses[id] = ch
	c.m.Unlock()
	return ch
}

func (c *Channel) publish(id uint32, status *protostatus.Status, resp agentv1.ServerResponsePayload) {
	if status != nil && grpcstatus.FromProto(status).Code() != codes.OK {
		c.l.Errorf("got response %v with status %v", resp, status)
		c.cancel(id, grpcstatus.FromProto(status).Err())
		return
	}

	if ch := c.removeResponseChannel(id); ch != nil {
		ch <- Response{Payload: resp}
	}
}

// Describe implements prometheus.Collector.
func (c *Channel) Describe(ch chan<- *prometheus.Desc) {
	c.mRecv.Describe(ch)
	c.mSend.Describe(ch)
}

// Collect implement prometheus.Collector.
func (c *Channel) Collect(ch chan<- prometheus.Metric) {
	c.mRecv.Collect(ch)
	c.mSend.Collect(ch)
}

// check interfaces.
var (
	_ prometheus.Collector = (*Channel)(nil)
)
