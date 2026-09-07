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

// bootstrapPollInterval is how often the leader re-reads every in-flight (and
// just-succeeded) bootstrap run and decides what happens next.
const bootstrapPollInterval = 15 * time.Second

// bootstrapMongoDBUsername is the one user every run's create_pmm_monitoring_user
// step creates -- see OmBootstrapSecret's own doc comment on why this is a single
// user, not a root account plus a separate PMM-only one (phase-1 scope,
// PMM-15347/questions.md Q7).
const bootstrapMongoDBUsername = "admin"

// RunBootstrapStepper drives every in-flight, and every just-succeeded but not yet
// PMM-registered, bootstrap run forward until ctx is cancelled.
//
// Registered as an HA-leader-only service in main() (ha.NewContextService), exactly
// like Run's own topology-collection ticker -- so exactly one PMM node decides what
// happens next for a run at a time (PMM-15347/plan.md §4 item 9). It persists none
// of that decision itself: every tick re-reads current state from SEP's
// om_bootstrap app via bootstrapClient, so a new leader after a failover simply
// resumes from wherever the last one left off, with no recovery logic of its own to
// run. The one thing it does persist -- the generated MongoDB user and keyFile
// (bootstrap_register.go, OmBootstrapSecret) -- exists specifically so that
// resumption is possible at all; see that type's own doc comment.
//
// A run reading SUCCEEDED in SEP is not necessarily done from PMM's side:
// registering the mongod with PMM's own inventory (registerBootstrapHost) is a step
// om_bootstrap knows nothing about, so every tick also revisits every succeeded
// run, not just the running ones, until that registration exists. Safe to repeat
// forever -- see registerBootstrapHost's own doc comment on why it is idempotent by
// construction rather than by a "was this already done" flag.
func (s *Service) RunBootstrapStepper(ctx context.Context) {
	ticker := time.NewTicker(bootstrapPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.bootstrap == nil || !s.Enabled() {
				continue
			}
			s.stepBootstrapRuns(ctx)
		}
	}
}

// stepBootstrapRuns discovers every run worth a look this tick and drives each one.
func (s *Service) stepBootstrapRuns(ctx context.Context) {
	for _, runStatus := range [...]string{bootstrapRunRunning, bootstrapRunSucceeded} {
		runs, err := s.bootstrap.listRuns(ctx, runStatus)
		if err != nil {
			s.l.Warnf("failed to list %s bootstrap runs: %s", runStatus, err)
			continue
		}
		for _, run := range runs {
			s.stepBootstrapRun(ctx, run.ID)
		}
	}
}

// stepBootstrapRun re-fetches one run (reconciling its in-flight dispatches as of
// this call) and drives it forward one tick's worth.
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
		// FAILED / ROLLED_BACK: terminal, and this stepper's own doing to get
		// there (finishBootstrapRun) -- nothing left to drive.
	}
}

// advanceRunningRun applies bootstrap_decision.go's pure decisions to one run: keep
// dispatching forward steps, or -- once anything has exhausted its retries --
// dispatch every host's rollback instead, until it is done.
func (s *Service) advanceRunningRun(ctx context.Context, run *sepBootstrapRun) {
	if runIsRollingBack(*run) {
		if runRollbackDone(*run) {
			s.finishBootstrapRun(ctx, run, bootstrapRunRolledBack,
				"a step exhausted its retries; every host was rolled back")
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

// dispatchHost dispatches one host's forward step, resolving whatever secret
// params it needs first.
func (s *Service) dispatchHost(ctx context.Context, run *sepBootstrapRun, host, stepName string) {
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

// dispatchFinalize dispatches one host's finalize step, resolving whatever
// secret params it needs first -- the same shape as dispatchHost, just against
// om_bootstrap's finalize route. Only ever called once runStepsSucceeded is
// true; see advanceRunningRun.
func (s *Service) dispatchFinalize(ctx context.Context, run *sepBootstrapRun, host, stepName string) {
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

// dispatchRollback dispatches one host's rollback step. Rollback steps take no
// params -- see om_bootstrap's own dispatch_rollback_step doc comment.
func (s *Service) dispatchRollback(ctx context.Context, run *sepBootstrapRun, host, stepName string) {
	_, err := s.bootstrap.dispatchRollbackStep(ctx, run.ID, host, stepName)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to dispatch rollback %s on %s: %s", run.ID, stepName, host, err)
	}
}

// dispatchRunStep dispatches one run-level step, resolving whatever secret params
// it needs first.
func (s *Service) dispatchRunStep(ctx context.Context, run *sepBootstrapRun, stepName string) {
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

// finishBootstrapRun records the stepper's own decision that a run is done.
func (s *Service) finishBootstrapRun(ctx context.Context, run *sepBootstrapRun, runStatus, detail string) {
	_, err := s.bootstrap.finishRun(ctx, run.ID, runStatus, &detail)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to record it as %s: %s", run.ID, runStatus, err)
	}
}

// completeSucceededRun registers every host in a succeeded run with PMM's own
// inventory, using the run's stored secret. A run with no stored secret yet is not
// actually possible here -- create_pmm_monitoring_user cannot have succeeded
// without one having been generated first -- so that case is reported rather than
// silently skipped: it means the secret disappeared after being used, not that
// nothing needs registering.
//
// run.Hosts is keyed on Nomad executor host (TriggerHostBootstrap's own doc
// comment), not the node id PMM's own inventory needs -- nodeIDForExecutorHost
// resolves each one back before registering it.
func (s *Service) completeSucceededRun(ctx context.Context, run *sepBootstrapRun) {
	secret, err := models.FindOmBootstrapSecretByRunID(s.db.Querier, run.ID)
	if err != nil {
		s.l.Warnf("bootstrap run %s: succeeded but has no stored secret to register with: %s", run.ID, err)
		return
	}
	for _, host := range run.Hosts {
		nodeID, err := s.nodeIDForExecutorHost(ctx, host.Host)
		if err != nil {
			s.l.Warnf("bootstrap run %s: failed to resolve executor %s to a node id: %s", run.ID, host.Host, err)
			continue
		}
		err = s.registerBootstrapHost(ctx, nodeID, run.ReplicaSetName, secret.MongoDBUsername, secret.MongoDBPassword)
		if err != nil {
			s.l.Warnf("bootstrap run %s: failed to register %s with PMM: %s", run.ID, host.Host, err)
		}
	}
}

// paramsForStep resolves the per-dispatch secret params one step name needs, or
// nil for a step that needs none. The only two steps that do (distribute_keyfile,
// create_pmm_monitoring_user) both draw from the same generated-once-per-run
// secret -- see ensureBootstrapSecret.
func (s *Service) paramsForStep(ctx context.Context, runID, stepName string) (map[string]string, error) {
	switch stepName {
	case "distribute_keyfile":
		secret, err := s.ensureBootstrapSecret(ctx, runID)
		if err != nil {
			return nil, err
		}
		return map[string]string{"key_file_content": secret.KeyFile}, nil
	case "create_pmm_monitoring_user":
		secret, err := s.ensureBootstrapSecret(ctx, runID)
		if err != nil {
			return nil, err
		}
		return map[string]string{"username": secret.MongoDBUsername, "password": secret.MongoDBPassword}, nil
	default:
		return nil, nil //nolint:nilnil // "no params needed" is not an error
	}
}

// ensureBootstrapSecret returns runID's stored secret, generating and persisting
// one on first use. Safe to call repeatedly, including from a different leader
// after a failover, than the tick that first generated it -- it reads before it
// writes, and a losing insert (a rare double-tick race, two ticks both finding
// nothing stored) re-reads what the winner wrote rather than erroring.
func (s *Service) ensureBootstrapSecret(_ context.Context, runID string) (*models.OmBootstrapSecret, error) {
	secret, err := models.FindOmBootstrapSecretByRunID(s.db.Querier, runID)
	if err == nil {
		return secret, nil
	}
	if !errors.Is(err, models.ErrNotFound) {
		return nil, fmt.Errorf("failed to look up the stored secret: %w", err)
	}

	password, err := generateBootstrapSecret(24)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a password: %w", err)
	}
	keyFile, err := generateBootstrapSecret(756)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a keyFile: %w", err)
	}

	created := &models.OmBootstrapSecret{
		RunID:           runID,
		MongoDBUsername: bootstrapMongoDBUsername,
		MongoDBPassword: password,
		KeyFile:         keyFile,
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

// generateBootstrapSecret returns n bytes of randomness, base64-encoded -- both the
// generated MongoDB password and the keyFile content are used this way; MongoDB
// itself recommends a base64 string between 6 and 1024 characters for a keyFile.
func generateBootstrapSecret(n int) (string, error) {
	raw := make([]byte, n)
	_, err := rand.Read(raw)
	if err != nil {
		return "", err //nolint:wrapcheck
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}
