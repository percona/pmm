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

import { useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import { alpha } from '@mui/material/styles';
import CancelIcon from '@mui/icons-material/Cancel';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import CircularProgress from '@mui/material/CircularProgress';
import {
  BOOTSTRAP_RUN_COLOR,
  BOOTSTRAP_RUN_LABEL,
  BOOTSTRAP_STEP_LABEL,
} from '../constants';
import {
  bootstrapRunDisplayStatus,
  canCancelBootstrapRun,
  isHostRollingBack,
} from '../api';
import { useCancelBootstrapRun } from '../inventoryHooks';
import type { OmBootstrapStep, OmGetBootstrapRunResponse } from '../types';

/**
 * Shared by {@link BootstrapPage}'s live "Bootstrap" step and the Automations
 * page's expanded run row - a run's progress reads the same whichever way it is
 * reached, so there is one rendering of it rather than two that can drift.
 *
 * A matrix, not a per-host list: hosts are columns, steps are rows, so a
 * reader compares hosts at a glance instead of scanning separate chip lists.
 * Pending/skipped render as a greyed-out icon rather than a text chip - the
 * point of the matrix is that a reader scans for the *one* cell that isn't
 * green or grey, not that they read every cell's label.
 */

/** One step's icon: grey for not-yet/skipped, spinner while running, green/red on a terminal outcome. */
const StepStatusIcon = ({ step }: { step: OmBootstrapStep }) => {
  const label = `${step.name}: ${BOOTSTRAP_STEP_LABEL[step.status] ?? step.status}${
    step.attempt_count > 1 ? ` (attempt ${step.attempt_count})` : ''
  }${step.detail ? ` — ${step.detail}` : ''}`;
  return (
    <Tooltip title={label}>
      <Box
        aria-label={label}
        sx={{ display: 'inline-flex', verticalAlign: 'middle' }}
      >
        {step.status === 'succeeded' ? (
          <CheckCircleIcon fontSize="small" color="success" />
        ) : step.status === 'failed' ? (
          <CancelIcon fontSize="small" color="error" />
        ) : step.status === 'running' ? (
          <CircularProgress size={16} />
        ) : step.status === 'skipped' ? (
          <RemoveCircleOutlineIcon
            fontSize="small"
            sx={{ color: 'text.disabled' }}
          />
        ) : (
          <RadioButtonUncheckedIcon
            fontSize="small"
            sx={{ color: 'text.disabled' }}
          />
        )}
      </Box>
    </Tooltip>
  );
};

/**
 * The step matrix: hosts as columns, steps as rows, in execution order - an
 * optional leading "Starting…" row (see {@link isStarting}) while nothing has
 * been dispatched yet, then every host's own forward steps, then the
 * run-level steps (one shared outcome, not per-host, repeated under every
 * column rather than merged into one spanning cell -- see that row's own
 * comment for why), then every host's finalize steps.
 *
 * Rollback steps, once any host starts rolling back, are appended below
 * those under a "Rollback" divider rather than replacing them: forward/
 * run-level/finalize rows are the record of what this run actually did
 * before it started rolling back, and a reader diagnosing a failed run needs
 * that record at least as much as the teardown that followed it.
 *
 * Row names come from `run.hosts[0]` rather than being hand-listed: every
 * host in one run shares one strategy (phase-1 scope), so every host's step
 * list is the same names in the same order.
 */
const StepMatrix = ({ run }: { run: OmGetBootstrapRunResponse }) => {
  const rollingBack = run.hosts.some(isHostRollingBack);
  const seedHost = run.hosts[0];
  if (!seedHost) {
    return null;
  }
  const columnCount = run.hosts.length + 1;

  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Step</TableCell>
            {run.hosts.map((host) => (
              <TableCell key={host.host} align="center">
                {host.host}
              </TableCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {isStarting(run) && (
            <TableRow>
              <TableCell colSpan={columnCount}>
                <Stack direction="row" spacing={1} alignItems="center">
                  <CircularProgress size={16} />
                  <Typography variant="body2" color="text.secondary">
                    Starting…
                  </Typography>
                </Stack>
              </TableCell>
            </TableRow>
          )}
          {seedHost.steps.map((forwardStep) => (
            <TableRow key={`forward-${forwardStep.name}`}>
              <TableCell>{forwardStep.name}</TableCell>
              {run.hosts.map((host) => {
                const step = host.steps.find(
                  (candidate) => candidate.name === forwardStep.name
                );
                return (
                  <TableCell key={host.host} align="center">
                    {step && <StepStatusIcon step={step} />}
                  </TableCell>
                );
              })}
            </TableRow>
          ))}
          {run.run_steps.map((step) => (
            <TableRow key={`run-${step.name}`} sx={{ bgcolor: 'action.hover' }}>
              <TableCell>{step.name}</TableCell>
              {run.hosts.map((host) => (
                // Not colSpan: a single merged cell centers within the
                // union of every host column's width, which for an odd
                // number of hosts lands the icon looking like it belongs to
                // whichever column is in the middle rather than to all of
                // them -- repeating the one shared outcome under every
                // column reads unambiguously instead.
                <TableCell key={host.host} align="center">
                  <StepStatusIcon step={step} />
                </TableCell>
              ))}
            </TableRow>
          ))}
          {seedHost.finalize_steps.map((finalizeStep) => (
            <TableRow key={`finalize-${finalizeStep.name}`}>
              <TableCell>{finalizeStep.name}</TableCell>
              {run.hosts.map((host) => {
                const step = host.finalize_steps.find(
                  (candidate) => candidate.name === finalizeStep.name
                );
                return (
                  <TableCell key={host.host} align="center">
                    {step && <StepStatusIcon step={step} />}
                  </TableCell>
                );
              })}
            </TableRow>
          ))}
          {rollingBack && (
            <TableRow>
              <TableCell
                colSpan={columnCount}
                sx={{
                  bgcolor: (theme) => alpha(theme.palette.warning.main, 0.16),
                }}
              >
                <Typography
                  variant="body2"
                  component="span"
                  color="warning.main"
                  fontWeight="bold"
                >
                  Rollback
                </Typography>
              </TableCell>
            </TableRow>
          )}
          {rollingBack &&
            seedHost.rollback_steps.map((rollbackStep) => (
              <TableRow key={`rollback-${rollbackStep.name}`}>
                <TableCell>{rollbackStep.name}</TableCell>
                {run.hosts.map((host) => {
                  const step = host.rollback_steps.find(
                    (candidate) => candidate.name === rollbackStep.name
                  );
                  return (
                    <TableCell key={host.host} align="center">
                      {step && <StepStatusIcon step={step} />}
                    </TableCell>
                  );
                })}
              </TableRow>
            ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

/**
 * True the instant a run exists but the stepper hasn't dispatched anything
 * yet - every host's own steps are still `pending`. A freshly-created run has
 * no `running` step to show until PMM's HA-leader-only stepper's next tick
 * (up to its own poll interval later), and until then every cell in the
 * matrix reads exactly like a run that is stuck, not one that just started.
 */
function isStarting(run: OmGetBootstrapRunResponse): boolean {
  return (
    bootstrapRunDisplayStatus(run) === 'running' &&
    run.hosts.length > 0 &&
    run.hosts.every((host) =>
      host.steps.every((step) => step.status === 'pending')
    )
  );
}

/**
 * Ask for confirmation, then request cancellation of `run`.
 *
 * A separate confirm step because this is destructive in the same sense
 * HostsPage's Forget dialog is: an operator's second thought should be caught
 * before the request goes out, not after. Renders nothing once the run is no
 * longer cancellable (`canCancelBootstrapRun`) - once cancellation has already
 * been requested, {@link RunProgress} shows that as a warning line instead;
 * once the run is terminal, there is nothing left to offer.
 */
const AbortButton = ({ run }: { run: OmGetBootstrapRunResponse }) => {
  const [confirming, setConfirming] = useState(false);
  const cancelRun = useCancelBootstrapRun();
  if (!canCancelBootstrapRun(run)) {
    return null;
  }
  return (
    <>
      <Button
        color="error"
        size="small"
        variant="outlined"
        onClick={() => setConfirming(true)}
      >
        Abort
      </Button>
      <Dialog
        open={confirming}
        onClose={() => setConfirming(false)}
        maxWidth="sm"
      >
        <DialogTitle>Abort this deployment?</DialogTitle>
        <DialogContent>
          <DialogContentText component="div">
            <p>
              This stops run {run.run_id} and rolls back every host in it -
              packages, configuration, and data directories this run has already
              written are removed. Hosts that had already finished successfully
              are rolled back too, since a partial replica set is not a state to
              leave running.
            </p>
            <p>This cannot be undone.</p>
          </DialogContentText>
          {cancelRun.isError && (
            <Alert severity="error" sx={{ mt: 1 }}>
              {cancelRun.error.message}
            </Alert>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirming(false)}>Cancel</Button>
          <Button
            color="error"
            disabled={cancelRun.isPending}
            onClick={async () => {
              await cancelRun.mutateAsync(run.run_id);
              setConfirming(false);
            }}
          >
            Abort
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

/** A run's full progress: its own status, then the step matrix. */
export const RunProgress = ({ run }: { run: OmGetBootstrapRunResponse }) => {
  const status = bootstrapRunDisplayStatus(run);
  return (
    <Stack spacing={2}>
      <Stack direction="row" spacing={2} alignItems="center">
        <Alert
          severity={
            status === 'succeeded'
              ? 'success'
              : status === 'failed' || status === 'rolled_back'
                ? 'error'
                : 'info'
          }
          sx={{ flexGrow: 1 }}
        >
          Run {run.run_id}:{' '}
          <Chip
            size="small"
            label={BOOTSTRAP_RUN_LABEL[status] ?? status}
            color={BOOTSTRAP_RUN_COLOR[status] ?? 'default'}
          />{' '}
          {run.replica_set_name}
          {(run.environment || run.cluster) &&
            ` (${[run.environment, run.cluster].filter(Boolean).join(' / ')})`}
          {run.error ? ` — ${run.error}` : ''}
        </Alert>
        <AbortButton run={run} />
      </Stack>
      {run.cancel_requested && !run.hosts.some(isHostRollingBack) && (
        <Typography variant="body2" color="warning.main">
          Abort requested - rolling back once the current step stops.
        </Typography>
      )}
      {run.hosts.some(isHostRollingBack) && (
        <Typography variant="body2" color="warning.main">
          Rolling back every host.
        </Typography>
      )}
      <StepMatrix run={run} />
    </Stack>
  );
};
