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

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/percona/pmm/managed/models"
)

// RunReplicaSync keeps this HA replica and its own pmm-agent in step with changes served by other replicas.
func (s *Server) RunReplicaSync(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.syncReplica(ctx)
		}
	}
}

// syncReplica applies settings that changed since this replica last applied them, then resends
// its own pmm-agent the state, which no other replica can reach.
func (s *Server) syncReplica(ctx context.Context) {
	changed, err := s.settingsChanged()
	if err != nil {
		s.l.Warnf("Couldn't check settings: %s.", err)
	}

	if changed {
		s.l.Info("Settings differ from the ones this replica applied, applying them.")
		err = s.applyConfigurations()
		if err != nil {
			s.l.Warnf("Couldn't apply settings: %s.", err)
		}
	}

	s.agentsState.RequestStateUpdate(ctx, models.PMMServerAgentID)
}

func (s *Server) settingsChanged() (bool, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return false, fmt.Errorf("failed to get settings: %w", err)
	}
	current, err := json.Marshal(settings) //nolint:musttag
	if err != nil {
		return false, fmt.Errorf("failed to marshal settings: %w", err)
	}

	s.configM.Lock()
	defer s.configM.Unlock()

	return !bytes.Equal(current, s.appliedSettings), nil
}
