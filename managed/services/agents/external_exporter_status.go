// Copyright (C) 2023 Percona LLC
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

package agents

import (
	"context"
	"time"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
)

const (
	// statusUpdateInterval is how often we check VictoriaMetrics for external exporter status
	statusUpdateInterval = 30 * time.Second
)

// ExternalExporterStatusService periodically updates status for external exporters
// by querying VictoriaMetrics for 'up' metrics.
type ExternalExporterStatusService struct {
	db       *reform.DB
	vmClient victoriaMetricsClient
	l        *logrus.Entry
}

// NewExternalExporterStatusService creates a new service for updating external exporter statuses.
func NewExternalExporterStatusService(db *reform.DB, vmClient victoriaMetricsClient) *ExternalExporterStatusService {
	return &ExternalExporterStatusService{
		db:       db,
		vmClient: vmClient,
		l:        logrus.WithField("component", "external-exporter-status"),
	}
}

// Run starts the periodic status update loop.
// It runs every statusUpdateInterval (30 seconds) and updates all external exporter statuses.
func (s *ExternalExporterStatusService) Run(ctx context.Context) {
	ticker := time.NewTicker(statusUpdateInterval)
	defer ticker.Stop()

	s.l.Info("External exporter status updater started.")

	for {
		s.updateAllExternalExporterStatuses(ctx)

		select {
		case <-ctx.Done():
			s.l.Info("External exporter status updater stopped.")
			return
		case <-ticker.C:
		}
	}
}

// updateAllExternalExporterStatuses queries VictoriaMetrics for all external exporter 'up' metrics
// and updates their status in the database.
func (s *ExternalExporterStatusService) updateAllExternalExporterStatuses(ctx context.Context) {
	// Query VictoriaMetrics for all external exporter 'up' metrics
	query := `up{agent_type="external-exporter"}`
	result, _, err := s.vmClient.Query(ctx, query, time.Now())
	if err != nil {
		s.l.Warnf("Failed to query VictoriaMetrics for external exporter status: %v", err)
		return
	}

	statusMap := make(map[string]inventoryv1.AgentStatus)
	if vector, ok := result.(model.Vector); ok {
		for _, sample := range vector {
			agentID := string(sample.Metric[model.LabelName("agent_id")])
			if agentID == "" {
				continue
			}

			if sample.Value == 1 {
				statusMap[agentID] = inventoryv1.AgentStatus_AGENT_STATUS_RUNNING
			} else {
				statusMap[agentID] = inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN
			}
		}
	}

	s.l.Debugf("Updating status for %d external exporters.", len(statusMap))

	err = s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		for agentID, status := range statusMap {
			agent := &models.Agent{AgentID: agentID}
			if err := tx.Reload(agent); err != nil {
				// Agent might have been deleted, skip it
				s.l.Debugf("Agent %s not found, skipping status update.", agentID)
				continue
			}

			if agent.AgentType != models.ExternalExporterType {
				s.l.Warnf("Agent %s has agent_type label but is not ExternalExporterType, skipping.", agentID)
				continue
			}

			newStatus := status.String()
			if agent.Status != newStatus {
				agent.Status = newStatus
				if err := tx.Update(agent); err != nil {
					s.l.Errorf("Failed to update status for agent %s: %v", agentID, err)
					continue
				}
				s.l.Debugf("Updated agent %s status to %s.", agentID, newStatus)
			}
		}
		return nil
	})
	if err != nil {
		s.l.Errorf("Transaction failed while updating external exporter statuses: %v", err)
	}
}
