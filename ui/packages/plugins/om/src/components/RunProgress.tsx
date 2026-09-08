/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { Alert, Box, Chip, Stack, Typography } from '@mui/material';
import {
  BOOTSTRAP_RUN_COLOR,
  BOOTSTRAP_RUN_LABEL,
  BOOTSTRAP_STEP_COLOR,
  BOOTSTRAP_STEP_LABEL,
} from '../constants';
import { isHostRollingBack } from '../api';
import type {
  OmBootstrapHost,
  OmBootstrapStep,
  OmGetBootstrapRunResponse,
} from '../types';

/**
 * Shared by {@link BootstrapWizard}'s live "Bootstrap" step and the Operations
 * page's expanded run row - a run's progress reads the same whichever way it is
 * reached, so there is one rendering of it rather than two that can drift.
 */

const StepChip = ({ step }: { step: OmBootstrapStep }) => (
  <Chip
    size="small"
    label={`${step.name}: ${BOOTSTRAP_STEP_LABEL[step.status] ?? step.status}${
      step.attempt_count > 1 ? ` (attempt ${step.attempt_count})` : ''
    }`}
    color={BOOTSTRAP_STEP_COLOR[step.status] ?? 'default'}
    variant={step.status === 'running' ? 'outlined' : 'filled'}
    title={step.detail ?? undefined}
  />
);

/**
 * One host's progress: its own steps then its finalize steps ordinarily, or its
 * rollback steps once the run has actually started tearing it down - never both
 * forward and rollback at once, since a host being rolled back has nothing left
 * to show from its forward attempt that the rollback list doesn't already
 * explain. Finalize steps stay alongside the host's own steps rather than
 * rolling back with them: they only ever run once every host's own steps and
 * the run's own steps have already succeeded, so a run that reaches rollback
 * never dispatched them in the first place.
 */
export const HostProgress = ({ host }: { host: OmBootstrapHost }) => {
  const rollingBack = isHostRollingBack(host);
  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>
        {host.host}
        {rollingBack ? ' — rolling back' : ''}
      </Typography>
      <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
        {(rollingBack
          ? host.rollback_steps
          : [...host.steps, ...host.finalize_steps]
        ).map((step) => (
          <StepChip key={step.name} step={step} />
        ))}
      </Stack>
    </Box>
  );
};

/** A run's full progress: its own status, every host, and the run-level steps. */
export const RunProgress = ({ run }: { run: OmGetBootstrapRunResponse }) => (
  <Stack spacing={2}>
    <Alert
      severity={
        run.status === 'succeeded'
          ? 'success'
          : run.status === 'failed' || run.status === 'rolled_back'
            ? 'error'
            : 'info'
      }
    >
      Run {run.run_id}:{' '}
      <Chip
        size="small"
        label={BOOTSTRAP_RUN_LABEL[run.status] ?? run.status}
        color={BOOTSTRAP_RUN_COLOR[run.status] ?? 'default'}
      />{' '}
      {run.replica_set_name}
      {(run.environment || run.cluster) &&
        ` (${[run.environment, run.cluster].filter(Boolean).join(' / ')})`}
      {run.error ? ` — ${run.error}` : ''}
    </Alert>
    {run.hosts.map((host) => (
      <HostProgress key={host.host} host={host} />
    ))}
    {run.run_steps.length > 0 && (
      <Box>
        <Typography variant="subtitle2" gutterBottom>
          Replica set
        </Typography>
        <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
          {run.run_steps.map((step) => (
            <StepChip key={step.name} step={step} />
          ))}
        </Stack>
      </Box>
    )}
  </Stack>
);
