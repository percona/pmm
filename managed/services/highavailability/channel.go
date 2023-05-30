package highavailability

import (
	"github.com/percona/pmm/api/agentpb"
)

type Channel struct{}

func NewChannel() *Channel {
	return &Channel{}
}

func (c *Channel) Send(*agentpb.ServerMessage) error {
	return nil
}

func (c *Channel) Recv() (*agentpb.AgentMessage, error) {
	return nil, nil
}
