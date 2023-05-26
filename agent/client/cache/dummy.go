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
func (c *Dummy) SendAndWaitResponse(payload agentpb.AgentRequestPayload) (agentpb.ServerResponsePayload, error) {
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
