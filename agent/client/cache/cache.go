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

// Package cache incapsulates agent message storing logic.
package cache

import (
	"path"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/models"
	"github.com/percona/pmm/agent/utils/buffer-ring/bigqueue"
	agentpb "github.com/percona/pmm/api/agentpb/v1"
)

// Cache represent cache implementation based on bigqueue.
type Cache struct {
	l *logrus.Entry
	// prioritized represent cache for high priority agent messages e.g. job, action results
	prioritized *bigqueue.Ring
	// unprioritized represent cache for low priority agent messages e.g. qan metrics
	unprioritized *bigqueue.Ring
}

// New recreates cache.
func New(cfg config.Cache) (*Cache, error) {
	if cfg.Disable {
		return nil, errors.New("disable in cache config is set to true")
	}
	if cfg.Dir == "" {
		return nil, errors.New("cache directory is not set up")
	}
	l := logrus.WithField("component", "cache")
	prioritized, err := bigqueue.New(path.Join(cfg.Dir, "prioritized"), cfg.PrioritizedSize, l.WithField("type", "prioritized"))
	if err != nil {
		return nil, err
	}
	unprioritized, err := bigqueue.New(path.Join(cfg.Dir, "unprioritized"), cfg.UnprioritizedSize, l.WithField("type", "unprioritized"))
	if err != nil {
		return nil, err
	}
	return &Cache{
		l:             l,
		prioritized:   prioritized,
		unprioritized: unprioritized,
	}, nil
}

// Send stores agent response to cache on nil channel.
func (c *Cache) Send(resp *models.AgentResponse) error {
	var cache *bigqueue.Ring
	switch resp.Payload.(type) {
	case *agentpb.StartActionResponse,
		*agentpb.StopActionResponse,
		*agentpb.PBMSwitchPITRResponse,
		*agentpb.StartJobResponse,
		*agentpb.JobStatusResponse,
		*agentpb.GetVersionsResponse,
		*agentpb.JobProgress,
		*agentpb.StopJobResponse,
		*agentpb.CheckConnectionResponse,
		*agentpb.JobResult,
		*agentpb.ServiceInfoResponse:
		cache = c.prioritized
	default:
		cache = c.unprioritized
	}
	return cache.Send(resp)
}

// SendAndWaitResponse stores AgentMessages with AgentMessageRequestPayload on nil channel.
func (c *Cache) SendAndWaitResponse(payload agentpb.AgentRequestPayload) (agentpb.ServerResponsePayload, error) { //nolint:ireturn
	switch payload.(type) {
	case *agentpb.ActionResultRequest:
		return c.prioritized.SendAndWaitResponse(payload)
	case *agentpb.QANCollectRequest,
		*agentpb.StateChangedRequest:
		return c.unprioritized.SendAndWaitResponse(payload)
	default:
	}
	return &agentpb.StateChangedResponse{}, nil
}

// Close closes cache databases.
func (c *Cache) Close() {
	c.prioritized.Close()
	c.unprioritized.Close()
}

// SetSender sets sender and sends stored agent messages with sender.
func (c *Cache) SetSender(s models.Sender) {
	c.prioritized.SetSender(s)
	c.unprioritized.SetSender(s)
}
