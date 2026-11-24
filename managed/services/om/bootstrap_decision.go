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
	"fmt"
	"slices"
	"strings"
)

// bootstrapMaxAttempts counts every dispatch including the first, so a failed step is retried once.
const bootstrapMaxAttempts = 2

func maxAttempts(step extensionsBootstrapStep) int {
	if step.Retryable != nil && !*step.Retryable {
		return 1
	}
	return bootstrapMaxAttempts
}

func exhausted(step extensionsBootstrapStep) bool {
	return step.Status == bootstrapStepFailed && step.AttemptCount >= maxAttempts(step)
}

type stepAction struct {
	name string
}

func nextHostAction(host extensionsBootstrapHost) *stepAction {
	for _, step := range host.Steps {
		switch step.Status {
		case bootstrapStepRunning:
			return nil
		case bootstrapStepPending:
			return &stepAction{name: step.Name}
		case bootstrapStepFailed:
			if step.AttemptCount < maxAttempts(step) {
				return &stepAction{name: step.Name}
			}
			return nil
		}
	}
	return nil
}

func nextFinalizeAction(host extensionsBootstrapHost) *stepAction {
	for _, step := range host.FinalizeSteps {
		switch step.Status {
		case bootstrapStepRunning:
			return nil
		case bootstrapStepPending:
			return &stepAction{name: step.Name}
		case bootstrapStepFailed:
			if step.AttemptCount < maxAttempts(step) {
				return &stepAction{name: step.Name}
			}
			return nil
		}
	}
	return nil
}

// nextRollbackAction returns nil for a rollback step that exhausted its retries: that needs an operator,
// so the stepper ends the run as failed instead of retrying forever.
func nextRollbackAction(host extensionsBootstrapHost) *stepAction {
	for _, step := range host.RollbackSteps {
		switch step.Status {
		case bootstrapStepRunning:
			return nil
		case bootstrapStepPending:
			return &stepAction{name: step.Name}
		case bootstrapStepFailed:
			if step.AttemptCount < maxAttempts(step) {
				return &stepAction{name: step.Name}
			}
			return nil
		}
	}
	return nil
}

// nextRunStepAction waits for every host to succeed: run-level steps such as rs.initiate need every member up.
func nextRunStepAction(run extensionsBootstrapRun) *stepAction {
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
			if step.AttemptCount < maxAttempts(step) {
				return &stepAction{name: step.Name}
			}
			return nil
		}
	}
	return nil
}

func hostSucceeded(host extensionsBootstrapHost) bool {
	for _, step := range host.Steps {
		if step.Status != bootstrapStepSucceeded && step.Status != bootstrapStepSkipped {
			return false
		}
	}
	return true
}

func runStepsSucceeded(run extensionsBootstrapRun) bool {
	for _, step := range run.RunSteps {
		if step.Status != bootstrapStepSucceeded && step.Status != bootstrapStepSkipped {
			return false
		}
	}
	return true
}

// hostExhaustedRetries counts finalize steps too: a host installed without authorization enabled is as
// unusable as one that never finished installing.
func hostExhaustedRetries(host extensionsBootstrapHost) bool {
	return slices.ContainsFunc(host.Steps, exhausted) || slices.ContainsFunc(host.FinalizeSteps, exhausted)
}

func rolledBackReason(run extensionsBootstrapRun) string {
	outcome := "every host was rolled back"
	if nothingRolledBack(run) {
		outcome = "nothing had been installed, so there was nothing to roll back"
	}
	return rollbackCause(run) + "; " + outcome
}

func rollbackFailedReason(run extensionsBootstrapRun) string {
	var failures, hosts []string
	for _, host := range run.Hosts {
		parts := exhaustedParts(host.Host, host.RollbackSteps)
		if len(parts) > 0 {
			failures = append(failures, parts...)
			hosts = append(hosts, host.Host)
		}
	}
	return fmt.Sprintf("%s; rollback failed: %s; clean up %s by hand",
		rollbackCause(run), strings.Join(failures, "; "), strings.Join(hosts, ", "))
}

func rollbackCause(run extensionsBootstrapRun) string {
	if !runExhaustedRetries(run) {
		return "an operator requested cancellation"
	}
	if summary := exhaustedStepsSummary(run); summary != "" {
		return summary
	}
	return "a step exhausted its retries"
}

// nothingRolledBack reports whether every rollback step was skipped, as happens when nothing was installed.
func nothingRolledBack(run extensionsBootstrapRun) bool {
	var seen bool
	for _, host := range run.Hosts {
		for _, step := range host.RollbackSteps {
			if step.Status != bootstrapStepSkipped {
				return false
			}
			seen = true
		}
	}
	return seen
}

func exhaustedStepsSummary(run extensionsBootstrapRun) string {
	parts := make([]string, 0, len(run.Hosts))
	for _, host := range run.Hosts {
		parts = append(parts, exhaustedParts(host.Host, slices.Concat(host.Steps, host.FinalizeSteps))...)
	}
	parts = append(parts, exhaustedParts("", run.RunSteps)...)
	return strings.Join(parts, "; ")
}

// exhaustedParts describes each step out of retries; host is empty for run-level steps.
func exhaustedParts(host string, steps []extensionsBootstrapStep) []string {
	var parts []string
	for _, step := range steps {
		if !exhausted(step) {
			continue
		}
		part := step.Name + " failed"
		if host != "" {
			part += " on " + host
		}
		if step.AttemptCount > 1 {
			part += fmt.Sprintf(" after %d attempts", step.AttemptCount)
		}
		if step.Detail != nil && *step.Detail != "" {
			part += ": " + *step.Detail
		}
		parts = append(parts, part)
	}
	return parts
}

func runStepsExhaustedRetries(run extensionsBootstrapRun) bool {
	return slices.ContainsFunc(run.RunSteps, exhausted)
}

// runNeedsRollback reports whether every host, including ones that succeeded, must roll back: a partly
// installed replica set is not valid to leave running. A cancel request counts as exhausted retries.
func runNeedsRollback(run extensionsBootstrapRun) bool {
	if run.CancelRequested {
		return true
	}
	if slices.ContainsFunc(run.Hosts, hostExhaustedRetries) {
		return true
	}
	return runStepsExhaustedRetries(run)
}

func runExhaustedRetries(run extensionsBootstrapRun) bool {
	if slices.ContainsFunc(run.Hosts, hostExhaustedRetries) {
		return true
	}
	return runStepsExhaustedRetries(run)
}

func hostRollbackStarted(host extensionsBootstrapHost) bool {
	for _, step := range host.RollbackSteps {
		if step.Status != bootstrapStepPending {
			return true
		}
	}
	return false
}

func hostRollbackDone(host extensionsBootstrapHost) bool {
	for _, step := range host.RollbackSteps {
		if step.Status != bootstrapStepSucceeded && step.Status != bootstrapStepSkipped {
			return false
		}
	}
	return true
}

func runIsRollingBack(run extensionsBootstrapRun) bool {
	if slices.ContainsFunc(run.Hosts, hostRollbackStarted) {
		return true
	}
	return runNeedsRollback(run)
}

func runStep(run extensionsBootstrapRun, name string) (extensionsBootstrapStep, bool) {
	i := slices.IndexFunc(run.RunSteps, func(step extensionsBootstrapStep) bool { return step.Name == name })
	if i < 0 {
		return extensionsBootstrapStep{}, false
	}
	return run.RunSteps[i], true
}

// adminPasswordSpent reports whether a run that planned create_admin_user can no longer dispatch it.
func adminPasswordSpent(run extensionsBootstrapRun) bool {
	step, ok := runStep(run, createAdminUserStep)
	if !ok {
		return false
	}
	if run.Status != bootstrapRunRunning || runIsRollingBack(run) {
		return true
	}
	return step.Status == bootstrapStepSucceeded || step.Status == bootstrapStepSkipped
}

// runRollbackFailed reports whether a rollback has gone as far as it can, with nothing left running
// and a step out of retries.
func runRollbackFailed(run extensionsBootstrapRun) bool {
	if slices.ContainsFunc(run.RunSteps, stepRunning) {
		return false
	}
	var failed bool
	for _, host := range run.Hosts {
		if nextRollbackAction(host) != nil ||
			slices.ContainsFunc(slices.Concat(host.Steps, host.FinalizeSteps, host.RollbackSteps), stepRunning) {
			return false
		}
		failed = failed || slices.ContainsFunc(host.RollbackSteps, exhausted)
	}
	return failed
}

func stepRunning(step extensionsBootstrapStep) bool {
	return step.Status == bootstrapStepRunning
}

func runRollbackDone(run extensionsBootstrapRun) bool {
	for _, host := range run.Hosts {
		if !hostRollbackDone(host) {
			return false
		}
	}
	return true
}
