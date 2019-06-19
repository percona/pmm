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
	"time"

	"github.com/percona/pmm/api/qanpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// Client represents qan-api client for data collection.
type Client struct {
	c  qanpb.CollectorClient
	db *reform.DB
	l  *logrus.Entry
}

// NewClient returns new client for given gRPC connection.
func NewClient(cc *grpc.ClientConn, db *reform.DB) *Client {
	return &Client{
		c:  qanpb.NewCollectorClient(cc),
		db: db,
		l:  logrus.WithField("component", "qan"),
	}
}

func mergeLabels(node *models.Node, service *models.Service, agent *models.Agent) (map[string]string, error) {
	res := make(map[string]string, 16)

	labels, err := node.UnifiedLabels()
	if err != nil {
		return nil, err
	}
	for name, value := range labels {
		res[name] = value
	}

	labels, err = service.UnifiedLabels()
	if err != nil {
		return nil, err
	}
	for name, value := range labels {
		res[name] = value
	}

	labels, err = agent.UnifiedLabels()
	if err != nil {
		return nil, err
	}
	for name, value := range labels {
		res[name] = value
	}

	return res, nil
}

// Collect adds labels to the data from pmm-agent and sends it to qan-api.
func (c *Client) Collect(ctx context.Context, req *qanpb.CollectRequest) error {
	// TODO That code is simple, but performance will be very bad for any non-trivial load.
	// https://jira.percona.com/browse/PMM-3894

	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > time.Second {
			c.l.Warnf("Collect for %d buckets took %s.", len(req.MetricsBucket), dur)
		}
	}()

	for i, m := range req.MetricsBucket {
		if m.AgentId == "" {
			c.l.Errorf("Empty agent_id for bucket with query_id %q, can't add labels.", m.Queryid)
			continue
		}

		// get agent
		agent, err := models.AgentFindByID(c.db.Querier, m.AgentId)
		if err != nil {
			c.l.Error(err)
			continue
		}

		// get service
		services, err := models.ServicesForAgent(c.db.Querier, m.AgentId)
		if err != nil {
			c.l.Error(err)
			continue
		}
		if len(services) != 1 {
			c.l.Errorf("Expected 1 Service, got %d.", len(services))
			continue
		}
		service := services[0]

		// get node for that service (not for that agent)
		node, err := models.FindNodeByID(c.db.Querier, service.NodeID)
		if err != nil {
			c.l.Error(err)
			continue
		}

		labels, err := mergeLabels(node, service, agent)
		if err != nil {
			c.l.Error(err)
			continue
		}

		// in order of fields in MetricsBucket
		for labelName, field := range map[string]*string{
			"service_name":    &m.ServiceName,
			"replication_set": &m.ReplicationSet,
			"cluster":         &m.Cluster,
			"service_type":    &m.ServiceType,
			"environment":     &m.Environment,
			"az":              &m.Az,
			"region":          &m.Region,
			"node_model":      &m.NodeModel,
			"container_name":  &m.ContainerName,
			"agent_id":        &m.AgentId,
		} {
			value := labels[labelName]
			delete(labels, labelName)
			if *field != "" {
				if *field == value {
					c.l.Debugf("%q is not empty, but has the same value %q.", labelName, *field)
				} else {
					c.l.Warnf("%q is not empty: overwriting %q with %q.", labelName, *field, value)
				}
			}
			*field = value
		}

		if m.Labels != nil {
			c.l.Errorf("Labels were not empty: %+v.", m.Labels)
		}
		m.Labels = labels

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
