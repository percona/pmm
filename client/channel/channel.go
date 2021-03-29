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

// Package channel contains protocol details of two-way communication channel between pmm-managed and pmm-agent.
package channel

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	serverRequestsCap = 32

	prometheusNamespace = "pmm_agent"
	prometheusSubsystem = "channel"
)

// ServerRequest represents an request from server.
// It is similar to agentpb.ServerMessage except it can contain only requests,
// and the payload is already unwrapped (XXX instead of ServerMessage_XXX).
type ServerRequest struct {
	ID      uint32
	Payload agentpb.ServerRequestPayload
}

// AgentResponse represents agent's response.
// It is similar to agentpb.AgentMessage except it can contain only responses,
// and the payload is already unwrapped (XXX instead of AgentMessage_XXX).
type AgentResponse struct {
	ID      uint32
	Payload agentpb.AgentResponsePayload
}

// Channel encapsulates two-way communication channel between pmm-managed and pmm-agent.
//
// All exported methods are thread-safe.
type Channel struct { //nolint:maligned
	s agentpb.Agent_ConnectClient
	l *logrus.Entry

	mRecv, mSend prometheus.Counter

	lastSentRequestID uint32

	sendM sync.Mutex

	m         sync.Mutex
	responses map[uint32]chan agentpb.ServerResponsePayload
	requests  chan *ServerRequest

	closeOnce sync.Once
	closeWait chan struct{}
	closeErr  error
}

// New creates new two-way communication channel with given stream.
//
// Stream should not be used by the caller after channel is created.
func New(stream agentpb.Agent_ConnectClient) *Channel {
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

		responses: make(map[uint32]chan agentpb.ServerResponsePayload),
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

// SendResponse sends message to pmm-managed. It is no-op once channel is closed (see Wait).
func (c *Channel) Send(resp *AgentResponse) {
	msg := &agentpb.AgentMessage{
		Id:      resp.ID,
		Payload: resp.Payload.AgentMessageResponsePayload(),
	}
	c.send(msg)
}

// SendAndWaitResponse sends request to pmm-managed, blocks until response is available, and returns it.
// Response will be nil if channel is closed.
// It is no-op once channel is closed (see Wait).
func (c *Channel) SendAndWaitResponse(payload agentpb.AgentRequestPayload) agentpb.ServerResponsePayload {
	id := atomic.AddUint32(&c.lastSentRequestID, 1)
	ch := c.subscribe(id)

	c.send(&agentpb.AgentMessage{
		Id:      id,
		Payload: payload.AgentMessageRequestPayload(),
	})

	return <-ch
}

func (c *Channel) send(msg *agentpb.AgentMessage) {
	c.sendM.Lock()
	select {
	case <-c.closeWait:
		c.sendM.Unlock()
		return
	default:
	}

	// do not use default compact representation for large/complex messages
	if size := proto.Size(msg); size < 100 {
		c.l.Debugf("Sending message (%d bytes): %s.", size, msg)
	} else {
		c.l.Debugf("Sending message (%d bytes):\n%s\n", size, proto.MarshalTextString(msg))
	}

	err := c.s.Send(msg)
	c.sendM.Unlock()
	if err != nil {
		c.close(errors.Wrap(err, "failed to send message"))
		return
	}
	c.mSend.Inc()
}

// runReader receives messages from server
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

		// do not use default compact representation for large/complex messages
		if size := proto.Size(msg); size < 100 {
			c.l.Debugf("Received message (%d bytes): %s.", size, msg)
		} else {
			c.l.Debugf("Received message (%d bytes):\n%s\n", size, proto.MarshalTextString(msg))
		}

		switch p := msg.Payload.(type) {
		// requests
		case *agentpb.ServerMessage_Ping:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.Ping,
			}
		case *agentpb.ServerMessage_SetState:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.SetState,
			}
		case *agentpb.ServerMessage_StartAction:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.StartAction,
			}
		case *agentpb.ServerMessage_StopAction:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.StopAction,
			}
		case *agentpb.ServerMessage_CheckConnection:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.CheckConnection,
			}
		case *agentpb.ServerMessage_StartJob:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.StartJob,
			}
		case *agentpb.ServerMessage_StopJob:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.StopJob,
			}
		case *agentpb.ServerMessage_JobStatus:
			c.requests <- &ServerRequest{
				ID:      msg.Id,
				Payload: p.JobStatus,
			}

		// responses
		case *agentpb.ServerMessage_Pong:
			c.publish(msg.Id, p.Pong)
		case *agentpb.ServerMessage_StateChanged:
			c.publish(msg.Id, p.StateChanged)
		case *agentpb.ServerMessage_QanCollect:
			c.publish(msg.Id, p.QanCollect)
		case *agentpb.ServerMessage_ActionResult:
			c.publish(msg.Id, p.ActionResult)

		case nil:
			c.close(errors.Errorf("failed to handle received message %s", msg))
			return
		}
	}
}

func (c *Channel) subscribe(id uint32) chan agentpb.ServerResponsePayload {
	ch := make(chan agentpb.ServerResponsePayload, 1)

	c.m.Lock()
	if c.responses == nil { // Channel is closed, no more subscriptions
		c.m.Unlock()
		close(ch)
		return ch
	}

	_, ok := c.responses[id]
	if ok {
		// it is possible only on lastSentRequestID wrap around, and we can't recover from that
		c.l.Panicf("Already have subscriber for ID %d.", id)
	}

	c.responses[id] = ch
	c.m.Unlock()
	return ch
}

func (c *Channel) publish(id uint32, resp agentpb.ServerResponsePayload) {
	c.m.Lock()
	if c.responses == nil { // Channel is closed, no more publishing
		c.m.Unlock()
		return
	}

	ch := c.responses[id]
	if ch == nil {
		c.m.Unlock()
		c.close(errors.WithStack(fmt.Errorf("no subscriber for ID %d", id)))
		return
	}

	delete(c.responses, id)
	c.m.Unlock()
	ch <- resp
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

// check interfaces
var (
	_ prometheus.Collector = (*Channel)(nil)
)
