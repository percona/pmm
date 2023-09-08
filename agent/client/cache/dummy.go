// Copyright 2023 Percona LLC
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

package cache

import (
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/percona/pmm/agent/models"
	agenterrors "github.com/percona/pmm/agent/utils/errors"
	"github.com/percona/pmm/api/agentpb"
)

// Dummy represent dummy cache.
type Dummy struct {
	s atomic.Pointer[models.Sender]
}

// Close to satisfy interface.
func (*Dummy) Close() {}

// Send drops agent responses on nil channel.
func (c *Dummy) Send(resp *models.AgentResponse) error {
	s := c.s.Load()
	if s == nil {
		return nil
	}
	err := (*s).Send(resp)
	if err != nil && errors.As(err, &agenterrors.ErrChanConn) {
		c.s.CompareAndSwap(s, nil)
	}
	return err
}

// SendAndWaitResponse drops AgentMessages on nil channel.
func (c *Dummy) SendAndWaitResponse(payload agentpb.AgentRequestPayload) (agentpb.ServerResponsePayload, error) { //nolint:ireturn
	s := c.s.Load()
	if s == nil {
		return &agentpb.StateChangedResponse{}, nil
	}
	resp, err := (*s).SendAndWaitResponse(payload)
	if err != nil && errors.As(err, &agenterrors.ErrChanConn) {
		c.s.CompareAndSwap(s, nil)
	}
	return resp, err
}

// SetSender sets sender.
func (c *Dummy) SetSender(s models.Sender) {
	c.s.Store(&s)
}
