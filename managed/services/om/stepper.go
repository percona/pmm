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

package om

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/percona/pmm/managed/models"
)

const bootstrapPollInterval = 15 * time.Second

// bootstrapInventoryRefreshWindow bounds how long a registered run waits in the sweep for the inventory
// app to show its service, so a host whose service never appears cannot pin the run there forever.
const bootstrapInventoryRefreshWindow = 5 * time.Minute

// A run with no finish time cannot be aged, so it is not held.
func refreshRetryWindowOpen(run *extensionsBootstrapRun, now time.Time) bool {
	if run.FinishedAt == nil {
		return false
	}
	return now.Sub(*run.FinishedAt) < bootstrapInventoryRefreshWindow
}

const bootstrapMongoDBUsername = "admin"

// Base64-encoded, the keyFile stays within MongoDB's 6-1024 character range.
const (
	bootstrapPasswordBytes = 24
	bootstrapKeyFileBytes  = 756
)

// RunBootstrapStepper drives bootstrap runs forward until ctx is cancelled; it must run on the HA leader only.
// Each tick re-reads run state from PMM Extensions, so a new leader after a failover resumes where the last stopped.
func (s *Service) RunBootstrapStepper(ctx context.Context) {
	ticker := time.NewTicker(bootstrapPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.stepBootstrapTick(ctx)
		}
	}
}

// stepBootstrapTick is one RunBootstrapStepper tick, skipped while OpenManager is off like Run's collection.
func (s *Service) stepBootstrapTick(ctx context.Context) {
	if s.bootstrap == nil || !s.Enabled() {
		return
	}
	s.stepBootstrapRuns(ctx)
}

// Succeeded runs stay SUCCEEDED forever, so skipping the ones PMM already registered keeps the sweep bounded.
func (s *Service) stepBootstrapRuns(ctx context.Context) {
	registered, err := models.FindRegisteredOmBootstrapRunIDs(s.db.Querier)
	if err != nil {
		s.l.Warnf("failed to list the bootstrap runs PMM has already registered: %s", err)
		return
	}

	for _, runStatus := range [...]string{bootstrapRunRunning, bootstrapRunSucceeded} {
		runs, err := s.bootstrap.listRuns(ctx, runStatus, 0)
		if err != nil {
			s.l.Warnf("failed to list %s bootstrap runs: %s", runStatus, err)
			continue
		}
		for _, run := range runs {
			if _, done := registered[run.ID]; done {
				continue
			}
			s.stepBootstrapRun(ctx, run.ID)
		}
	}
}

func (s *Service) stepBootstrapRun(ctx context.Context, runID string) {
	run, err := s.bootstrap.getRun(ctx, runID)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to fetch: %s", runID, err)
		return
	}

	switch run.Status {
	case bootstrapRunSucceeded:
		s.completeSucceededRun(ctx, run)
	case bootstrapRunRunning:
		s.advanceRunningRun(ctx, run)
	default:
		// FAILED and ROLLED_BACK are terminal.
	}
}

func (s *Service) advanceRunningRun(ctx context.Context, run *extensionsBootstrapRun) {
	if runIsRollingBack(*run) {
		if runRollbackDone(*run) {
			s.finishBootstrapRun(ctx, run, bootstrapRunRolledBack, rolledBackReason(*run))
			return
		}
		for _, host := range run.Hosts {
			if action := nextRollbackAction(host); action != nil {
				s.dispatchRollback(ctx, run, host.Host, action.name)
			} else if hostRollbackStarted(host) && !hostRollbackDone(host) {
				s.l.Warnf("bootstrap run %s: host %s's rollback is stuck; an operator needs to intervene",
					run.ID, host.Host)
			}
		}
		return
	}

	for _, host := range run.Hosts {
		if action := nextHostAction(host); action != nil {
			s.dispatchHost(ctx, run, host.Host, action.name)
		}
	}
	if action := nextRunStepAction(*run); action != nil {
		s.dispatchRunStep(ctx, run, action.name)
	}
	if runStepsSucceeded(*run) {
		for _, host := range run.Hosts {
			if action := nextFinalizeAction(host); action != nil {
				s.dispatchFinalize(ctx, run, host.Host, action.name)
			}
		}
	}
}

func (s *Service) dispatchHost(ctx context.Context, run *extensionsBootstrapRun, host, stepName string) {
	params, err := s.paramsForStep(ctx, run.ID, stepName)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to prepare %s on %s: %s", run.ID, stepName, host, err)
		return
	}
	_, err = s.bootstrap.dispatchStep(ctx, run.ID, host, stepName, params)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to dispatch %s on %s: %s", run.ID, stepName, host, err)
	}
}

func (s *Service) dispatchFinalize(ctx context.Context, run *extensionsBootstrapRun, host, stepName string) {
	params, err := s.paramsForStep(ctx, run.ID, stepName)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to prepare finalize step %s on %s: %s", run.ID, stepName, host, err)
		return
	}
	_, err = s.bootstrap.dispatchFinalizeStep(ctx, run.ID, host, stepName, params)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to dispatch finalize step %s on %s: %s", run.ID, stepName, host, err)
	}
}

func (s *Service) dispatchRollback(ctx context.Context, run *extensionsBootstrapRun, host, stepName string) {
	_, err := s.bootstrap.dispatchRollbackStep(ctx, run.ID, host, stepName)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to dispatch rollback %s on %s: %s", run.ID, stepName, host, err)
	}
}

func (s *Service) dispatchRunStep(ctx context.Context, run *extensionsBootstrapRun, stepName string) {
	params, err := s.paramsForStep(ctx, run.ID, stepName)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to prepare run-level step %s: %s", run.ID, stepName, err)
		return
	}
	_, err = s.bootstrap.dispatchRunStep(ctx, run.ID, stepName, params)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to dispatch run-level step %s: %s", run.ID, stepName, err)
	}
}

func (s *Service) finishBootstrapRun(ctx context.Context, run *extensionsBootstrapRun, runStatus, detail string) {
	_, err := s.bootstrap.finishRun(ctx, run.ID, runStatus, &detail)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to record it as %s: %s", run.ID, runStatus, err)
	}
}

// completeSucceededRun registers a succeeded run's hosts with PMM, then marks the run registered once the
// inventory app shows a service on each host or bootstrapInventoryRefreshWindow has passed.
func (s *Service) completeSucceededRun(ctx context.Context, run *extensionsBootstrapRun) {
	secret, err := models.FindOmBootstrapSecretByRunID(s.db.Querier, run.ID)
	if err != nil {
		s.l.Warnf("bootstrap run %s: succeeded but has no stored secret to register with: %s", run.ID, err)
		return
	}
	environment, cluster := s.bootstrapRunConfigLabels(run.ID)

	hostsByExecutor, err := s.inventoryHostsByExecutor(ctx)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to look up the inventory app's hosts: %s", run.ID, err)
		return
	}

	unconfirmed := make([]string, 0, len(run.Hosts))
	done := 0
	for _, host := range run.Hosts {
		inventoryHost, ok := hostsByExecutor[host.Host]
		if !ok {
			s.l.Warnf("bootstrap run %s: executor %s has no row in the inventory app yet", run.ID, host.Host)
			continue
		}
		err = s.registerBootstrapHost(ctx, inventoryHost.NodeID, host.Host, run.ReplicaSetName, environment, cluster,
			secret.MongoDBUsername.Reveal(), secret.MongoDBPassword.Reveal())
		if err != nil {
			s.l.Warnf("bootstrap run %s: failed to register %s with PMM: %s", run.ID, host.Host, err)
			continue
		}
		done++
		if len(inventoryHost.Services) == 0 {
			unconfirmed = append(unconfirmed, inventoryHost.NodeID)
		}
	}

	if len(unconfirmed) > 0 {
		// The probe resolves services from PMM Extensions' copy of the inventory, so sync before re-probing.
		s.triggerInventorySync(ctx)
		s.triggerScopedInventoryRefresh(ctx, unconfirmed)
	}

	if done < len(run.Hosts) {
		return
	}
	if len(unconfirmed) > 0 && refreshRetryWindowOpen(run, time.Now()) {
		// The sync answers 202 and completes in the background, so this tick's probe can miss the service;
		// keep the run in the sweep until a later tick sees it.
		return
	}
	err = models.MarkOmBootstrapRunRegistered(s.db.Querier, run.ID)
	if err != nil {
		s.l.Warnf("bootstrap run %s: registered every host, but failed to record it: %s", run.ID, err)
	}
}

func (s *Service) paramsForStep(ctx context.Context, runID, stepName string) (map[string]string, error) {
	switch stepName {
	case "distribute_keyfile":
		secret, err := s.ensureBootstrapSecret(ctx, runID)
		if err != nil {
			return nil, err
		}
		return map[string]string{"key_file_content": secret.KeyFile.Reveal()}, nil
	case "create_pmm_monitoring_user":
		secret, err := s.ensureBootstrapSecret(ctx, runID)
		if err != nil {
			return nil, err
		}
		return map[string]string{"username": secret.MongoDBUsername.Reveal(), "password": secret.MongoDBPassword.Reveal()}, nil
	default:
		return nil, nil //nolint:nilnil // "no params needed" is not an error
	}
}

// ensureBootstrapSecret returns runID's stored secret, generating it on first use.
// If two ticks race to create it, the losing insert returns the winner's secret.
func (s *Service) ensureBootstrapSecret(_ context.Context, runID string) (*models.OmBootstrapSecret, error) {
	secret, err := models.FindOmBootstrapSecretByRunID(s.db.Querier, runID)
	if err == nil {
		return secret, nil
	}
	if !errors.Is(err, models.ErrNotFound) {
		return nil, fmt.Errorf("failed to look up the stored secret: %w", err)
	}

	password, err := generateBootstrapSecret(bootstrapPasswordBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a password: %w", err)
	}
	keyFile, err := generateBootstrapSecret(bootstrapKeyFileBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a keyFile: %w", err)
	}

	created := &models.OmBootstrapSecret{
		RunID:           runID,
		MongoDBUsername: bootstrapMongoDBUsername,
		MongoDBPassword: models.EncryptedString(password),
		KeyFile:         models.EncryptedString(keyFile),
	}
	err = models.CreateOmBootstrapSecret(s.db.Querier, created)
	if err != nil {
		existing, findErr := models.FindOmBootstrapSecretByRunID(s.db.Querier, runID)
		if findErr == nil {
			return existing, nil
		}
		return nil, fmt.Errorf("failed to store the generated secret: %w", err)
	}
	return created, nil
}

func generateBootstrapSecret(n int) (string, error) {
	raw := make([]byte, n)
	_, err := rand.Read(raw)
	if err != nil {
		return "", err //nolint:wrapcheck
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}
