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
	"sort"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const (
	rdsPrefix = "rds/"
)

// roster groups several Agent IDs from an Inventory model to a single Group ID, as seen by pmm-agent.
//
// Currently, it is used only for rds_exporter.
// TODO Revisit it once we need it for something else.
type roster struct {
	l *logrus.Entry

	db *reform.DB
	rw sync.RWMutex
	m  map[string][]string
}

// newRoster creates a new roster instance.
func newRoster(db *reform.DB) *roster {
	return &roster{
		db: db,
		l:  logrus.WithField("component", "roster"),
		m:  make(map[string][]string),
	}
}

// add adds a new group of exporter IDs to the roster.
func (r *roster) add(pmmAgentID string, group string, exporters map[*models.Node]*models.Agent) string {
	r.rw.Lock()
	defer r.rw.Unlock()

	groupID := pmmAgentID + ":" + group
	exporterIDs := make([]string, 0, len(exporters))
	for _, exporter := range exporters {
		exporterIDs = append(exporterIDs, exporter.AgentID)
	}

	sort.Strings(exporterIDs)

	r.m[groupID] = exporterIDs
	r.l.Debugf("add: %s = %v", groupID, exporterIDs)
	return groupID
}

// get returns a PMMAgentID and a group of exporter IDs for a given Group ID.
func (r *roster) get(groupID string) (string, []string, error) {
	r.rw.RLock()
	defer r.rw.RUnlock()

	parts := strings.Split(groupID, ":")
	ok := len(parts) == 2

	PMMAgentID := parts[0]
	agentIDs := r.m[groupID]

	if agentIDs == nil {
		if !ok {
			agentIDs = []string{PMMAgentID}
		} else {
			rdsExporterType := models.RDSExporterType
			awsAccessKey := strings.TrimPrefix(parts[1], rdsPrefix)
			filters := models.AgentFilters{PMMAgentID: PMMAgentID, AgentType: &rdsExporterType, AWSAccessKey: awsAccessKey}
			agents, err := models.FindAgents(r.db.Querier, filters)
			if err != nil {
				return "", nil, err
			}
			agentIDs = make([]string, 0, len(agents))
			for _, agent := range agents {
				agentIDs = append(agentIDs, agent.AgentID)
			}
		}
	}

	r.l.Debugf("get: %s = %v", groupID, agentIDs)
	return PMMAgentID, agentIDs, nil
}

// clear removes the group of exporter IDs for a given PMM Agent ID.
func (r *roster) clear(pmmAgentID string) {
	r.rw.Lock()
	defer r.rw.Unlock()

	prefix := pmmAgentID + ":"
	var toDelete []string
	for groupID := range r.m {
		if strings.HasPrefix(groupID, prefix) {
			toDelete = append(toDelete, groupID)
		}
	}
	for _, groupID := range toDelete {
		delete(r.m, groupID)
	}

	r.l.Debugf("clear: %q", pmmAgentID)
}
