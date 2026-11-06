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

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/utils/vault"
)

// WatchServerSecrets sends a new state to the pmm-agents, and a new configuration to VictoriaMetrics, when a
// secret that PMM Server reads from Vault for their Agents changes there, as a rotated Agent password or key
// does; the secrets pmm-agent reads itself, it watches itself. It checks them at the given interval until
// ctx is done.
func (u *StateUpdater) WatchServerSecrets(ctx context.Context, interval time.Duration) {
	l := logger.Get(ctx).WithField("component", "vault-secrets")
	t := time.NewTicker(interval)
	defer t.Stop()

	seen := make(map[string]string)
	for {
		changed := u.changedServerSecrets(ctx, seen, l)
		for pmmAgentID := range changed {
			u.RequestStateUpdate(ctx, pmmAgentID)
		}
		// Without the batch delay of a requested update: from the moment an exporter restarts with a new
		// password, VictoriaMetrics fails to scrape it with the old one.
		if len(changed) != 0 {
			err := u.vmdb.ForceConfigurationUpdate(ctx)
			if err != nil {
				l.Warnf("Cannot update the VictoriaMetrics configuration, requesting an update: %s.", err)
				u.vmdb.RequestConfigurationUpdate()
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// changedServerSecrets returns the pmm-agents of the Agents whose secrets read by PMM Server changed since the
// fingerprints of seen, which it updates. An Agent seen for the first time is not a change: its state was
// sent when it was added.
func (u *StateUpdater) changedServerSecrets(ctx context.Context, seen map[string]string, l *logrus.Entry) map[string]struct{} {
	agents, err := models.FindAgents(u.db.Querier, models.AgentFilters{})
	if err != nil {
		l.Errorf("Cannot check the secrets read from Vault: %s.", err)
		return nil
	}

	// Each secret is read from Vault again, once, rather than taken from the cache: a change is seen at this
	// check rather than at the next one.
	ctx = vault.ReadSince(ctx, time.Now())
	changed := make(map[string]struct{})
	present := make(map[string]struct{}, len(agents))
	for _, agent := range agents {
		if !agent.HasServerSecrets() {
			continue
		}
		present[agent.AgentID] = struct{}{}

		resolved, err := agent.WithServerSecrets(ctx)
		if err != nil {
			// Kept running with what it has, as pmm-agent does when Vault is unreachable.
			l.Warnf("Cannot check the secrets of %s %s: %s.", agent.AgentType, agent.AgentID, err)
			continue
		}

		fingerprint := resolved.ServerSecretsFingerprint()
		if previous, ok := seen[agent.AgentID]; ok && previous != fingerprint {
			l.Infof("Vault secrets of %s %s changed.", agent.AgentType, agent.AgentID)
			changed[pointer.GetString(agent.PMMAgentID)] = struct{}{}
		}
		seen[agent.AgentID] = fingerprint
	}

	for id := range seen {
		if _, ok := present[id]; !ok {
			delete(seen, id)
		}
	}
	delete(changed, "")
	return changed
}
