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

package agents

import (
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

type agentGroup string

const (
	rdsGroup = agentGroup("rds")
)

// roster groups several Agent IDs from an Inventory model to a single Group ID, as seen by pmm-agent.
//
// Currently, it is used only for rds_exporter.
// TODO Revisit it once we need it for something else.
type roster struct {
	l *logrus.Entry

	rw sync.RWMutex
	m  map[string][]string
}

func newRoster() *roster {
	return &roster{
		l: logrus.WithField("component", "roster"),
		m: make(map[string][]string),
	}
}

func (r *roster) add(pmmAgentID string, group agentGroup, agentIDs []string) (groupID string) {
	r.rw.Lock()
	defer r.rw.Unlock()

	groupID = pmmAgentID + "/" + string(group)
	r.m[groupID] = agentIDs
	r.l.Debugf("add: %s = %v", groupID, agentIDs)
	return
}

func (r *roster) get(groupID string) (agentIDs []string) {
	r.rw.RLock()
	defer r.rw.RUnlock()

	agentIDs = r.m[groupID]
	r.l.Debugf("Get: %s = %v", groupID, agentIDs)
	return agentIDs
}

func (r *roster) clear(pmmAgentID string) {
	r.rw.Lock()
	defer r.rw.Unlock()

	prefix := pmmAgentID + "/"
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
