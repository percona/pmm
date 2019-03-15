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

package qan

import (
	"context"

	"github.com/percona/pmm/api/qanpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/percona/pmm-managed/models"
)

type Client struct {
	c qanpb.CollectorClient
	l *logrus.Entry
}

func NewClient(cc *grpc.ClientConn) *Client {
	return &Client{
		c: qanpb.NewCollectorClient(cc),
		l: logrus.WithField("component", "qan"),
	}
}

func (c *Client) Collect(ctx context.Context, req *qanpb.CollectRequest, agent *models.Agent) error {
	labels, err := agent.GetCustomLabels()
	if err != nil {
		c.l.Error(err)
	}

	for _, m := range req.MetricsBucket {
		if m.Labels == nil {
			m.Labels = make(map[string]string)
		}
		for k, v := range labels {
			m.Labels[k] = v
		}
	}

	c.l.Debugf("%+v", req)
	res, err := c.c.Collect(ctx, req)
	if err != nil {
		return errors.Wrap(err, "failed to sent CollectRequest to QAN")
	}
	c.l.Debugf("%+v", res)
	return nil
}
