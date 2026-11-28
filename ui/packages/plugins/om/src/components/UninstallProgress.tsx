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

import { useMemo } from 'react';
import {
  Box,
  CircularProgress,
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
import { OPERATION_STATUS_LABEL, UNINSTALL_STEP_LABEL } from '../constants';
import {
  isOperationProgressing,
  useOmOperation,
  useUninstallReport,
} from '../operationHooks';
import type { OmOperation, OmOperationStep } from '../types';
import { OmError } from './OmError';

/** An operation step's own status words, as a hover label. */
const STEP_STATUS_LABEL: Record<string, string> = {
  planned: 'Pending',
  dispatching: 'Starting',
  running: 'Running',
  succeeded: 'Succeeded',
  failed: 'Failed',
  unknown: 'Outcome unknown',
};

/** Every uninstall step a member runs, in order, so the matrix shows the whole teardown from the start. */
const UNINSTALL_STEPS = Object.keys(UNINSTALL_STEP_LABEL);

/** Whether a step is on a member host, the only kind an uninstall plans. */
const isMemberStep = (step: OmOperationStep) => step.target_kind === 'member';

/**
 * One step's icon: grey until it starts, a spinner while it runs, green or red once it settles.
 * An operation step in flight reads as `unknown` (the driver re-probes it for proof); it spins
 * while the operation is still progressing, so a running teardown shows a spinner like an install.
 */
const StepStatusIcon = ({
  step,
  active,
}: {
  step: OmOperationStep;
  active: boolean;
}) => {
  const running =
    step.status === 'running' ||
    step.status === 'dispatching' ||
    (active && step.status === 'unknown');
  const label = `${UNINSTALL_STEP_LABEL[step.step] ?? step.step}: ${
    running
      ? STEP_STATUS_LABEL.running
      : (STEP_STATUS_LABEL[step.status] ?? step.status)
  }${step.attempt > 1 ? ` (attempt ${step.attempt})` : ''}`;
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
        ) : running ? (
          <CircularProgress size={16} />
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

/** A step not yet dispatched: the grey circle of a planned step, with a Pending hover label. */
const PendingStepIcon = ({ name }: { name: string }) => {
  const label = `${UNINSTALL_STEP_LABEL[name] ?? name}: ${STEP_STATUS_LABEL.planned}`;
  return (
    <Tooltip title={label}>
      <Box
        aria-label={label}
        sx={{ display: 'inline-flex', verticalAlign: 'middle' }}
      >
        <RadioButtonUncheckedIcon
          fontSize="small"
          sx={{ color: 'text.disabled' }}
        />
      </Box>
    </Tooltip>
  );
};

/** The sentence above the matrix: where the teardown is, or how it ended. */
function summaryLine(
  operation: OmOperation,
  done: number,
  total: number
): string {
  switch (operation.status) {
    case 'succeeded':
      return 'The cluster was removed.';
    case 'queued':
    case 'running':
    case 'cancelling':
      return total > 0
        ? `${done} of ${total} steps done.`
        : `${OPERATION_STATUS_LABEL[operation.status]}.`;
    default:
      return `The uninstall is ${OPERATION_STATUS_LABEL[operation.status].toLowerCase()}.`;
  }
}

/**
 * An uninstall's progress, as the Automations page's expanded row shows it.
 *
 * The teardown's own steps per member host: hosts are columns and steps are rows, so a
 * reader compares hosts at a glance and looks for the one cell that is not green. The
 * operation is read through the same query the readiness dialog used, so it keeps moving
 * while the operation does and stops once it settles.
 */
export const UninstallProgress = ({
  operation,
}: {
  operation: OmOperation;
}) => {
  const { data: detail } = useOmOperation(operation.id);
  const { data: report } = useUninstallReport(
    operation.id,
    true,
    operation.version
  );
  // The list row is as fresh as the list's poll; the detail query polls faster.
  const live = detail?.operation ?? operation;
  const steps = useMemo(
    () => (detail?.steps ?? []).filter(isMemberStep),
    [detail?.steps]
  );
  const members = report?.members ?? [];

  const stepFor = (memberId: string, name: string) =>
    steps.find((step) => step.target_id === memberId && step.step === name);
  const done = steps.filter((step) => step.status === 'succeeded').length;
  const total = members.length * UNINSTALL_STEPS.length;
  const failed = live.status === 'failed' || live.status === 'abandoned';
  const active = isOperationProgressing(live);

  return (
    <Stack spacing={1.5} sx={{ py: 1 }}>
      <Typography variant="body2">{summaryLine(live, done, total)}</Typography>
      {failed && (
        <OmError
          placement="item"
          title="The uninstall did not finish"
          messages={live.error_code ?? 'No reason was recorded.'}
        />
      )}
      {members.length > 0 && (
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Step</TableCell>
                {members.map((member) => (
                  <TableCell key={member.member_id} align="center">
                    {member.host}
                  </TableCell>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {UNINSTALL_STEPS.map((name) => (
                <TableRow key={name}>
                  <TableCell>{UNINSTALL_STEP_LABEL[name] ?? name}</TableCell>
                  {members.map((member) => {
                    const step = stepFor(member.member_id, name);
                    return (
                      <TableCell
                        key={member.member_id}
                        align="center"
                        sx={
                          step?.status === 'failed'
                            ? {
                                bgcolor: (theme) =>
                                  alpha(theme.palette.error.main, 0.12),
                              }
                            : undefined
                        }
                      >
                        {step ? (
                          <StepStatusIcon step={step} active={active} />
                        ) : (
                          <PendingStepIcon name={name} />
                        )}
                      </TableCell>
                    );
                  })}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Stack>
  );
};
