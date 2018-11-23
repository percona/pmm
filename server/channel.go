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
	"sync"
	"sync/atomic"

	"github.com/percona/pmm/api/agent"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	serverRequestsCap = 32
)

// Channel contains business logic of communication with pmm-managed.
type Channel struct {
	s agent.Agent_ConnectClient
	l *logrus.Entry

	// server requests
	requests chan *agent.ServerMessage

	lastSentRequestID uint32

	//
	m         sync.Mutex
	responses map[uint32]chan agent.ServerMessagePayload

	//
	closeOnce sync.Once
	closeDone chan struct{}
	closeErr  error
}

// NewChannel starts goroutine to dispatch messages from server and returns new Conn object
func NewChannel(stream agent.Agent_ConnectClient) *Channel {
	s := &Channel{
		s:         stream,
		l:         logrus.WithField("component", "channel"), // only for debug logging
		requests:  make(chan *agent.ServerMessage, serverRequestsCap),
		responses: make(map[uint32]chan agent.ServerMessagePayload),
		closeDone: make(chan struct{}),
	}

	go s.runReceiver()
	return s
}

func (c *Channel) close(err error) {
	c.closeOnce.Do(func() {
		c.l.Debugf("Closing with error: %+v", err)
		c.closeErr = err

		c.m.Lock()
		for _, ch := range c.responses {
			close(ch)
		}
		c.responses = nil
		c.m.Unlock()

		close(c.requests)
		close(c.closeDone)
	})
}

func (c *Channel) Wait() error {
	<-c.closeDone
	return c.closeErr
}

func (c *Channel) Requests() <-chan *agent.ServerMessage {
	return c.requests
}

func (c *Channel) SendResponse(msg *agent.AgentMessage) {
	c.send(msg)
}

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
	c.l.Debugf("Sending message: %c.", msg)

	c.m.Lock()
	err := c.s.Send(msg)
	c.m.Unlock()
	if err != nil {
		c.close(errors.Wrap(err, "failed to send message"))
	}
}

// runReader receives messages from server
func (c *Channel) runReceiver() {
	defer c.l.Debug("Exiting receiver goroutine.")

	for {
		msg, err := c.s.Recv()
		if err != nil {
			c.close(errors.Wrap(err, "failed to receive message"))
			return
		}
		c.l.Debugf("Received message: %s.", msg)

		switch msg.Payload.(type) {
		// requests
		case *agent.ServerMessage_Ping, *agent.ServerMessage_State:
			c.requests <- msg

		// responses
		case *agent.ServerMessage_Auth, *agent.ServerMessage_QanData:
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
	if c.responses == nil {
		c.m.Unlock()
		close(ch)
		return ch
	}

	_, ok := c.responses[id]
	if ok {
		c.m.Unlock()
		c.close(errors.Errorf("already have subscriber for ID %d", id))
		close(ch)
		return ch
	}

	c.responses[id] = ch
	c.m.Unlock()
	return ch
}

func (c *Channel) publish(id uint32, payload agent.ServerMessagePayload) {
	c.m.Lock()
	if c.responses == nil {
		c.m.Unlock()
		return
	}

	ch := c.responses[id]
	if ch == nil {
		c.m.Unlock()
		c.close(errors.Errorf("no subscriber for ID %d", id))
		return
	}

	delete(c.responses, id)
	c.m.Unlock()
	ch <- payload
}
