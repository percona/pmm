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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func step(name, status string, attempts int) extensionsBootstrapStep {
	return extensionsBootstrapStep{Name: name, Status: status, AttemptCount: attempts}
}

// checkStep is a step om_bootstrap marked not retryable, as it marks pre_check.
func checkStep(name, status string, attempts int) extensionsBootstrapStep {
	retryable := false
	s := step(name, status, attempts)
	s.Retryable = &retryable
	return s
}

func TestNextHostAction(t *testing.T) {
	t.Run("dispatches the first pending step", func(t *testing.T) {
		host := extensionsBootstrapHost{Steps: []extensionsBootstrapStep{
			step("pre_check", bootstrapStepSucceeded, 1),
			step("configure_repository", bootstrapStepPending, 0),
		}}
		action := nextHostAction(host)
		require.NotNil(t, action)
		assert.Equal(t, "configure_repository", action.name)
	})

	t.Run("waits while a step is running", func(t *testing.T) {
		host := extensionsBootstrapHost{Steps: []extensionsBootstrapStep{
			step("pre_check", bootstrapStepRunning, 1),
			step("configure_repository", bootstrapStepPending, 0),
		}}
		assert.Nil(t, nextHostAction(host))
	})

	t.Run("retries a failed step under the attempt cap", func(t *testing.T) {
		host := extensionsBootstrapHost{Steps: []extensionsBootstrapStep{
			step("install_package", bootstrapStepFailed, 1),
		}}
		action := nextHostAction(host)
		require.NotNil(t, action)
		assert.Equal(t, "install_package", action.name)
	})

	t.Run("gives up once the attempt cap is reached", func(t *testing.T) {
		host := extensionsBootstrapHost{Steps: []extensionsBootstrapStep{
			step("install_package", bootstrapStepFailed, bootstrapMaxAttempts),
		}}
		assert.Nil(t, nextHostAction(host))
	})

	t.Run("does not retry a step marked not retryable", func(t *testing.T) {
		host := extensionsBootstrapHost{Steps: []extensionsBootstrapStep{
			checkStep("pre_check", bootstrapStepFailed, 1),
			step("configure_repository", bootstrapStepPending, 0),
		}}
		assert.Nil(t, nextHostAction(host))
		assert.True(t, hostExhaustedRetries(host))
	})

	t.Run("has nothing to do once every step succeeded or was skipped", func(t *testing.T) {
		host := extensionsBootstrapHost{Steps: []extensionsBootstrapStep{
			step("pre_check", bootstrapStepSucceeded, 1),
			step("verify", bootstrapStepSkipped, 0),
		}}
		assert.Nil(t, nextHostAction(host))
	})
}

func TestNextRunStepAction(t *testing.T) {
	succeededHost := extensionsBootstrapHost{Steps: []extensionsBootstrapStep{step("verify", bootstrapStepSucceeded, 1)}}
	pendingHost := extensionsBootstrapHost{Steps: []extensionsBootstrapStep{step("verify", bootstrapStepPending, 0)}}

	t.Run("waits until every host has succeeded", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts:    []extensionsBootstrapHost{succeededHost, pendingHost},
			RunSteps: []extensionsBootstrapStep{step("rs_initiate", bootstrapStepPending, 0)},
		}
		assert.Nil(t, nextRunStepAction(run))
	})

	t.Run("dispatches the first pending run-level step once every host is done", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts:    []extensionsBootstrapHost{succeededHost, succeededHost},
			RunSteps: []extensionsBootstrapStep{step("rs_initiate", bootstrapStepPending, 0)},
		}
		action := nextRunStepAction(run)
		require.NotNil(t, action)
		assert.Equal(t, "rs_initiate", action.name)
	})

	t.Run("moves to the next run-level step once the first succeeded", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts: []extensionsBootstrapHost{succeededHost},
			RunSteps: []extensionsBootstrapStep{
				step("rs_initiate", bootstrapStepSucceeded, 1),
				step("create_pmm_monitoring_user", bootstrapStepPending, 0),
			},
		}
		action := nextRunStepAction(run)
		require.NotNil(t, action)
		assert.Equal(t, "create_pmm_monitoring_user", action.name)
	})

	t.Run("gives up once a run-level step exhausts its retries", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts:    []extensionsBootstrapHost{succeededHost},
			RunSteps: []extensionsBootstrapStep{step("rs_initiate", bootstrapStepFailed, bootstrapMaxAttempts)},
		}
		assert.Nil(t, nextRunStepAction(run))
	})
}

func TestNextFinalizeAction(t *testing.T) {
	t.Run("dispatches the first pending finalize step", func(t *testing.T) {
		host := extensionsBootstrapHost{FinalizeSteps: []extensionsBootstrapStep{
			step("enable_auth", bootstrapStepPending, 0),
		}}
		action := nextFinalizeAction(host)
		require.NotNil(t, action)
		assert.Equal(t, "enable_auth", action.name)
	})

	t.Run("waits while a finalize step is running", func(t *testing.T) {
		host := extensionsBootstrapHost{FinalizeSteps: []extensionsBootstrapStep{
			step("enable_auth", bootstrapStepRunning, 1),
		}}
		assert.Nil(t, nextFinalizeAction(host))
	})

	t.Run("retries a failed finalize step under the attempt cap", func(t *testing.T) {
		host := extensionsBootstrapHost{FinalizeSteps: []extensionsBootstrapStep{
			step("enable_auth", bootstrapStepFailed, 1),
		}}
		action := nextFinalizeAction(host)
		require.NotNil(t, action)
		assert.Equal(t, "enable_auth", action.name)
	})

	t.Run("gives up once the attempt cap is reached", func(t *testing.T) {
		host := extensionsBootstrapHost{FinalizeSteps: []extensionsBootstrapStep{
			step("enable_auth", bootstrapStepFailed, bootstrapMaxAttempts),
		}}
		assert.Nil(t, nextFinalizeAction(host))
	})

	t.Run("has nothing to do once every finalize step succeeded", func(t *testing.T) {
		host := extensionsBootstrapHost{FinalizeSteps: []extensionsBootstrapStep{
			step("enable_auth", bootstrapStepSucceeded, 1),
		}}
		assert.Nil(t, nextFinalizeAction(host))
	})
}

func TestRunStepsSucceeded(t *testing.T) {
	t.Run("false while a run-level step is still pending", func(t *testing.T) {
		run := extensionsBootstrapRun{RunSteps: []extensionsBootstrapStep{
			step("rs_initiate", bootstrapStepSucceeded, 1),
			step("create_pmm_monitoring_user", bootstrapStepPending, 0),
		}}
		assert.False(t, runStepsSucceeded(run))
	})

	t.Run("true once every run-level step succeeded or was skipped", func(t *testing.T) {
		run := extensionsBootstrapRun{RunSteps: []extensionsBootstrapStep{
			step("rs_initiate", bootstrapStepSucceeded, 1),
			step("create_pmm_monitoring_user", bootstrapStepSucceeded, 1),
		}}
		assert.True(t, runStepsSucceeded(run))
	})

	t.Run("true when there are no run-level steps at all", func(t *testing.T) {
		assert.True(t, runStepsSucceeded(extensionsBootstrapRun{}))
	})
}

func TestRunNeedsRollback(t *testing.T) {
	t.Run("false while every step is still pending, running, or succeeded", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts: []extensionsBootstrapHost{{Steps: []extensionsBootstrapStep{step("pre_check", bootstrapStepRunning, 1)}}},
		}
		assert.False(t, runNeedsRollback(run))
	})

	t.Run("true once a host exhausts a step's retries", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts: []extensionsBootstrapHost{{Steps: []extensionsBootstrapStep{
				step("install_package", bootstrapStepFailed, bootstrapMaxAttempts),
			}}},
		}
		assert.True(t, runNeedsRollback(run))
	})

	t.Run("true once a run-level step exhausts its retries", func(t *testing.T) {
		run := extensionsBootstrapRun{
			RunSteps: []extensionsBootstrapStep{step("rs_initiate", bootstrapStepFailed, bootstrapMaxAttempts)},
		}
		assert.True(t, runNeedsRollback(run))
	})

	t.Run("false while a failed step still has a retry left", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts: []extensionsBootstrapHost{{Steps: []extensionsBootstrapStep{step("install_package", bootstrapStepFailed, 1)}}},
		}
		assert.False(t, runNeedsRollback(run))
	})

	t.Run("true once a host's finalize step exhausts its retries", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts: []extensionsBootstrapHost{{
				Steps:         []extensionsBootstrapStep{step("verify", bootstrapStepSucceeded, 1)},
				FinalizeSteps: []extensionsBootstrapStep{step("enable_auth", bootstrapStepFailed, bootstrapMaxAttempts)},
			}},
		}
		assert.True(t, runNeedsRollback(run))
	})

	t.Run("true once an operator has requested cancellation, even with nothing failed", func(t *testing.T) {
		run := extensionsBootstrapRun{
			CancelRequested: true,
			Hosts:           []extensionsBootstrapHost{{Steps: []extensionsBootstrapStep{step("pre_check", bootstrapStepRunning, 1)}}},
		}
		assert.True(t, runNeedsRollback(run))
	})
}

func TestRunIsRollingBack(t *testing.T) {
	t.Run("true once any host's rollback has started, even without a fresh failure", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts: []extensionsBootstrapHost{
				{
					Steps:         []extensionsBootstrapStep{step("install_package", bootstrapStepFailed, bootstrapMaxAttempts)},
					RollbackSteps: []extensionsBootstrapStep{step("stop_service", bootstrapStepRunning, 1)},
				},
				// A second, otherwise-healthy host: still counts as rolling back
				// because rollback means every host, not just the failed one.
				{
					Steps:         []extensionsBootstrapStep{step("verify", bootstrapStepSucceeded, 1)},
					RollbackSteps: []extensionsBootstrapStep{step("stop_service", bootstrapStepPending, 0)},
				},
			},
		}
		assert.True(t, runIsRollingBack(run))
	})

	t.Run("false when nothing has failed and no rollback has started", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts: []extensionsBootstrapHost{{
				Steps:         []extensionsBootstrapStep{step("pre_check", bootstrapStepPending, 0)},
				RollbackSteps: []extensionsBootstrapStep{step("stop_service", bootstrapStepPending, 0)},
			}},
		}
		assert.False(t, runIsRollingBack(run))
	})
}

func TestRunRollbackDone(t *testing.T) {
	t.Run("false while any host's rollback is still in flight", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts: []extensionsBootstrapHost{
				{RollbackSteps: []extensionsBootstrapStep{step("stop_service", bootstrapStepSucceeded, 1)}},
				{RollbackSteps: []extensionsBootstrapStep{step("stop_service", bootstrapStepRunning, 1)}},
			},
		}
		assert.False(t, runRollbackDone(run))
	})

	t.Run("true once every host's rollback steps have all succeeded or been skipped", func(t *testing.T) {
		run := extensionsBootstrapRun{
			Hosts: []extensionsBootstrapHost{
				{RollbackSteps: []extensionsBootstrapStep{step("stop_service", bootstrapStepSucceeded, 1)}},
				{RollbackSteps: []extensionsBootstrapStep{step("stop_service", bootstrapStepSkipped, 0)}},
			},
		}
		assert.True(t, runRollbackDone(run))
	})
}

func TestNextRollbackAction(t *testing.T) {
	t.Run("dispatches the first pending rollback step", func(t *testing.T) {
		host := extensionsBootstrapHost{RollbackSteps: []extensionsBootstrapStep{
			step("stop_service", bootstrapStepPending, 0),
			step("remove_config", bootstrapStepPending, 0),
		}}
		action := nextRollbackAction(host)
		require.NotNil(t, action)
		assert.Equal(t, "stop_service", action.name)
	})

	t.Run("gives up when a rollback step itself exhausts its retries", func(t *testing.T) {
		host := extensionsBootstrapHost{RollbackSteps: []extensionsBootstrapStep{
			step("stop_service", bootstrapStepFailed, bootstrapMaxAttempts),
		}}
		assert.Nil(t, nextRollbackAction(host))
	})
}

func TestExhaustedStepsSummary(t *testing.T) {
	t.Parallel()

	detail := func(text string) *string { return &text }

	t.Run("names every exhausted step with its host and detail", func(t *testing.T) {
		t.Parallel()

		disk := step("pre_check", bootstrapStepFailed, bootstrapMaxAttempts)
		disk.Detail = detail("pre_check: less than 5368709120 bytes free for the data directory")
		repo := step("configure_repository", bootstrapStepFailed, bootstrapMaxAttempts)
		run := extensionsBootstrapRun{
			Hosts: []extensionsBootstrapHost{
				{Host: "node00", Steps: []extensionsBootstrapStep{disk}},
				{Host: "node01", Steps: []extensionsBootstrapStep{
					step("pre_check", bootstrapStepSucceeded, 1),
					repo,
				}},
			},
			RunSteps: []extensionsBootstrapStep{step("rs_initiate", bootstrapStepFailed, bootstrapMaxAttempts)},
		}

		assert.Equal(t,
			"pre_check failed on node00 after 2 attempts: pre_check: less than 5368709120 bytes free for the data directory; "+
				"configure_repository failed on node01 after 2 attempts; "+
				"rs_initiate failed after 2 attempts",
			exhaustedStepsSummary(run))
	})

	t.Run("a step that had one attempt does not count it", func(t *testing.T) {
		t.Parallel()

		taken := checkStep("pre_check", bootstrapStepFailed, 1)
		taken.Detail = detail("pre_check: port 27017 is already in use (pid 31, python3) (task history 33)")
		run := extensionsBootstrapRun{Hosts: []extensionsBootstrapHost{
			{Host: "node00", Steps: []extensionsBootstrapStep{taken}},
		}}

		assert.Equal(t,
			"pre_check failed on node00: pre_check: port 27017 is already in use (pid 31, python3) (task history 33)",
			exhaustedStepsSummary(run))
	})

	t.Run("a step with attempts left is not reported", func(t *testing.T) {
		t.Parallel()

		run := extensionsBootstrapRun{Hosts: []extensionsBootstrapHost{
			{Host: "node00", Steps: []extensionsBootstrapStep{step("pre_check", bootstrapStepFailed, 1)}},
		}}

		assert.Empty(t, exhaustedStepsSummary(run))
	})
}

func TestRolledBackReason(t *testing.T) {
	t.Parallel()

	failedCheck := func(rollback ...extensionsBootstrapStep) extensionsBootstrapRun {
		return extensionsBootstrapRun{Hosts: []extensionsBootstrapHost{{
			Host:          "node00",
			Steps:         []extensionsBootstrapStep{checkStep("pre_check", bootstrapStepFailed, 1)},
			RollbackSteps: rollback,
		}}}
	}

	t.Run("says nothing was rolled back when every rollback step was skipped", func(t *testing.T) {
		t.Parallel()

		run := failedCheck(
			step("stop_service", bootstrapStepSkipped, 0),
			step("purge_package", bootstrapStepSkipped, 0),
		)

		assert.Equal(t,
			"pre_check failed on node00; nothing had been installed, so there was nothing to roll back",
			rolledBackReason(run))
	})

	t.Run("says every host was rolled back when any rollback step ran", func(t *testing.T) {
		t.Parallel()

		run := failedCheck(
			step("stop_service", bootstrapStepSucceeded, 1),
			step("purge_package", bootstrapStepSkipped, 0),
		)

		assert.Equal(t, "pre_check failed on node00; every host was rolled back", rolledBackReason(run))
	})
}
