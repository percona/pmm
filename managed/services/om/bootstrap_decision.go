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

// This file is pure decision logic for the bootstrap stepper (stepper.go): given a
// run's current state (as om_bootstrap's own API reports it), what should happen
// next. No network, no database, no clock -- every function here is a plain
// function of its arguments, unit-testable without a fake SEP or a database, the
// same split SEP's own strategy.py/reconcile.py commit to on the other side of this
// same feature.
//
// Adamo's decided partial-failure policy (PMM-15347/questions.md Q8): retry a
// failed step once, and if retries are exhausted, roll back -- and "roll back",
// read plainly, means every host, not just the one that failed: a replica set with
// two members up and one torn down is not a valid state to leave running. So once
// any host (or any run-level step) exhausts its retries, every host starts rolling
// back, including ones that had already fully succeeded.

// bootstrapMaxAttempts caps a step's attempt_count before its failure is treated as
// permanent. attempt_count counts every dispatch including the first, so 2 means
// "the first attempt failed, and so did the one retry."
const bootstrapMaxAttempts = 2

// stepAction names one step the stepper has decided to dispatch next.
type stepAction struct {
	name string
}

// nextHostAction returns the next forward step to dispatch for host, or nil when
// there is nothing to do: a step is already running, every step has succeeded or
// been skipped, or a step has failed with retries exhausted (that failure is a
// run-level rollback decision, not this host's own -- see runNeedsRollback).
func nextHostAction(host sepBootstrapHost) *stepAction {
	for _, step := range host.Steps {
		switch step.Status {
		case bootstrapStepRunning:
			return nil
		case bootstrapStepPending:
			return &stepAction{name: step.Name}
		case bootstrapStepFailed:
			if step.AttemptCount < bootstrapMaxAttempts {
				return &stepAction{name: step.Name}
			}
			return nil
		}
	}
	return nil
}

// nextRollbackAction returns the next rollback step to dispatch for host, or nil
// when there is nothing to do: a rollback step is already running, every rollback
// step has succeeded or been skipped (rollback is done -- see hostRollbackDone), or
// a rollback step has itself failed with retries exhausted. The last case is not
// retried further: an operator has to intervene, so the stepper reports it (see
// stepper.go) rather than looping on it forever.
func nextRollbackAction(host sepBootstrapHost) *stepAction {
	for _, step := range host.RollbackSteps {
		switch step.Status {
		case bootstrapStepRunning:
			return nil
		case bootstrapStepPending:
			return &stepAction{name: step.Name}
		case bootstrapStepFailed:
			if step.AttemptCount < bootstrapMaxAttempts {
				return &stepAction{name: step.Name}
			}
			return nil
		}
	}
	return nil
}

// nextRunStepAction returns the next run-level step to dispatch, or nil when there
// is nothing to do: not every host has succeeded yet (run-level steps need every
// member up first -- rs.initiate, most concretely), a run-level step is already
// running, every run-level step has succeeded or been skipped, or one has failed
// with retries exhausted (again a run-level rollback decision, not returned here).
func nextRunStepAction(run sepBootstrapRun) *stepAction {
	for _, host := range run.Hosts {
		if !hostSucceeded(host) {
			return nil
		}
	}
	for _, step := range run.RunSteps {
		switch step.Status {
		case bootstrapStepRunning:
			return nil
		case bootstrapStepPending:
			return &stepAction{name: step.Name}
		case bootstrapStepFailed:
			if step.AttemptCount < bootstrapMaxAttempts {
				return &stepAction{name: step.Name}
			}
			return nil
		}
	}
	return nil
}

// hostSucceeded reports whether every one of host's forward steps succeeded or was
// skipped -- the same derivation om_bootstrap's own HostBootstrapState.status
// property makes on SEP's side, needed here too since nextRunStepAction gates on it
// without a round trip back to SEP.
func hostSucceeded(host sepBootstrapHost) bool {
	for _, step := range host.Steps {
		if step.Status != bootstrapStepSucceeded && step.Status != bootstrapStepSkipped {
			return false
		}
	}
	return true
}

// hostExhaustedRetries reports whether host has a forward step that failed with no
// retries left.
func hostExhaustedRetries(host sepBootstrapHost) bool {
	for _, step := range host.Steps {
		if step.Status == bootstrapStepFailed && step.AttemptCount >= bootstrapMaxAttempts {
			return true
		}
	}
	return false
}

// runStepsExhaustedRetries reports whether run has a run-level step that failed
// with no retries left.
func runStepsExhaustedRetries(run sepBootstrapRun) bool {
	for _, step := range run.RunSteps {
		if step.Status == bootstrapStepFailed && step.AttemptCount >= bootstrapMaxAttempts {
			return true
		}
	}
	return false
}

// runNeedsRollback reports whether run has just now exhausted retries somewhere --
// a host's forward step, or a run-level step -- and should start rolling every host
// back. Does not itself check whether rollback has already started; see
// runIsRollingBack, which callers use instead.
func runNeedsRollback(run sepBootstrapRun) bool {
	for _, host := range run.Hosts {
		if hostExhaustedRetries(host) {
			return true
		}
	}
	return runStepsExhaustedRetries(run)
}

// hostRollbackStarted reports whether any of host's rollback steps has ever been
// dispatched -- the same "not still all pending" test as any other planned-up-front
// step list.
func hostRollbackStarted(host sepBootstrapHost) bool {
	for _, step := range host.RollbackSteps {
		if step.Status != bootstrapStepPending {
			return true
		}
	}
	return false
}

// hostRollbackDone reports whether every one of host's rollback steps succeeded or
// was skipped.
func hostRollbackDone(host sepBootstrapHost) bool {
	for _, step := range host.RollbackSteps {
		if step.Status != bootstrapStepSucceeded && step.Status != bootstrapStepSkipped {
			return false
		}
	}
	return true
}

// runIsRollingBack reports whether run has already started rolling back (any host's
// rollback under way), or should start now (something just exhausted its retries).
// Once true, the stepper drives every host's rollback and dispatches no forward
// step anywhere in the run -- see the module doc comment on why "roll back" means
// every host, not just the one that failed.
func runIsRollingBack(run sepBootstrapRun) bool {
	for _, host := range run.Hosts {
		if hostRollbackStarted(host) {
			return true
		}
	}
	return runNeedsRollback(run)
}

// runRollbackDone reports whether every host's rollback has finished -- the point
// at which the stepper records the run itself as ROLLED_BACK.
func runRollbackDone(run sepBootstrapRun) bool {
	for _, host := range run.Hosts {
		if !hostRollbackDone(host) {
			return false
		}
	}
	return true
}
