// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package supervisor

import (
	"maps"
	"slices"
	"time"

	"github.com/percona/pmm/agent/utils/secrets"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/utils/vault"
)

// processStrings returns the parameters of an Agent process that may hold Vault tokens, in a stable order.
func (s *Supervisor) processStrings(p *agentv1.SetStateRequest_AgentProcess) []string {
	res := slices.Concat(p.Args, p.Env)
	for _, name := range slices.Sorted(maps.Keys(p.TextFiles)) {
		res = append(res, p.TextFiles[name])
	}

	// vmagent gets the PMM Server password, which may be read from Vault too, see processParams.
	if p.Type == inventoryv1.AgentType_AGENT_TYPE_VM_AGENT {
		ref, err := vault.ParseRef(s.cfg.Get().Server.Password)
		if err == nil {
			res = append(res, vault.Token(ref.Path, ref.Key, vault.EncodingRaw))
		}
	}
	return res
}

// builtinStrings returns the parameters of a built-in Agent that may hold Vault tokens, in a stable order.
func builtinStrings(b *agentv1.SetStateRequest_BuiltinAgent) []string {
	files := b.GetTextFiles().GetFiles()
	res := make([]string, 0, 1+len(files))
	res = append(res, b.Dsn)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		res = append(res, files[name])
	}
	return res
}

func hasTokens(strs []string) bool {
	return slices.ContainsFunc(strs, vault.HasTokens)
}

// retryProcessLocked makes refreshSecrets retry an Agent process that failed to start, if its
// credentials are stored in Vault: Vault may only be unreachable for a while. The Agent is forgotten
// until then, so that a SetState with the same parameters starts it too instead of taking it for running.
// Must be called with s.rw held for writing.
func (s *Supervisor) retryProcessLocked(agentID string, agentProcess *agentv1.SetStateRequest_AgentProcess) {
	if !hasTokens(s.processStrings(agentProcess)) {
		return
	}

	if agent, ok := s.agentProcesses[agentID]; ok {
		delete(s.agentProcesses, agentID)
		s.releaseAgentResources(agentID, agent.done, agent.listenPort, "")
	}
	s.pendingProcesses[agentID] = agentProcess
}

// retryBuiltinLocked is retryProcessLocked for built-in Agents.
// Must be called with s.rw held for writing.
func (s *Supervisor) retryBuiltinLocked(agentID string, builtinAgent *agentv1.SetStateRequest_BuiltinAgent) {
	if !hasTokens(builtinStrings(builtinAgent)) {
		return
	}

	if agent, ok := s.builtinAgents[agentID]; ok {
		delete(s.builtinAgents, agentID)
		s.releaseAgentResources(agentID, agent.done, 0, "")
	}
	s.pendingBuiltins[agentID] = builtinAgent
}

// refreshSecrets restarts the running Agents whose Vault secrets changed since they started, so that
// rotated static credentials and renewed dynamic ones reach them, and retries the Agents that failed
// to start for want of their secrets. The secrets are read without holding s.rw, since Vault may be slow.
func (s *Supervisor) refreshSecrets() {
	type candidate struct {
		id          string
		builtin     bool
		state       any
		strs        []string
		fingerprint string
	}

	s.rw.RLock()
	var candidates []candidate
	for id, agent := range s.agentProcesses {
		if agent.secretsFingerprint != "" {
			candidates = append(candidates, candidate{
				id:          id,
				state:       agent.requestedState,
				strs:        s.processStrings(agent.requestedState),
				fingerprint: agent.secretsFingerprint,
			})
		}
	}
	for id, agent := range s.builtinAgents {
		if agent.secretsFingerprint != "" {
			candidates = append(candidates, candidate{
				id:          id,
				builtin:     true,
				state:       agent.requestedState,
				strs:        builtinStrings(agent.requestedState),
				fingerprint: agent.secretsFingerprint,
			})
		}
	}
	pending := len(s.pendingProcesses)+len(s.pendingBuiltins) != 0
	s.rw.RUnlock()

	// Each secret is read from Vault again, once, rather than taken from the cache: a change is seen at
	// this check rather than at the next one.
	ctx := vault.ReadSince(s.ctx, time.Now())
	var changed []candidate
	for _, c := range candidates {
		fingerprint, err := secrets.Fingerprint(ctx, c.strs...)
		if err != nil {
			s.l.WithField("agentID", c.id).Warnf("Cannot check the Vault secrets of the Agent, it keeps running: %s.", err)
			continue
		}
		if fingerprint != c.fingerprint {
			changed = append(changed, c)
		}
	}

	if len(changed) == 0 && !pending {
		return
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	// Only the Agents still running with the parameters they were checked with: a SetState in the
	// meantime has already restarted the others with fresh secrets, or stopped them.
	var processIDs, builtinIDs []string
	for _, c := range changed {
		if c.builtin {
			if agent, ok := s.builtinAgents[c.id]; ok && agent.requestedState == c.state {
				builtinIDs = append(builtinIDs, c.id)
			}
			continue
		}
		if agent, ok := s.agentProcesses[c.id]; ok && agent.requestedState == c.state {
			processIDs = append(processIDs, c.id)
		}
	}

	if len(processIDs)+len(builtinIDs) != 0 {
		s.l.Infof("Vault secrets changed, restarting %d agent processes and %d built-in agents.", len(processIDs), len(builtinIDs))
		s.restartAgentsLocked(processIDs, builtinIDs)
	}

	s.startPendingLocked()
}

// startPendingLocked starts the Agents that failed to start for want of their Vault secrets.
// Must be called with s.rw held for writing.
func (s *Supervisor) startPendingLocked() {
	for id, agentProcess := range s.pendingProcesses {
		err := s.tryStartProcess(id, agentProcess, 0)
		if err != nil {
			s.l.WithField("agentID", id).Warnf("Agent still cannot be started: %s.", err)
			continue
		}
		delete(s.pendingProcesses, id)
	}

	for id, builtinAgent := range s.pendingBuiltins {
		err := s.startBuiltin(id, builtinAgent)
		if err != nil {
			s.l.WithField("agentID", id).Warnf("Agent still cannot be started: %s.", err)
			continue
		}
		delete(s.pendingBuiltins, id)
	}
}
