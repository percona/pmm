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
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	protostatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

const (
	serverRequestsCap = 32

	prometheusNamespace = "pmm_agent"
	prometheusSubsystem = "channel"
)

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
type Channel struct { //nolint:maligned
	s agentv1.AgentService_ConnectClient
	l *logrus.Entry

	mRecv, mSend prometheus.Counter

	lastSentRequestID uint32

	sendM sync.Mutex

	m         sync.Mutex
	responses map[uint32]chan Response
	requests  chan *ServerRequest

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

// SendAndWaitResponse sends request to pmm-managed, blocks until response is available.
// If error occurred - subscription got canceled - returned payload is nil and error contains reason for cancelation.
// Response and error will be both nil if channel is closed.
// It is no-op once channel is closed (see Wait).
func (c *Channel) SendAndWaitResponse(payload agentv1.AgentRequestPayload) (agentv1.ServerResponsePayload, error) { //nolint:ireturn,nolintlint
	id := atomic.AddUint32(&c.lastSentRequestID, 1)
	ch := c.subscribe(id)

	c.send(&agentv1.AgentMessage{
		Id:      id,
		Payload: payload.AgentMessageRequestPayload(),
	})

	resp := <-ch
	return resp.Payload, resp.Error
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
		if size := proto.Size(msg); size < 100 {
			c.l.Debugf("Sending message (%d bytes): %s.", size, msg)
		} else {
			c.l.Debugf("Sending message (%d bytes):\n%s\n", size, prototext.Format(msg))
		}
	}

	err := c.s.Send(msg)
	c.sendM.Unlock()
	if err != nil {
		c.close(errors.Wrap(err, "failed to send message"))
		return
	}
	c.mSend.Inc()
}

// runReader receives messages from server.
func (c *Channel) runReceiver() {
	defer func() {
		close(c.requests)
		c.l.Debug("Exiting receiver goroutine.")
	}()

	for {
		msg, err := c.s.Recv()
		if err != nil {
			c.close(errors.Wrap(err, "failed to receive message"))
			return
		}
		c.mRecv.Inc()

		// Check log level before calling formatting function.
		// Do not waste resources in case debug level is not enabled.
		if c.l.Logger.IsLevelEnabled(logrus.DebugLevel) {
			// do not use default compact representation for large/complex messages
			if size := proto.Size(msg); size < 100 {
				c.l.Debugf("Received message (%d bytes): %s.", size, msg)
			} else {
				c.l.Debugf("Received message (%d bytes):\n%s\n", size, prototext.Format(msg))
			}
		}

		switch p := msg.Payload.(type) {
		// requests
		case *agentv1.ServerMessage_Ping:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.Ping,
			}
		case *agentv1.ServerMessage_SetState:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.SetState,
			}
		case *agentv1.ServerMessage_StartAction:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.StartAction,
			}
		case *agentv1.ServerMessage_StopAction:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.StopAction,
			}
		case *agentv1.ServerMessage_CheckConnection:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.CheckConnection,
			}
		case *agentv1.ServerMessage_StartJob:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.StartJob,
			}
		case *agentv1.ServerMessage_StopJob:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.StopJob,
			}
		case *agentv1.ServerMessage_JobStatus:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.JobStatus,
			}
		case *agentv1.ServerMessage_GetVersions:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.GetVersions,
			}
		case *agentv1.ServerMessage_PbmSwitchPitr:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.PbmSwitchPitr,
			}
		case *agentv1.ServerMessage_AgentLogs:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.AgentLogs,
			}
		case *agentv1.ServerMessage_ServiceInfo:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.ServiceInfo,
			}

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
			c.cancel(msg.Id, errors.Errorf("unimplemented: failed to handle received message %s", msg))
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
