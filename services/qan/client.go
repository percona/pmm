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

// Package qan contains business logic of working with QAN.
package qan

import (
	"context"

	"github.com/percona/pmm/api/qanpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/percona/pmm-managed/models"
)

// Client represents qan-api client for data collection.
type Client struct {
	c qanpb.CollectorClient
	l *logrus.Entry
}

// NewClient returns new client for given gRPC connection.
func NewClient(cc *grpc.ClientConn) *Client {
	return &Client{
		c: qanpb.NewCollectorClient(cc),
		l: logrus.WithField("component", "qan"),
	}
}

// Collect adds custom labels to the data from pmm-agent and sends it to qan-api.
func (c *Client) Collect(ctx context.Context, req *qanpb.CollectRequest, agent *models.Agent) error {
	labels, err := agent.GetCustomLabels()
	if err != nil {
		c.l.Error(err)
	}

	for i, m := range req.MetricsBucket {
		if m.Labels == nil {
			m.Labels = make(map[string]string)
		}
		for k, v := range labels {
			m.Labels[k] = v
		}
		if m.AgentUuid == "" {
			m.AgentUuid = agent.AgentID
		}

		req.MetricsBucket[i] = m
	}

	c.l.Debugf("%+v", req)
	res, err := c.c.Collect(ctx, req)
	if err != nil {
		return errors.Wrap(err, "failed to sent CollectRequest to QAN")
	}
	c.l.Debugf("%+v", res)
	return nil
}
