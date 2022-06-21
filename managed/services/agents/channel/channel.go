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
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	protostatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/managed/utils/logger"
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
	Status  *grpcstatus.Status
	Payload agentpb.ServerResponsePayload
}

// Metrics represents useful channel metrics.
type Metrics struct {
	Sent      float64 // a total number of messages sent to pmm-agent
	Recv      float64 // a total number of messages received from pmm-agent
	Responses float64 // the current length of the response queue
	Requests  float64 // the current length of the request queue
}

// Response is a type used to pass response from pmm-agent to the subscriber.
type Response struct {
	Payload agentpb.AgentResponsePayload
	Error   error
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
	responses map[uint32]chan Response
	requests  chan *AgentRequest

	closeOnce sync.Once
	closeWait chan struct{}
	closeErr  error

	l *logrus.Entry
}

// New creates new two-way communication channel with given stream.
//
// Stream should not be used by the caller after channel is created.
func New(stream agentpb.Agent_ConnectServer) *Channel {
	s := &Channel{
		s: stream,

		responses: make(map[uint32]chan Response),
		requests:  make(chan *AgentRequest, agentRequestsCap),

		closeWait: make(chan struct{}),

		l: logger.Get(stream.Context()),
	}

	go s.runReceiver()
	return s
}

// close marks channel as closed with given error - only once.
func (c *Channel) close(err error) {
	c.closeOnce.Do(func() {
		c.l.Debugf("Closing with error: %+v", err)
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
		Id:     resp.ID,
		Status: resp.Status.Proto(),
	}
	if resp.Payload != nil {
		msg.Payload = resp.Payload.ServerMessageResponsePayload()
	}
	c.send(msg)
}

// SendAndWaitResponse sends request to pmm-managed, blocks until response is available.
// If error occurred - subscription got canceled - returned payload is nil and error contains reason for cancellation.
// Response and error will be both nil if channel is closed.
// It is no-op once channel is closed (see Wait).
func (c *Channel) SendAndWaitResponse(payload agentpb.ServerRequestPayload) (agentpb.AgentResponsePayload, error) {
	id := atomic.AddUint32(&c.lastSentRequestID, 1)
	ch := c.subscribe(id)

	c.send(&agentpb.ServerMessage{
		Id:      id,
		Payload: payload.ServerMessageRequestPayload(),
	})
	resp, ok := <-ch
	if !ok {
		return nil, errors.New("channel is closed")
	}

	return resp.Payload, resp.Error
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
	atomic.AddUint32(&c.mSent, 1)
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
		atomic.AddUint32(&c.mRecv, 1)

		// do not use default compact representation for large/complex messages
		if size := proto.Size(msg); size < 100 {
			c.l.Debugf("Received message (%d bytes): %s.", size, msg)
		} else {
			c.l.Debugf("Received message (%d bytes):\n%s\n", size, proto.MarshalTextString(msg))
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
			c.publish(msg.Id, msg.Status, p.Pong)
		case *agentpb.AgentMessage_SetState:
			c.publish(msg.Id, msg.Status, p.SetState)
		case *agentpb.AgentMessage_StartAction:
			c.publish(msg.Id, msg.Status, p.StartAction)
		case *agentpb.AgentMessage_StopAction:
			c.publish(msg.Id, msg.Status, p.StopAction)
		case *agentpb.AgentMessage_StartJob:
			c.publish(msg.Id, msg.Status, p.StartJob)
		case *agentpb.AgentMessage_StopJob:
			c.publish(msg.Id, msg.Status, p.StopJob)
		case *agentpb.AgentMessage_JobStatus:
			c.publish(msg.Id, msg.Status, p.JobStatus)
		case *agentpb.AgentMessage_CheckConnection:
			c.publish(msg.Id, msg.Status, p.CheckConnection)
		case *agentpb.AgentMessage_GetVersions:
			c.publish(msg.Id, msg.Status, p.GetVersions)
		case *agentpb.AgentMessage_PbmSwitchPitr:
			c.publish(msg.Id, msg.Status, p.PbmSwitchPitr)
		case *agentpb.AgentMessage_ParseDefaultsFile:
			c.publish(msg.Id, msg.Status, p.ParseDefaultsFile)

		case nil:
			c.cancel(msg.Id, errors.Errorf("unimplemented: failed to handle received message %s", msg))
			if msg.Status != nil && grpcstatus.FromProto(msg.Status).Code() == codes.Unimplemented {
				// This means pmm-agent does not know the message payload type we just sent.
				// We continue here to stop endless cycle of Unimplemented messages between pmm-managed and pmm-agent.
				c.l.Warnf("pmm-agent was not able to process message with id: %d, handling of that payload type is unimplemented", msg.Id)
				continue
			}
			c.Send(&ServerResponse{
				ID:     msg.Id,
				Status: grpcstatus.New(codes.Unimplemented, "can't handle message type send, it is not implemented"),
			})
		}
	}
}

func (c *Channel) subscribe(id uint32) chan Response {
	ch := make(chan Response, 1)
	c.rw.Lock()
	if c.responses == nil { // Channel is closed, no more subscriptions
		c.rw.Unlock()
		close(ch)
		return ch
	}

	_, ok := c.responses[id]
	if ok {
		// it is possible only on lastSentRequestID wrap around, and we can't recover from that
		c.l.Panicf("Already have subscriber for ID %d.", id)
	}

	c.responses[id] = ch
	c.rw.Unlock()
	return ch
}

func (c *Channel) removeResponseChannel(id uint32) chan Response {
	c.rw.Lock()
	defer c.rw.Unlock()
	if c.responses == nil { // Channel is closed, no more publishing
		return nil
	}

	ch := c.responses[id]
	if ch == nil {
		c.l.Errorf("No subscriber for ID %d", id)
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

func (c *Channel) publish(id uint32, status *protostatus.Status, resp agentpb.AgentResponsePayload) {
	if status != nil && grpcstatus.FromProto(status).Code() != codes.OK {
		c.l.Errorf("got response %v with status %v", resp, status)
		c.cancel(id, grpcstatus.FromProto(status).Err())
		return
	}

	if ch := c.removeResponseChannel(id); ch != nil {
		ch <- Response{Payload: resp}
	}
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
