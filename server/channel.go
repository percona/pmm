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

// Package server contains business logic of working with pmm-managed.
package server

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/percona/pmm/api/agent"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	serverRequestsCap = 32

	prometheusNamespace = "pmm_agent"
	prometheusSubsystem = "channel"
)

// Channel encapsulates two-way communication channel between pmm-managed and pmm-agent.
//
// All exported methods are thread-safe.
type Channel struct { //nolint:maligned
	s agent.Agent_ConnectClient
	l *logrus.Entry

	mRecv, mSend prometheus.Counter

	lastSentRequestID uint32

	sendM sync.Mutex

	m         sync.Mutex
	responses map[uint32]chan agent.ServerMessagePayload
	requests  chan *agent.ServerMessage

	closeOnce sync.Once
	closeWait chan struct{}
	closeErr  error
}

// NewChannel creates new two-way communication channel with given stream.
//
// Stream should not be used by the caller after channel is created.
func NewChannel(stream agent.Agent_ConnectClient) *Channel {
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

		responses: make(map[uint32]chan agent.ServerMessagePayload),
		requests:  make(chan *agent.ServerMessage, serverRequestsCap),

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
func (c *Channel) Requests() <-chan *agent.ServerMessage {
	return c.requests
}

// SendResponse sends message to pmm-managed. It is no-op once channel is closed (see Wait).
func (c *Channel) SendResponse(msg *agent.AgentMessage) {
	c.send(msg)
}

// SendRequest sends request to pmm-managed, blocks until response is available, and returns it.
// Response will nil if channel is closed.
// It is no-op once channel is closed (see Wait).
func (c *Channel) SendRequest(payload agent.AgentMessagePayload) agent.ServerMessagePayload {
	id := atomic.AddUint32(&c.lastSentRequestID, 1)
	ch := c.subscribe(id)

	c.send(&agent.AgentMessage{
		Id:      id,
		Payload: payload,
	})

	return <-ch
}

func (c *Channel) send(msg *agent.AgentMessage) {
	c.sendM.Lock()
	select {
	case <-c.closeWait:
		c.sendM.Unlock()
		return
	default:
	}

	c.l.Debugf("Sending message: %s.", msg)
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
		c.l.Debugf("Received message: %s.", msg)
		c.mRecv.Inc()

		switch msg.Payload.(type) {
		// requests
		case *agent.ServerMessage_Ping, *agent.ServerMessage_State:
			c.requests <- msg

		// responses
		case *agent.ServerMessage_QanData:
			c.publish(msg.Id, msg.Payload)

		default:
			c.close(errors.Wrapf(err, "failed to handle received message %s", msg))
			return
		}
	}
}

func (c *Channel) subscribe(id uint32) chan agent.ServerMessagePayload {
	ch := make(chan agent.ServerMessagePayload, 1)

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

func (c *Channel) publish(id uint32, payload agent.ServerMessagePayload) {
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
	ch <- payload
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
