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

import { Fragment, useEffect, useMemo, useState, type ReactNode } from 'react';
import { Link as RouterLink } from 'react-router-dom';
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
  IconButton,
  Link,
  Paper,
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
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import CircularProgress from '@mui/material/CircularProgress';
import {
  BOOTSTRAP_RUN_COLOR,
  BOOTSTRAP_RUN_LABEL,
  BOOTSTRAP_STEP_LABEL,
  OM_ROUTE_FLEET,
  OM_ROUTE_NODES,
  bootstrapStepLabel,
} from '../constants';
import {
  bootstrapRunDisplayStatus,
  canCancelBootstrapRun,
  isHostRollingBack,
} from '../api';
import { formatRunElapsed } from '../format';
import { useCancelBootstrapRun, useOmInventoryHosts } from '../inventoryHooks';
import { runHasFailure, runSummaryLine } from '../runSummary';
import type { OmBootstrapStep, OmGetBootstrapRunResponse } from '../types';
import { useOmBase } from '../useOmBase';
import { useNow } from '../useNow';
import { SecurityPosture } from './SecurityPosture';

/**
 * A run's progress, as the Automations page's expanded run row shows it.
 *
 * Leads with one line rather than the matrix: which step of how many, what it is
 * doing and for how long. The matrix is one click away, and opens by itself the
 * moment anything fails, because that is when a reader needs the cell.
 *
 * A matrix, not a per-host list: hosts are columns, steps are rows, so a
 * reader compares hosts at a glance instead of scanning separate chip lists.
 * Pending/skipped render as a greyed-out icon rather than a text chip - the
 * point of the matrix is that a reader scans for the *one* cell that isn't
 * green or grey, not that they read every cell's label.
 */

/** One step's icon: grey for not-yet/skipped, spinner while running, green/red on a terminal outcome. */
const StepStatusIcon = ({ step }: { step: OmBootstrapStep }) => {
  const label = `${bootstrapStepLabel(step.name)}: ${BOOTSTRAP_STEP_LABEL[step.status] ?? step.status}${
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

/** A run's host key to the name a reader knows the node by. */
type NodeNameOf = (host: string) => string;

/**
 * Resolve a run's host keys to node names.
 *
 * A run names its hosts by executor hostname (`BootstrapHost.host` - pmm-managed
 * hands PMM Extensions executor hosts, and joins them back the same way for
 * confirm_monitoring), which is not what anyone calls a node. The names come from
 * the same nodes list the Nodes page reads, so this is a cache hit there. Node ids
 * resolve too, in case the key is ever the id the install was requested with; a key
 * the list does not know - a node forgotten since the run - shows as itself.
 */
function useNodeNames(): NodeNameOf {
  const { data: hosts } = useOmInventoryHosts();
  return useMemo(() => {
    const names = new Map<string, string>();
    for (const host of hosts ?? []) {
      names.set(host.node_id, host.name);
      if (host.executor_host) {
        names.set(host.executor_host, host.name);
      }
    }
    return (key: string) => names.get(key) || key;
  }, [hosts]);
}

/**
 * One cell of the matrix. A failed one is tinted as well as iconed, so it is the
 * first thing a reader of a failed run finds when the matrix opens by itself.
 */
const StepCell = ({ step }: { step: OmBootstrapStep | undefined }) => (
  <TableCell
    align="center"
    sx={
      step?.status === 'failed'
        ? { bgcolor: (theme) => alpha(theme.palette.error.main, 0.12) }
        : undefined
    }
  >
    {step && <StepStatusIcon step={step} />}
  </TableCell>
);

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
const StepMatrix = ({
  run,
  nodeName,
}: {
  run: OmGetBootstrapRunResponse;
  nodeName: NodeNameOf;
}) => {
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
                {nodeName(host.host)}
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
              <TableCell>{bootstrapStepLabel(forwardStep.name)}</TableCell>
              {run.hosts.map((host) => {
                const step = host.steps.find(
                  (candidate) => candidate.name === forwardStep.name
                );
                return <StepCell key={host.host} step={step} />;
              })}
            </TableRow>
          ))}
          {run.run_steps.map((step) => (
            <TableRow key={`run-${step.name}`} sx={{ bgcolor: 'action.hover' }}>
              <TableCell>{bootstrapStepLabel(step.name)}</TableCell>
              {run.hosts.map((host) => (
                // Not colSpan: a single merged cell centers within the
                // union of every host column's width, which for an odd
                // number of hosts lands the icon looking like it belongs to
                // whichever column is in the middle rather than to all of
                // them -- repeating the one shared outcome under every
                // column reads unambiguously instead.
                <StepCell key={host.host} step={step} />
              ))}
            </TableRow>
          ))}
          {seedHost.finalize_steps.map((finalizeStep) => (
            <TableRow key={`finalize-${finalizeStep.name}`}>
              <TableCell>{bootstrapStepLabel(finalizeStep.name)}</TableCell>
              {run.hosts.map((host) => {
                const step = host.finalize_steps.find(
                  (candidate) => candidate.name === finalizeStep.name
                );
                return <StepCell key={host.host} step={step} />;
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
                <TableCell>{bootstrapStepLabel(rollbackStep.name)}</TableCell>
                {run.hosts.map((host) => {
                  const step = host.rollback_steps.find(
                    (candidate) => candidate.name === rollbackStep.name
                  );
                  return <StepCell key={host.host} step={step} />;
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
 * NodesPage's Forget dialog is: an operator's second thought should be caught
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
              This stops run {run.run_id} and rolls back every node in it -
              packages, configuration, and data directories this run has already
              written are removed. Nodes that had already finished successfully
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

/** The run id, secondary to the replica set it installs, and copyable for a ticket. */
const RunId = ({ runId }: { runId: string }) => {
  const [copied, setCopied] = useState(false);
  return (
    <Stack direction="row" spacing={0.5} alignItems="center">
      <Typography variant="caption" color="text.secondary">
        Run {runId}
      </Typography>
      <Tooltip title={copied ? 'Copied' : 'Copy run ID'}>
        <IconButton
          size="small"
          aria-label="Copy run ID"
          onClick={async () => {
            try {
              await navigator.clipboard.writeText(runId);
              setCopied(true);
            } catch {
              setCopied(false);
            }
          }}
          onMouseLeave={() => setCopied(false)}
        >
          <ContentCopyIcon sx={{ fontSize: 14 }} />
        </IconButton>
      </Tooltip>
    </Stack>
  );
};

/** One labelled fact on the completion card. */
const Fact = ({ label, children }: { label: string; children: ReactNode }) => (
  <Stack direction="row" spacing={1}>
    <Typography
      variant="body2"
      color="text.secondary"
      sx={{ minWidth: 140, flexShrink: 0 }}
    >
      {label}
    </Typography>
    <Typography variant="body2" component="div">
      {children}
    </Typography>
  </Stack>
);

/**
 * How a successful install ends: what was created, where it is, and the security
 * settings it was installed with - the wording the install wizard used, so what the
 * user agreed to is what they are told they got.
 */
const CompletionCard = ({
  run,
  nodeName,
}: {
  run: OmGetBootstrapRunResponse;
  nodeName: NodeNameOf;
}) => {
  const omBase = useOmBase();
  const took = formatRunElapsed(run.started_at, run.finished_at);
  return (
    <Paper variant="outlined" sx={{ p: 2 }} aria-label="Install summary">
      <Stack spacing={2}>
        <Stack direction="row" spacing={1} alignItems="center">
          <CheckCircleIcon color="success" />
          <Typography variant="subtitle1" component="h3">
            Replica set {run.replica_set_name} is installed
          </Typography>
        </Stack>
        <Stack spacing={0.5}>
          <Fact label="MongoDB version">{run.mongodb_version || '—'}</Fact>
          <Fact label="Members">
            {run.hosts.map((host, index) => {
              const name = nodeName(host.host);
              return (
                <Fragment key={host.host}>
                  {index > 0 && ', '}
                  <Link
                    component={RouterLink}
                    to={`${omBase}/${OM_ROUTE_NODES}?node=${encodeURIComponent(name)}`}
                  >
                    {name}
                  </Link>
                </Fragment>
              );
            })}
          </Fact>
          {run.environment && (
            <Fact label="Environment">{run.environment}</Fact>
          )}
          {run.cluster && <Fact label="Cluster">{run.cluster}</Fact>}
          {took && <Fact label="Took">{took}</Fact>}
        </Stack>
        <SecurityPosture title="Security settings still in place" />
        <Stack direction="row" spacing={1}>
          <Button
            component={RouterLink}
            to={`${omBase}/${OM_ROUTE_FLEET}?tab=clusters`}
            variant="outlined"
            size="small"
          >
            View in Fleet
          </Button>
          <Button
            component={RouterLink}
            to={`${omBase}/${OM_ROUTE_NODES}`}
            variant="outlined"
            size="small"
          >
            View nodes
          </Button>
        </Stack>
      </Stack>
    </Paper>
  );
};

/**
 * A run's full progress: the replica set and its status, one line saying where the
 * run has got to, the completion card once it succeeds, and the step matrix behind a
 * toggle.
 */
export const RunProgress = ({ run }: { run: OmGetBootstrapRunResponse }) => {
  const status = bootstrapRunDisplayStatus(run);
  const nodeName = useNodeNames();
  const failed = runHasFailure(run);
  const [showDetails, setShowDetails] = useState(failed);
  // Opens on the transition too, not only on mount: a reader watching a run live
  // should see the failing cell the moment it fails, without looking for a toggle.
  useEffect(() => {
    if (failed) {
      setShowDetails(true);
    }
  }, [failed]);
  const now = useNow(status === 'running');
  const summary = runSummaryLine(run, nodeName, now);
  const rollingBack = run.hosts.some(isHostRollingBack);
  const labels = [run.environment, run.cluster].filter(Boolean).join(' / ');
  const detailsId = `run-${run.run_id}-steps`;

  return (
    <Stack spacing={2}>
      <Stack direction="row" spacing={2} alignItems="flex-start">
        <Box sx={{ flexGrow: 1 }}>
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="subtitle1" component="h3" fontWeight="bold">
              {run.replica_set_name}
            </Typography>
            <Chip
              size="small"
              label={BOOTSTRAP_RUN_LABEL[status] ?? status}
              color={BOOTSTRAP_RUN_COLOR[status] ?? 'default'}
            />
            {labels && (
              <Typography variant="body2" color="text.secondary">
                {labels}
              </Typography>
            )}
          </Stack>
          <RunId runId={run.run_id} />
        </Box>
        <AbortButton run={run} />
      </Stack>
      {summary && (
        <Typography
          variant="body1"
          role="status"
          color={failed ? 'error.main' : undefined}
        >
          {summary}
        </Typography>
      )}
      {run.error && <Alert severity="error">{run.error}</Alert>}
      {run.cancel_requested && !rollingBack && (
        <Typography variant="body2" color="warning.main">
          Abort requested - rolling back once the current step stops.
        </Typography>
      )}
      {rollingBack && (
        <Typography variant="body2" color="warning.main">
          Rolling back every node.
        </Typography>
      )}
      {status === 'succeeded' && (
        <CompletionCard run={run} nodeName={nodeName} />
      )}
      <Box>
        <Button
          size="small"
          aria-expanded={showDetails}
          aria-controls={detailsId}
          onClick={() => setShowDetails((open) => !open)}
        >
          {showDetails ? 'Hide step details' : 'Show step details'}
        </Button>
        {showDetails && (
          <Box id={detailsId} sx={{ mt: 1 }}>
            <StepMatrix run={run} nodeName={nodeName} />
          </Box>
        )}
      </Box>
    </Stack>
  );
};
