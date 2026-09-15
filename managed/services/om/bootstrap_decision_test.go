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

func step(name, status string, attempts int) sepBootstrapStep {
	return sepBootstrapStep{Name: name, Status: status, AttemptCount: attempts}
}

func TestNextHostAction(t *testing.T) {
	t.Run("dispatches the first pending step", func(t *testing.T) {
		host := sepBootstrapHost{Steps: []sepBootstrapStep{
			step("pre_check", bootstrapStepSucceeded, 1),
			step("configure_repository", bootstrapStepPending, 0),
		}}
		action := nextHostAction(host)
		require.NotNil(t, action)
		assert.Equal(t, "configure_repository", action.name)
	})

	t.Run("waits while a step is running", func(t *testing.T) {
		host := sepBootstrapHost{Steps: []sepBootstrapStep{
			step("pre_check", bootstrapStepRunning, 1),
			step("configure_repository", bootstrapStepPending, 0),
		}}
		assert.Nil(t, nextHostAction(host))
	})

	t.Run("retries a failed step under the attempt cap", func(t *testing.T) {
		host := sepBootstrapHost{Steps: []sepBootstrapStep{
			step("install_package", bootstrapStepFailed, 1),
		}}
		action := nextHostAction(host)
		require.NotNil(t, action)
		assert.Equal(t, "install_package", action.name)
	})

	t.Run("gives up once the attempt cap is reached", func(t *testing.T) {
		host := sepBootstrapHost{Steps: []sepBootstrapStep{
			step("install_package", bootstrapStepFailed, bootstrapMaxAttempts),
		}}
		assert.Nil(t, nextHostAction(host))
	})

	t.Run("has nothing to do once every step succeeded or was skipped", func(t *testing.T) {
		host := sepBootstrapHost{Steps: []sepBootstrapStep{
			step("pre_check", bootstrapStepSucceeded, 1),
			step("verify", bootstrapStepSkipped, 0),
		}}
		assert.Nil(t, nextHostAction(host))
	})
}

func TestNextRunStepAction(t *testing.T) {
	succeededHost := sepBootstrapHost{Steps: []sepBootstrapStep{step("verify", bootstrapStepSucceeded, 1)}}
	pendingHost := sepBootstrapHost{Steps: []sepBootstrapStep{step("verify", bootstrapStepPending, 0)}}

	t.Run("waits until every host has succeeded", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts:    []sepBootstrapHost{succeededHost, pendingHost},
			RunSteps: []sepBootstrapStep{step("rs_initiate", bootstrapStepPending, 0)},
		}
		assert.Nil(t, nextRunStepAction(run))
	})

	t.Run("dispatches the first pending run-level step once every host is done", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts:    []sepBootstrapHost{succeededHost, succeededHost},
			RunSteps: []sepBootstrapStep{step("rs_initiate", bootstrapStepPending, 0)},
		}
		action := nextRunStepAction(run)
		require.NotNil(t, action)
		assert.Equal(t, "rs_initiate", action.name)
	})

	t.Run("moves to the next run-level step once the first succeeded", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts: []sepBootstrapHost{succeededHost},
			RunSteps: []sepBootstrapStep{
				step("rs_initiate", bootstrapStepSucceeded, 1),
				step("create_pmm_monitoring_user", bootstrapStepPending, 0),
			},
		}
		action := nextRunStepAction(run)
		require.NotNil(t, action)
		assert.Equal(t, "create_pmm_monitoring_user", action.name)
	})

	t.Run("gives up once a run-level step exhausts its retries", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts:    []sepBootstrapHost{succeededHost},
			RunSteps: []sepBootstrapStep{step("rs_initiate", bootstrapStepFailed, bootstrapMaxAttempts)},
		}
		assert.Nil(t, nextRunStepAction(run))
	})
}

func TestNextFinalizeAction(t *testing.T) {
	t.Run("dispatches the first pending finalize step", func(t *testing.T) {
		host := sepBootstrapHost{FinalizeSteps: []sepBootstrapStep{
			step("enable_auth", bootstrapStepPending, 0),
		}}
		action := nextFinalizeAction(host)
		require.NotNil(t, action)
		assert.Equal(t, "enable_auth", action.name)
	})

	t.Run("waits while a finalize step is running", func(t *testing.T) {
		host := sepBootstrapHost{FinalizeSteps: []sepBootstrapStep{
			step("enable_auth", bootstrapStepRunning, 1),
		}}
		assert.Nil(t, nextFinalizeAction(host))
	})

	t.Run("retries a failed finalize step under the attempt cap", func(t *testing.T) {
		host := sepBootstrapHost{FinalizeSteps: []sepBootstrapStep{
			step("enable_auth", bootstrapStepFailed, 1),
		}}
		action := nextFinalizeAction(host)
		require.NotNil(t, action)
		assert.Equal(t, "enable_auth", action.name)
	})

	t.Run("gives up once the attempt cap is reached", func(t *testing.T) {
		host := sepBootstrapHost{FinalizeSteps: []sepBootstrapStep{
			step("enable_auth", bootstrapStepFailed, bootstrapMaxAttempts),
		}}
		assert.Nil(t, nextFinalizeAction(host))
	})

	t.Run("has nothing to do once every finalize step succeeded", func(t *testing.T) {
		host := sepBootstrapHost{FinalizeSteps: []sepBootstrapStep{
			step("enable_auth", bootstrapStepSucceeded, 1),
		}}
		assert.Nil(t, nextFinalizeAction(host))
	})
}

func TestRunStepsSucceeded(t *testing.T) {
	t.Run("false while a run-level step is still pending", func(t *testing.T) {
		run := sepBootstrapRun{RunSteps: []sepBootstrapStep{
			step("rs_initiate", bootstrapStepSucceeded, 1),
			step("create_pmm_monitoring_user", bootstrapStepPending, 0),
		}}
		assert.False(t, runStepsSucceeded(run))
	})

	t.Run("true once every run-level step succeeded or was skipped", func(t *testing.T) {
		run := sepBootstrapRun{RunSteps: []sepBootstrapStep{
			step("rs_initiate", bootstrapStepSucceeded, 1),
			step("create_pmm_monitoring_user", bootstrapStepSucceeded, 1),
		}}
		assert.True(t, runStepsSucceeded(run))
	})

	t.Run("true when there are no run-level steps at all", func(t *testing.T) {
		assert.True(t, runStepsSucceeded(sepBootstrapRun{}))
	})
}

func TestRunNeedsRollback(t *testing.T) {
	t.Run("false while every step is still pending, running, or succeeded", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts: []sepBootstrapHost{{Steps: []sepBootstrapStep{step("pre_check", bootstrapStepRunning, 1)}}},
		}
		assert.False(t, runNeedsRollback(run))
	})

	t.Run("true once a host exhausts a step's retries", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts: []sepBootstrapHost{{Steps: []sepBootstrapStep{
				step("install_package", bootstrapStepFailed, bootstrapMaxAttempts),
			}}},
		}
		assert.True(t, runNeedsRollback(run))
	})

	t.Run("true once a run-level step exhausts its retries", func(t *testing.T) {
		run := sepBootstrapRun{
			RunSteps: []sepBootstrapStep{step("rs_initiate", bootstrapStepFailed, bootstrapMaxAttempts)},
		}
		assert.True(t, runNeedsRollback(run))
	})

	t.Run("false while a failed step still has a retry left", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts: []sepBootstrapHost{{Steps: []sepBootstrapStep{step("install_package", bootstrapStepFailed, 1)}}},
		}
		assert.False(t, runNeedsRollback(run))
	})

	t.Run("true once a host's finalize step exhausts its retries", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts: []sepBootstrapHost{{
				Steps:         []sepBootstrapStep{step("verify", bootstrapStepSucceeded, 1)},
				FinalizeSteps: []sepBootstrapStep{step("enable_auth", bootstrapStepFailed, bootstrapMaxAttempts)},
			}},
		}
		assert.True(t, runNeedsRollback(run))
	})
}

func TestRunIsRollingBack(t *testing.T) {
	t.Run("true once any host's rollback has started, even without a fresh failure", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts: []sepBootstrapHost{
				{
					Steps:         []sepBootstrapStep{step("install_package", bootstrapStepFailed, bootstrapMaxAttempts)},
					RollbackSteps: []sepBootstrapStep{step("stop_service", bootstrapStepRunning, 1)},
				},
				// A second, otherwise-healthy host: still counts as rolling back
				// because rollback means every host, not just the failed one.
				{
					Steps:         []sepBootstrapStep{step("verify", bootstrapStepSucceeded, 1)},
					RollbackSteps: []sepBootstrapStep{step("stop_service", bootstrapStepPending, 0)},
				},
			},
		}
		assert.True(t, runIsRollingBack(run))
	})

	t.Run("false when nothing has failed and no rollback has started", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts: []sepBootstrapHost{{
				Steps:         []sepBootstrapStep{step("pre_check", bootstrapStepPending, 0)},
				RollbackSteps: []sepBootstrapStep{step("stop_service", bootstrapStepPending, 0)},
			}},
		}
		assert.False(t, runIsRollingBack(run))
	})
}

func TestRunRollbackDone(t *testing.T) {
	t.Run("false while any host's rollback is still in flight", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts: []sepBootstrapHost{
				{RollbackSteps: []sepBootstrapStep{step("stop_service", bootstrapStepSucceeded, 1)}},
				{RollbackSteps: []sepBootstrapStep{step("stop_service", bootstrapStepRunning, 1)}},
			},
		}
		assert.False(t, runRollbackDone(run))
	})

	t.Run("true once every host's rollback steps have all succeeded or been skipped", func(t *testing.T) {
		run := sepBootstrapRun{
			Hosts: []sepBootstrapHost{
				{RollbackSteps: []sepBootstrapStep{step("stop_service", bootstrapStepSucceeded, 1)}},
				{RollbackSteps: []sepBootstrapStep{step("stop_service", bootstrapStepSkipped, 0)}},
			},
		}
		assert.True(t, runRollbackDone(run))
	})
}

func TestNextRollbackAction(t *testing.T) {
	t.Run("dispatches the first pending rollback step", func(t *testing.T) {
		host := sepBootstrapHost{RollbackSteps: []sepBootstrapStep{
			step("stop_service", bootstrapStepPending, 0),
			step("remove_config", bootstrapStepPending, 0),
		}}
		action := nextRollbackAction(host)
		require.NotNil(t, action)
		assert.Equal(t, "stop_service", action.name)
	})

	t.Run("gives up when a rollback step itself exhausts its retries", func(t *testing.T) {
		host := sepBootstrapHost{RollbackSteps: []sepBootstrapStep{
			step("stop_service", bootstrapStepFailed, bootstrapMaxAttempts),
		}}
		assert.Nil(t, nextRollbackAction(host))
	})
}
