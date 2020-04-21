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

// Package alertmanager contains business logic of working with Alertmanager.
package alertmanager

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/percona/pmm/api/alertmanager/amclient"
	"github.com/percona/pmm/api/alertmanager/amclient/alert"
	"github.com/percona/pmm/api/alertmanager/amclient/general"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

const resendInterval = 30 * time.Second

// FIXME remove completely before release
const (
	addTestingAlerts   = false
	testingAlertsDelay = time.Second
)

// Service is responsible for interactions with Prometheus.
type Service struct {
	db             *reform.DB
	serverVersion  *version.Parsed
	agentsRegistry agentsRegistry
	r              *registry
	l              *logrus.Entry
}

// New creates new service.
func New(db *reform.DB, v string, agentsRegistry agentsRegistry) (*Service, error) {
	serverVersion, err := version.Parse(v)
	if err != nil {
		return nil, err
	}

	return &Service{
		db:             db,
		serverVersion:  serverVersion,
		agentsRegistry: agentsRegistry,
		r:              newRegistry(),
		l:              logrus.WithField("component", "alertmanager"),
	}, nil
}

func (svc *Service) AddAlert(id string, delayFor time.Duration, params *AlertParams) error {
	alert, err := makeAlert(params)
	if err != nil {
		return err
	}

	svc.r.Add(id, delayFor, alert)
	return nil
}

func (svc *Service) RemoveAlert(id string) {
	svc.r.Remove(id)
}

// Run runs Alertmanager configuration update loop until ctx is canceled.
func (svc *Service) Run(ctx context.Context) {
	svc.l.Info("Starting...")
	defer svc.l.Info("Done.")

	generateBaseConfig()

	t := time.NewTicker(resendInterval)
	defer t.Stop()

	for {
		if addTestingAlerts {
			svc.updateInventoryAlerts(ctx)
		}

		svc.sendAlerts(ctx)

		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// nothing, continue for loop
		}
	}
}

// generateBaseConfig generates /srv/alertmanager/alertmanager.base.yml if it is not present.
//
// TODO That's a temporary measure until we start generating /etc/alertmanager.yml
// using /srv/alertmanager/alertmanager.base.yml as a base. See supervisord config.
func generateBaseConfig() {
	const path = "/srv/alertmanager/alertmanager.base.yml"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		defaultBase := strings.TrimSpace(`
---
# You can edit this file; changes will be preserved.

route:
  receiver: empty
  routes: []

receivers:
  - name: empty
`) + "\n"
		_ = ioutil.WriteFile(path, []byte(defaultBase), 0644)
	}
}

// updateInventoryAlerts adds/updates alerts for inventory information in the registry.
func (svc *Service) updateInventoryAlerts(ctx context.Context) {
	var nodes []*models.Node
	var agents []*models.Agent
	err := svc.db.InTransaction(func(t *reform.TX) error {
		var e error
		nodes, e = models.FindNodes(t.Querier, models.NodeFilters{})
		if e != nil {
			return e
		}

		agents, e = models.FindAgents(t.Querier, models.AgentFilters{})
		return e
	})
	if err != nil {
		svc.l.Error(err)
		return
	}

	nodesMap := make(map[string]*models.Node, len(nodes))
	for _, n := range nodes {
		nodesMap[n.NodeID] = n
	}

	svc.r.RemovePrefix("inventory/")

	for _, agent := range agents {
		switch agent.AgentType {
		case models.PMMAgentType:
			svc.updateInventoryAlertsForPMMAgent(agent, nodesMap[pointer.GetString(agent.RunsOnNodeID)])
		}
	}
}

func (svc *Service) updateInventoryAlertsForPMMAgent(agent *models.Agent, node *models.Node) {
	if node == nil {
		svc.l.Errorf("Node with ID %v not found.", agent.RunsOnNodeID)
		return
	}

	prefix := "inventory/" + agent.AgentID + "/"

	if !svc.agentsRegistry.IsConnected(agent.AgentID) {
		name, alert, err := makeAlertPMMAgentNotConnected(agent, node)
		if err == nil {
			svc.r.Add(prefix+name, testingAlertsDelay, alert)
		} else {
			svc.l.Error(err)
		}
	}

	agentVersion, err := version.Parse(pointer.GetString(agent.Version))
	if err != nil {
		svc.l.Error(err)
	}
	if agentVersion != nil && agentVersion.Less(svc.serverVersion) {
		name, alert, err := makeAlertPMMAgentIsOutdated(agent, node, svc.serverVersion.String())
		if err == nil {
			svc.r.Add(prefix+name, testingAlertsDelay, alert)
		} else {
			svc.l.Error(err)
		}
	}
}

// sendAlerts sends alerts collected in the registry.
func (svc *Service) sendAlerts(ctx context.Context) {
	alerts := svc.r.Collect()
	if len(alerts) == 0 {
		return
	}

	svc.l.Infof("Sending %d alerts...", len(alerts))
	_, err := amclient.Default.Alert.PostAlerts(&alert.PostAlertsParams{
		Alerts:  alerts,
		Context: ctx,
	})
	if err != nil {
		svc.l.Error(err)
	}
}

// IsReady verifies that Alertmanager works.
func (svc *Service) IsReady(ctx context.Context) error {
	_, err := amclient.Default.General.GetStatus(&general.GetStatusParams{
		Context: ctx,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// configure default client; we use it mainly because we can't remove it from generated code
//nolint:gochecknoinits
func init() {
	amclient.Default.SetTransport(httptransport.New("127.0.0.1:9093", "/alertmanager/api/v2", []string{"http"}))
}
