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

// Package channel contains protocol details of two-way communication channel between pmm-managed and pmm-agent.
package channel

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"

	"github.com/percona/pmm-managed/utils/logger"
)

const (
	agentRequestsCap = 32
)

// AgentRequest represents an request from agent.
// It is similar to agentpb.AgentMessage except it can contain only requests,
// and the payload is already unwrapped (XXX instead of AgentMessage_XXX).
type AgentRequest struct {
	ID      uint32
	Payload agentpb.AgentRequestPayload
}

// ServerResponse represents server's response.
// It is similar to agentpb.ServerMessage except it can contain only responses,
// and the payload is already unwrapped (XXX instead of ServerMessage_XXX).
type ServerResponse struct {
	ID      uint32
	Payload agentpb.ServerResponsePayload
}

// Metrics represents useful channel metrics.
type Metrics struct {
	Sent      float64 // a total number of messages sent to pmm-agent
	Recv      float64 // a total number of messages received from pmm-agent
	Responses float64 // the current length of the response queue
	Requests  float64 // the current length of the request queue
}

// Channel encapsulates two-way communication channel between pmm-managed and pmm-agent.
//
// All exported methods are thread-safe.
//nolint:maligned
type Channel struct {
	s agentpb.Agent_ConnectServer

	mSent, mRecv uint32

	lastSentRequestID uint32

	sendM sync.Mutex

	rw        sync.RWMutex
	responses map[uint32]chan agentpb.AgentResponsePayload
	requests  chan *AgentRequest

	closeOnce sync.Once
	closeWait chan struct{}
	closeErr  error
}

// New creates new two-way communication channel with given stream.
//
// Stream should not be used by the caller after channel is created.
func New(stream agentpb.Agent_ConnectServer) *Channel {
	s := &Channel{
		s: stream,

		responses: make(map[uint32]chan agentpb.AgentResponsePayload),
		requests:  make(chan *AgentRequest, agentRequestsCap),

		closeWait: make(chan struct{}),
	}

	go s.runReceiver()
	return s
}

// close marks channel as closed with given error - only once.
func (c *Channel) close(err error) {
	c.closeOnce.Do(func() {
		logger.Get(c.s.Context()).Debugf("Closing with error: %+v", err)
		c.closeErr = err

		c.rw.Lock()
		for _, ch := range c.responses { // unblock all subscribers
			close(ch)
		}
		c.responses = nil // prevent future subscriptions
		c.rw.Unlock()

		close(c.closeWait)
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
func (c *Channel) Requests() <-chan *AgentRequest {
	return c.requests
}

// Send sends message to pmm-managed. It is no-op once channel is closed (see Wait).
func (c *Channel) Send(resp *ServerResponse) {
	msg := &agentpb.ServerMessage{
		Id:      resp.ID,
		Payload: resp.Payload.ServerMessageResponsePayload(),
	}
	c.send(msg)
}

// SendAndWaitResponse sends request to pmm-managed, blocks until response is available, and returns it.
// Response will be nil if channel is closed.
// It is no-op once channel is closed (see Wait).
func (c *Channel) SendAndWaitResponse(payload agentpb.ServerRequestPayload) agentpb.AgentResponsePayload {
	id := atomic.AddUint32(&c.lastSentRequestID, 1)
	ch := c.subscribe(id)

	c.send(&agentpb.ServerMessage{
		Id:      id,
		Payload: payload.ServerMessageRequestPayload(),
	})

	return <-ch
}

func (c *Channel) send(msg *agentpb.ServerMessage) {
	c.sendM.Lock()
	select {
	case <-c.closeWait:
		c.sendM.Unlock()
		return
	default:
	}

	// do not use default compact representation for large/complex messages
	l := logger.Get(c.s.Context())
	if size := proto.Size(msg); size < 100 {
		l.Debugf("Sending message (%d bytes): %s.", size, msg)
	} else {
		l.Debugf("Sending message (%d bytes):\n%s\n", size, proto.MarshalTextString(msg))
	}

	err := c.s.Send(msg)
	c.sendM.Unlock()
	if err != nil {
		c.close(errors.Wrap(err, "failed to send message"))
		return
	}
	atomic.AddUint32(&c.mSent, 1)
}

// runReader receives messages from server.
func (c *Channel) runReceiver() {
	l := logger.Get(c.s.Context())
	defer func() {
		close(c.requests)
		l.Debug("Exiting receiver goroutine.")
	}()

	for {
		msg, err := c.s.Recv()
		if err != nil {
			c.close(errors.Wrap(err, "failed to receive message"))
			return
		}
		atomic.AddUint32(&c.mRecv, 1)

		// do not use default compact representation for large/complex messages
		if size := proto.Size(msg); size < 100 {
			l.Debugf("Received message (%d bytes): %s.", size, msg)
		} else {
			l.Debugf("Received message (%d bytes):\n%s\n", size, proto.MarshalTextString(msg))
		}

		switch p := msg.Payload.(type) {
		// requests
		case *agentpb.AgentMessage_Ping:
			c.requests <- &AgentRequest{
				ID:      msg.Id,
				Payload: p.Ping,
			}
		case *agentpb.AgentMessage_StateChanged:
			c.requests <- &AgentRequest{
				ID:      msg.Id,
				Payload: p.StateChanged,
			}
		case *agentpb.AgentMessage_QanCollect:
			c.requests <- &AgentRequest{
				ID:      msg.Id,
				Payload: p.QanCollect,
			}
		case *agentpb.AgentMessage_ActionResult:
			c.requests <- &AgentRequest{
				ID:      msg.Id,
				Payload: p.ActionResult,
			}

		// simple messages
		case *agentpb.AgentMessage_JobResult:
			c.requests <- &AgentRequest{
				ID:      msg.Id,
				Payload: p.JobResult,
			}
		case *agentpb.AgentMessage_JobProgress:
			c.requests <- &AgentRequest{
				ID:      msg.Id,
				Payload: p.JobProgress,
			}

		// responses
		case *agentpb.AgentMessage_Pong:
			c.publish(msg.Id, p.Pong)
		case *agentpb.AgentMessage_SetState:
			c.publish(msg.Id, p.SetState)
		case *agentpb.AgentMessage_StartAction:
			c.publish(msg.Id, p.StartAction)
		case *agentpb.AgentMessage_StopAction:
			c.publish(msg.Id, p.StopAction)
		case *agentpb.AgentMessage_StartJob:
			c.publish(msg.Id, p.StartJob)
		case *agentpb.AgentMessage_StopJob:
			c.publish(msg.Id, p.StopJob)
		case *agentpb.AgentMessage_JobStatus:
			c.publish(msg.Id, p.JobStatus)
		case *agentpb.AgentMessage_CheckConnection:
			c.publish(msg.Id, p.CheckConnection)

		case nil:
			c.close(errors.Errorf("failed to handle received message %s", msg))
			return
		}
	}
}

func (c *Channel) subscribe(id uint32) chan agentpb.AgentResponsePayload {
	ch := make(chan agentpb.AgentResponsePayload, 1)

	c.rw.Lock()
	if c.responses == nil { // Channel is closed, no more subscriptions
		c.rw.Unlock()
		close(ch)
		return ch
	}

	_, ok := c.responses[id]
	if ok {
		// it is possible only on lastSentRequestID wrap around, and we can't recover from that
		logger.Get(c.s.Context()).Panicf("Already have subscriber for ID %d.", id)
	}

	c.responses[id] = ch
	c.rw.Unlock()
	return ch
}

func (c *Channel) publish(id uint32, resp agentpb.AgentResponsePayload) {
	c.rw.Lock()
	if c.responses == nil { // Channel is closed, no more publishing
		c.rw.Unlock()
		return
	}

	ch := c.responses[id]
	if ch == nil {
		c.rw.Unlock()
		c.close(errors.WithStack(fmt.Errorf("no subscriber for ID %d", id)))
		return
	}

	delete(c.responses, id)
	c.rw.Unlock()
	ch <- resp
}

// Metrics returns current channel metrics.
func (c *Channel) Metrics() *Metrics {
	c.rw.RLock()
	responses := len(c.responses)
	requests := len(c.requests)
	c.rw.RUnlock()

	return &Metrics{
		Sent:      float64(atomic.LoadUint32(&c.mSent)),
		Recv:      float64(atomic.LoadUint32(&c.mRecv)),
		Responses: float64(responses),
		Requests:  float64(requests),
	}
}
