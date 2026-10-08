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

import { useEffect, useMemo, useState } from 'react';
import {
  Link as RouterLink,
  useNavigate,
  useSearchParams,
} from 'react-router-dom';
import { useSnackbar } from 'notistack';
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
  LinearProgress,
  Link,
  MenuItem,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import {
  MaterialReactTable,
  useMaterialReactTable,
  type MRT_ColumnDef,
} from 'material-react-table';
import { formatTimestamp } from '@pmm-extensions/framework';
import {
  HOST_DATABASE_STATE_COLOR,
  HOST_DATABASE_STATE_LABEL,
  HOST_DATABASE_STATE_PHRASE,
  OM_ROUTE_AUTOMATIONS,
  OM_ROUTE_INSTALL,
} from './constants';
import { Age } from './components/Age';
import { EmptyState } from './components/EmptyState';
import { RowOverflowMenu } from './components/RowOverflowMenu';
import { NotOnboardedDialog } from './components/NotOnboardedDialog';
import { OmHeader } from './components/OmHeader';
import { Unavailable } from './components/Unavailable';
import { formatCompactDuration, pluralize } from './format';
import {
  ageSeconds,
  describeScanFailure,
  isFailing,
  toHostRows,
  type ScanFailure,
} from './inventory';
import {
  useForgetHost,
  useIsEstateRefreshing,
  useOmBootstrapRuns,
  useOmInventoryHosts,
  useRefreshInventory,
} from './inventoryHooks';
import { isBootstrapRunActive } from './api';
import { useScanConflict } from './ScanFeedback';
import { useOmBase } from './useOmBase';
import type { OmHostRow } from './types';

/**
 * The columns a row opens with: which node, what is on it, whether Operations can
 * work with it, and when it was last seen.
 *
 * Eleven columns plus select, expand and a three-button action column pushed the
 * actions off-screen behind horizontal scrolling -- including the red Forget, which
 * is the one a reader should never meet by accident while hunting for it (design
 * the width it gets). Everything hidden here is still a column-chooser away, and the agent
 * detail is in the row's own panel.
 */
const HIDDEN_BY_DEFAULT = {
  node_id: false,
  address: false,
  kernel: false,
  executor: false,
  repo: false,
  os: false,
  executor_host: false,
};

/**
 * The Hosts page's own filter, on top of what the estate returns.
 *
 * `all` is the default: a filter that hides rows by default reads as data loss the
 * first time it hides something a reader expected to see, which is exactly what
 * happened here in review — a host that already had a database on it vanished
 * under the old `available`-by-default and looked like a sync bug. `unmonitored`
 * and `monitored` are still one click away, for the two narrower questions
 * ("where could I install something" / "what is PMM already watching") — they
 * just no longer answer themselves on page load.
 */
type HostFilter = 'unmonitored' | 'monitored' | 'failing' | 'all';

const HOST_FILTERS: { id: HostFilter; label: string }[] = [
  { id: 'unmonitored', label: 'Not monitored' },
  { id: 'monitored', label: 'Monitored' },
  { id: 'all', label: 'All' },
];

/**
 * Whether a node's failing scans count against the fleet. Not the PMM Server's own
 * node's: Operations never acts on it, so its scans failing is not a fleet problem
 * (Pedro, 2026-10-06). Its failure is still shown on its row, without the alarm.
 */
const countsAsFailing = (row: OmHostRow) =>
  isFailing(row) && !row.is_pmm_server_node;

/**
 * `unregistered_only` counts as not monitored: a host with a mongod PMM cannot see
 * is not a place a fresh install can safely target, but it is also not one PMM is
 * monitoring — grouping it with `has_service` would hide it from both filters.
 */
function matchesHostFilter(row: OmHostRow, filter: HostFilter): boolean {
  if (filter === 'all') {
    return true;
  }
  if (filter === 'failing') {
    return countsAsFailing(row);
  }
  const monitored = row.database_state === 'has_service';
  return filter === 'monitored' ? monitored : !monitored;
}

const HostFilterChips = ({
  value,
  onChange,
}: {
  value: HostFilter;
  onChange: (next: HostFilter) => void;
}) => (
  <Stack direction="row" gap={1} flexWrap="wrap">
    {HOST_FILTERS.map((option) => (
      <Chip
        key={option.id}
        size="small"
        label={option.label}
        color="default"
        variant={value === option.id ? 'filled' : 'outlined'}
        onClick={() => onChange(option.id)}
      />
    ))}
  </Stack>
);

/**
 * Why nothing can run on a host, in the three ways it can be true.
 *
 * Kept as one cell rather than three columns because they are not independent: a host
 * that is not registered cannot be reachable, and reading three booleans to reach one
 * conclusion is work the page should do for its reader. The distinction that matters
 * is what to go and do, so that is what the cell says.
 */
const ExecutorCell = ({
  row,
  anyNodeOnboarded,
}: {
  row: OmHostRow;
  anyNodeOnboarded: boolean;
}) => {
  const { registered, reachable, driver_healthy, detail } = row.executor;
  const [helpOpen, setHelpOpen] = useState(false);
  if (!registered) {
    return (
      <>
        {/* Clickable: the other states name a machine to go and look at, this one
            has to be explained before anyone knows where to look. */}
        <Tooltip title="No automation agent is registered for this node, so nothing can be run on it. Open for what to check.">
          <Chip
            size="small"
            variant="outlined"
            label="Not onboarded"
            onClick={() => setHelpOpen(true)}
          />
        </Tooltip>
        <NotOnboardedDialog
          open={helpOpen}
          onClose={() => setHelpOpen(false)}
          nodeName={row.name}
          pmmAgentConnected={row.pmm_agent_connected}
          anyNodeOnboarded={anyNodeOnboarded}
        />
      </>
    );
  }
  if (!reachable) {
    return (
      <Tooltip title="An automation agent is registered for this node but PMM has lost contact with it. The machine is down, or the agent is stopped.">
        <Chip size="small" color="error" label="Agent down" />
      </Tooltip>
    );
  }
  if (!driver_healthy) {
    return (
      <Tooltip
        title={
          detail ??
          'The automation agent is up but cannot run jobs, so it cannot scan this node.'
        }
      >
        <Chip size="small" color="warning" label="Driver unhealthy" />
      </Tooltip>
    );
  }
  return <Chip size="small" color="success" variant="outlined" label="Ready" />;
};

/**
 * Why a host cannot be automated, for a tooltip.
 *
 * PMM Extensions supplies the reasons, but a host can read as ineligible with none given,
 * and an empty title makes MUI render no tooltip at all -- a disabled control
 * with no explanation. Shared by the Automation cell and the Bootstrap button so
 * the two cannot drift, which they had: the button showed nothing in that case.
 */
/**
 * Why nothing can be done to a node that an install is already running on.
 *
 * Shared rather than repeated: the Automation chip, the row's Install button and the
 * row's Forget all say it, and the point of Forget saying it is that it matches the
 * others. Three copies would drift the first time one is reworded.
 */
const BUSY_TITLE = 'Already part of an install in progress.';

const automationBlockedTitle = (reasons: string[]) =>
  reasons.join('; ') || 'Not eligible for automation.';

/**
 * Why the bulk Install button is disabled for this selection count.
 *
 * Two sentences, not one, because the counts it refuses fail for two unrelated
 * reasons and a single explanation would state something false (see the second
 * direction). **Two** is a MongoDB fact worth teaching: a two-member set cannot form
 * a majority when either member is lost, so it stops accepting writes on any single
 * failure. **Four or more** is perfectly ordinary in MongoDB and is refused only
 * because this preview implements one and three - our limit, not the database's, and
 * saying otherwise would teach a DBA something untrue.
 *
 * Zero gets the plain instruction: there is no rule to explain yet.
 */
const selectionCountTitle = (count: number): string => {
  if (count === 2) {
    return 'A two-member replica set cannot form a majority if either member is lost, so it would stop accepting writes on any single failure. Select one node, or three.';
  }
  if (count > 3) {
    return `This preview installs a one- or three-member replica set, and ${count} nodes are selected. MongoDB itself supports larger sets; Operations does not yet. Select one node, or three.`;
  }
  return 'Select exactly one node for a single-member replica set, or three for a three-member one.';
};

/**
 * A failing node's failure as one statement: how long, how many, and why in short.
 *
 * The row's whole account of the failure, so the page answers "why is this node
 * failing" without a hover or a trip to the scan history. "Needs attention" and the
 * detail panel reuse its short reason, so a reader who has seen it once recognises
 * it in the other two places.
 */
function failureStatement(failure: ScanFailure): string {
  const parts: string[] = [];
  if (failure.failingForSeconds != null) {
    parts.push(
      `Failing for ${formatCompactDuration(failure.failingForSeconds) || '0s'}`
    );
  } else {
    parts.push('Failing');
  }
  // "1 failed scan in a row" reads as a typo; a single failure is not yet a streak.
  parts.push(
    failure.consecutiveFailures > 1
      ? `${failure.consecutiveFailures} failed scans in a row`
      : '1 failed scan'
  );
  return `${parts.join(', ')}: ${failure.shortReason}`;
}

const AutomationCell = ({ row, busy }: { row: OmHostRow; busy: boolean }) => {
  if (busy) {
    return (
      <Tooltip title={BUSY_TITLE}>
        <Chip size="small" color="info" label="Installing" />
      </Tooltip>
    );
  }
  if (row.automation_eligible) {
    return (
      <Chip size="small" color="success" variant="outlined" label="Ready" />
    );
  }
  // Two kinds of ineligible, and conflating them was a false alarm: a healthy
  // replica-set member and the PMM Server's own node are not things to go and fix,
  // they are nodes Operations deliberately leaves alone. Only a fault gets the
  // warning colour and the word "attention".
  if (row.automation_blocked_by_design) {
    return (
      <Tooltip title={automationBlockedTitle(row.automation_blocked_reasons)}>
        <Chip size="small" variant="outlined" label="Not a target" />
      </Tooltip>
    );
  }
  // Said under the chip rather than only in its tooltip, so a column of "Needs
  // attention" can be read without hovering each one. When the node's scans are
  // failing that failure is the reason to show: PMM's install gate then reads "no scan
  // has reported this node's operating system yet", which is true and is a symptom -
  // it sent readers to the gate when the thing to fix was the scan.
  const failure = describeScanFailure(row.freshness);
  const reason = failure
    ? `Scans failing: ${failure.shortReason}`
    : automationBlockedTitle(row.automation_blocked_reasons);
  return (
    <Stack spacing={0.5} alignItems="flex-start">
      <Tooltip title={automationBlockedTitle(row.automation_blocked_reasons)}>
        <Chip size="small" color="warning" label="Needs attention" />
      </Tooltip>
      <Typography
        variant="caption"
        color="text.secondary"
        sx={{ whiteSpace: 'normal', lineHeight: 1.3 }}
      >
        {reason}
      </Typography>
    </Stack>
  );
};

/** Whether this host can fetch packages, and what stopped it when it cannot. */
const RepoCell = ({ row }: { row: OmHostRow }) => {
  if (!row.repo) {
    return <Unavailable reason="probe_never_succeeded" />;
  }
  if (row.repo.reachable) {
    return (
      <Tooltip
        title={`${row.repo.url ?? 'The repository'} answered in ${row.repo.latency_ms ?? '?'} ms${
          row.repo.proxy ? ` via ${row.repo.proxy}` : ' with no proxy'
        }.`}
      >
        <Chip
          size="small"
          color="success"
          variant="outlined"
          label="Reachable"
        />
      </Tooltip>
    );
  }
  return (
    <Tooltip
      // The proxy is named whether or not one is set: a refused connection direct and
      // one through a broken proxy are the same message and different jobs.
      title={`${row.repo.error ?? 'The repository did not answer.'} ${
        row.repo.proxy ? `Proxy: ${row.repo.proxy}.` : 'No proxy configured.'
      }`}
    >
      <Chip size="small" color="error" variant="outlined" label="Unreachable" />
    </Tooltip>
  );
};

/**
 * Is there a database on this host, in the three answers that question has.
 *
 * The page's headline, and the reason it is not a boolean. PMM's own inventory cannot
 * tell a bare pmm-client host from an arbiter - same node type, same agents, no
 * services - so "no registered service" alone would report an arbiter as an empty
 * machine and invite someone to install over a port already in use.
 */
const DatabaseCell = ({ row }: { row: OmHostRow }) => {
  const extra =
    row.database_state === 'has_service'
      ? ` (${row.service_count})`
      : row.database_state === 'unregistered_only'
        ? ` (${row.unregistered_mongods.length})`
        : '';
  return (
    <Tooltip title={HOST_DATABASE_STATE_PHRASE[row.database_state]}>
      <Chip
        size="small"
        color={HOST_DATABASE_STATE_COLOR[row.database_state]}
        variant={row.database_state === 'has_service' ? 'filled' : 'outlined'}
        label={`${HOST_DATABASE_STATE_LABEL[row.database_state]}${extra}`}
      />
    </Tooltip>
  );
};

function useColumns(
  busyExecutorHosts: Set<string>,
  // Whether *any* node has an agent decides whether "not onboarded" is about this
  // node or about the server. See NotOnboardedDialog.
  anyNodeOnboarded: boolean
): MRT_ColumnDef<OmHostRow>[] {
  return useMemo(
    () => [
      { accessorKey: 'name', header: 'Node', size: 180 },
      {
        accessorKey: 'address',
        header: 'Address',
        Cell: ({ row: { original } }) =>
          original.address ?? <Unavailable reason="not_applicable" />,
      },
      {
        id: 'database_state',
        size: 130,
        accessorFn: (row) => HOST_DATABASE_STATE_LABEL[row.database_state],
        header: 'Database',
        Cell: ({ row: { original } }) => <DatabaseCell row={original} />,
      },
      {
        id: 'executor',
        // Sorted on the conclusion rather than on a boolean, so the unusable hosts
        // sort together whichever way they are unusable.
        accessorFn: (row) =>
          !row.executor.registered
            ? 'Not onboarded'
            : !row.executor.reachable
              ? 'Agent down'
              : !row.executor.driver_healthy
                ? 'Driver unhealthy'
                : 'Ready',
        header: 'Automation agent',
        Cell: ({ row: { original } }) => (
          <ExecutorCell row={original} anyNodeOnboarded={anyNodeOnboarded} />
        ),
      },
      {
        id: 'automation_eligible',
        // Wide enough for the reason written under "Needs attention" to wrap onto two
        // or three lines rather than one word per line.
        size: 180,
        accessorFn: (row) =>
          row.executor_host && busyExecutorHosts.has(row.executor_host)
            ? 'Installing'
            : row.automation_eligible
              ? 'Ready'
              : row.automation_blocked_by_design
                ? 'Not a target'
                : 'Needs attention',
        header: 'Automation',
        Cell: ({ row: { original } }) => (
          <AutomationCell
            row={original}
            busy={Boolean(
              original.executor_host &&
              busyExecutorHosts.has(original.executor_host)
            )}
          />
        ),
      },
      {
        id: 'repo',
        accessorFn: (row) => row.repo?.latency_ms ?? Infinity,
        header: 'Repository',
        Cell: ({ row: { original } }) => <RepoCell row={original} />,
      },
      {
        accessorKey: 'os',
        header: 'OS',
        Cell: ({ row: { original } }) =>
          original.os ?? <Unavailable reason="probe_never_succeeded" />,
      },
      {
        accessorKey: 'kernel',
        header: 'Kernel',
        Cell: ({ row: { original } }) =>
          original.kernel ?? <Unavailable reason="probe_never_succeeded" />,
      },
      {
        id: 'collected',
        // Holds a failing node's whole statement, not just an age.
        size: 220,
        // Never-answered sorts last rather than first: as a timestamp string it would
        // sort beside the oldest row, which reads as "very stale" when it is "never".
        accessorFn: (row) =>
          ageSeconds(row.freshness.last_success_at) ?? Infinity,
        header: 'Collected',
        Cell: ({ row: { original } }) => {
          const collected = original.freshness.last_success_at;
          const age = ageSeconds(collected);
          // Asked first, and of `failing_since` rather than the success time. Keyed on
          // the success time, a node whose scans had never succeeded fell into the
          // "never collected" branch below and its error was nowhere on the page -
          // the node that most needed explaining was the one that got none.
          const failure = describeScanFailure(original.freshness);
          if (!failure) {
            return age == null ? (
              <Unavailable reason="probe_never_succeeded" />
            ) : (
              <Age value={collected} />
            );
          }
          return (
            <Stack spacing={0.25}>
              <Box
                component="span"
                sx={{
                  color: original.is_pmm_server_node
                    ? 'text.secondary'
                    : 'error.main',
                  whiteSpace: 'normal',
                }}
              >
                {failureStatement(failure)}
              </Box>
              <Typography variant="caption" color="text.secondary">
                {age == null ? (
                  'Never collected. Expand for the full error.'
                ) : (
                  <>
                    Last collected <Age value={collected} />. Expand for the
                    full error.
                  </>
                )}
              </Typography>
            </Stack>
          );
        },
      },
      {
        accessorKey: 'node_id',
        header: 'Node ID',
      },
      {
        accessorKey: 'executor_host',
        header: 'Agent name',
        Cell: ({ row: { original } }) =>
          original.executor_host ?? <Unavailable reason="not_applicable" />,
      },
    ],
    [busyExecutorHosts, anyNodeOnboarded]
  );
}

/**
 * What a host has on it, under its row.
 *
 * `GET /hosts` nests the services, so opening a row needs no second request - which is
 * why the API declined a `/hosts/{id}/services` collection. The unregistered mongods
 * are listed beside them because on a host with no registered service they are the
 * whole story.
 */
/**
 * A failing node's failure, in full, at the top of its detail panel.
 *
 * The row can only say it in short. This is the whole of it: the raw error untrimmed
 * and with its line breaks (a traceback is unreadable folded onto one line), what kind
 * of failure that is, and what to do about it. The hint sits below the error because
 * some of them refer to "the excerpt above". The run link is only drawn when the
 * server names the run; a link to the scan history in general would send the reader
 * hunting through it, which is the trip this panel exists to save.
 */
const ScanFailureDetail = ({
  failure,
  neutral,
}: {
  failure: ScanFailure;
  /** True for the PMM Server's own node; see countsAsFailing. */
  neutral: boolean;
}) => {
  const omBase = useOmBase();
  return (
    <Box data-testid="scan-failure">
      <Typography
        variant="subtitle2"
        gutterBottom
        color={neutral ? undefined : 'error.main'}
      >
        Scans failing: {failure.label}
      </Typography>
      <Typography variant="body2" sx={{ mb: 1 }}>
        {/* Relative for reading, absolute for matching against a log or a run. */}
        {failure.failingForSeconds != null
          ? `Failing for ${formatCompactDuration(failure.failingForSeconds) || '0s'} (since ${formatTimestamp(failure.failingSince)?.title})`
          : `Failing since ${failure.failingSince}`}
        {failure.consecutiveFailures > 1
          ? `, ${failure.consecutiveFailures} failed scans in a row.`
          : ', 1 failed scan.'}
      </Typography>
      {failure.error ? (
        <Box
          component="pre"
          data-testid="scan-error"
          sx={{
            m: 0,
            mb: 1,
            p: 1,
            fontFamily: 'monospace',
            fontSize: '0.8125rem',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            bgcolor: 'action.hover',
            borderRadius: 1,
          }}
        >
          {failure.error}
        </Box>
      ) : (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
          The scan recorded no error message.
        </Typography>
      )}
      {failure.hint && (
        <Typography variant="body2" sx={{ mb: 1 }}>
          {failure.hint}
        </Typography>
      )}
      {failure.runId && (
        <Link
          component={RouterLink}
          to={`${omBase}/${OM_ROUTE_AUTOMATIONS}?tab=scans&expand=${encodeURIComponent(
            failure.runId
          )}`}
          underline="hover"
          variant="body2"
        >
          Open the scan that failed
        </Link>
      )}
    </Box>
  );
};

const HostDetail = ({ row }: { row: OmHostRow }) => {
  const failure = describeScanFailure(row.freshness);
  return (
    <Stack spacing={2} sx={{ p: 2 }}>
      {failure && (
        <ScanFailureDetail failure={failure} neutral={row.is_pmm_server_node} />
      )}
      <Box>
        <Typography variant="subtitle2" gutterBottom>
          Services PMM monitors ({row.services.length})
        </Typography>
        {row.services.length === 0 ? (
          <Typography variant="body2" color="text.secondary">
            None. PMM has no registered MongoDB service on this node.
          </Typography>
        ) : (
          <Stack spacing={0.5}>
            {row.services.map((service) => (
              <Typography variant="body2" key={service.service_id}>
                {service.name ?? service.service_id}
                {service.port ? `:${service.port}` : ''}
                {service.installed_version
                  ? ` — ${service.installed_version}`
                  : ''}
              </Typography>
            ))}
          </Stack>
        )}
      </Box>
      {row.unregistered_mongods.length > 0 && (
        <Box>
          <Typography variant="subtitle2" gutterBottom color="warning.main">
            Running but not registered ({row.unregistered_mongods.length})
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>
            Found by a scan with no PMM service to match. An arbiter is the
            ordinary case: it holds no data, so PMM cannot authenticate against
            it to register one.
          </Typography>
          <Stack spacing={0.5}>
            {row.unregistered_mongods.map((mongod, index) => (
              <Typography variant="body2" key={`${mongod.port}-${index}`}>
                {mongod.program ?? 'mongod'}
                {mongod.port ? ` on :${mongod.port}` : ''}
                {mongod.config_path ? ` — ${mongod.config_path}` : ''}
              </Typography>
            ))}
          </Stack>
        </Box>
      )}
    </Stack>
  );
};

function forgottenMessage(rows: OmHostRow[]): string {
  return rows.length === 1
    ? `Removed ${rows[0].name} from Operations. If PMM still monitors it, it comes back on the next scan and is counted again.`
    : `Removed ${rows.length} nodes from Operations. Any that PMM still monitors come back on the next scan and are counted again.`;
}

/**
 * The dialog that has to tell the truth about what deleting achieves, for one
 * host's row or several.
 *
 * Deleting is not suppression: an entity PMM still knows about comes straight back on
 * the next sweep. A confirm reading "delete this host?" invites the reader to believe
 * otherwise and use it as a mute exactly once, so this says what it is actually for -
 * clearing a row left behind when `pmm-agent setup --force` re-registered a node under
 * a new id, which leaves the old row with nothing to refresh it.
 *
 * The row menu's removal and the bulk one share this dialog: the only real
 * difference is how many names are in the title and how many DELETE calls go out.
 * PMM Extensions has no batch-delete endpoint, so a bulk forget is N independent requests, not
 * one. They are dispatched together and awaited together; a partial failure keeps
 * the dialog open with the failures named, rather than closing over an incomplete
 * result the reader would have to notice was incomplete.
 *
 * `onClose` is "the reader backed out" -- Cancel and the dialog's own dismissal
 * (backdrop, Escape). `onForgotten` is "it worked" -- only called once every target
 * host is actually gone. They are two different callbacks because they mean two
 * different things to the caller: NodesPage clears the row-selection on the second,
 * never the first. Conflating them into one `onClose` was the bug -- backing out of
 * a destructive confirmation is not the same event as the destruction succeeding.
 */
const ForgetDialog = ({
  rows,
  onClose,
  onForgotten,
}: {
  rows: OmHostRow[];
  onClose: () => void;
  onForgotten: () => void;
}) => {
  const forget = useForgetHost();
  const [failures, setFailures] = useState<
    { nodeId: string; name: string; message: string }[]
  >([]);
  // Set while a batch is in flight. `forget.isPending` is one mutation's state, not
  // "any of N concurrent mutateAsync calls still pending" -- it can flip false as
  // soon as whichever call the shared observer last updated on settles, re-enabling
  // Forget while other DELETEs in the same batch are still out.
  const [busy, setBusy] = useState(false);
  // `rows.length === 0` below returns null rather than unmounting the component, so
  // `failures` would otherwise survive a close-and-reopen for an unrelated selection.
  // Left stale, it would filter `targets` (below) against node ids that do not exist
  // in the new `rows` at all -- an empty target list that "succeeds" without deleting
  // anything. `rows` is a fresh array reference exactly when the parent opens the
  // dialog for a new selection or closes it, never mid-retry, so resetting on it is
  // the right key.
  useEffect(() => {
    setFailures([]);
  }, [rows]);
  if (rows.length === 0) {
    return null;
  }
  // After a partial failure, retry dispatches only the rows named in `failures` --
  // the ones that already succeeded are gone from the estate, and re-sending their
  // DELETE would 404 and show up as a fresh, false failure for a host that is, in
  // fact, already forgotten.
  const failedIds = new Set(failures.map((failure) => failure.nodeId));
  const targets =
    failures.length > 0
      ? rows.filter((row) => failedIds.has(row.node_id))
      : rows;
  const onlyOneRow = rows.length === 1;
  const totalServices = rows.reduce((sum, row) => sum + row.services.length, 0);
  const handleForget = async () => {
    setBusy(true);
    setFailures([]);
    try {
      const outcomes = await Promise.allSettled(
        targets.map((row) => forget.mutateAsync(row.node_id))
      );
      const failed = outcomes.flatMap((outcome, index) =>
        outcome.status === 'rejected'
          ? [
              {
                nodeId: targets[index].node_id,
                name: targets[index].name,
                message:
                  outcome.reason instanceof Error
                    ? outcome.reason.message
                    : String(outcome.reason),
              },
            ]
          : []
      );
      if (failed.length === 0) {
        onForgotten();
      } else {
        setFailures(failed);
      }
    } finally {
      setBusy(false);
    }
  };
  return (
    <Dialog open onClose={onClose} maxWidth="sm">
      <DialogTitle>
        {onlyOneRow
          ? `Remove the entry for ${rows[0].name}?`
          : `Remove the entries for ${rows.length} nodes?`}
      </DialogTitle>
      <DialogContent>
        <DialogContentText component="div">
          <p>
            This removes {onlyOneRow ? 'this node' : 'these nodes'} from the
            Operations node list
            {totalServices > 0
              ? `, along with the ${totalServices} ${pluralize(totalServices, 'service')} that Operations recorded on ${onlyOneRow ? 'it' : 'them'}`
              : ''}
            .{' '}
            <strong>
              {onlyOneRow ? 'Its' : 'Their'} scan history is deleted
              permanently.
            </strong>
          </p>
          <p>
            Nothing changes on the {onlyOneRow ? 'machine' : 'machines'}, and
            PMM keeps monitoring {onlyOneRow ? 'it' : 'them'}. If PMM still has{' '}
            {onlyOneRow ? 'the node' : 'a node'}, it comes back on the next
            scan, with no scan history.
          </p>
          <p>
            Use this to clear a duplicate entry left behind when a node was
            re-registered in PMM under a new ID.
          </p>
        </DialogContentText>
        {failures.map((failure) => (
          <Alert severity="error" key={failure.nodeId} sx={{ mt: 1 }}>
            {failure.name}: {failure.message}
          </Alert>
        ))}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" disabled={busy} onClick={handleForget}>
          {onlyOneRow ? 'Remove entry' : 'Remove entries'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

/**
 * One row per host OM keeps, whether or not a database runs on it.
 *
 * The page §4 exists for. A host with no database is not an edge case here: it is
 * where a database can be installed, and it has no service to be discovered through,
 * which is why the estate keys hosts separately at all.
 */
export const NodesPage = () => {
  const { data, isPending, isError, error } = useOmInventoryHosts();
  const refresh = useRefreshInventory();
  // Any active run, matching AutomationsScansTab's button: firing an estate-wide sweep into
  // one already in flight only earns a 409, and the row actions below cannot succeed
  // against a host that sweep already holds. The refetch when a sweep lands is the
  // estate query's own business now, so this page no longer arranges it.
  const refreshing = useIsEstateRefreshing();
  const { conflict: scanConflict, runningScan } = useScanConflict(refresh);
  const navigate = useNavigate();
  const omBase = useOmBase();
  const [forgetting, setForgetting] = useState<OmHostRow[]>([]);
  const { enqueueSnackbar } = useSnackbar();
  const [hostFilter, setHostFilter] = useState<HostFilter>('all');
  // Keyed by node_id (this table's getRowId), independent of which filter is
  // active — switching filters does not silently drop a selection made under a
  // different one.
  const [rowSelection, setRowSelection] = useState<Record<string, boolean>>({});
  const bootstrapRuns = useOmBootstrapRuns();
  // Keyed by executor host, not node id: a bootstrap run's own `hosts` field
  // is the Nomad executor hostname (TriggerHostBootstrap's own doc comment on
  // why), which is what `OmHostRow.executor_host` carries too. PMM's own
  // inventory has no notion of an in-flight om_bootstrap run at all -- the
  // two apps don't share state -- so this is computed here by joining the
  // two queries rather than read off either row directly.
  const busyExecutorHosts = useMemo(() => {
    const busy = new Set<string>();
    for (const run of bootstrapRuns.data ?? []) {
      if (!isBootstrapRunActive(run)) {
        continue;
      }
      for (const host of run.hosts) {
        busy.add(host.host);
      }
    }
    return busy;
  }, [bootstrapRuns.data]);
  const isHostBusy = (row: OmHostRow) =>
    Boolean(row.executor_host && busyExecutorHosts.has(row.executor_host));
  const rows = useMemo(() => toHostRows(data), [data]);
  // Whole fleet, not the filtered view: filtering to the broken nodes must not turn
  // "only this one is broken" into "none of them work".
  const anyNodeOnboarded = useMemo(
    () => rows.some((row) => row.executor.registered),
    [rows]
  );
  const columns = useColumns(busyExecutorHosts, anyNodeOnboarded);
  // Filtered for the table only — the counts below stay whole-estate so switching
  // filters does not make the headline numbers look like they changed too.
  const filteredRows = useMemo(
    () => rows.filter((row) => matchesHostFilter(row, hostFilter)),
    [rows, hostFilter]
  );
  // Only the selected rows that are currently visible: a selection made under one
  // filter should not let a bulk action reach into rows the reader cannot see.
  const selectedRows = useMemo(
    () => filteredRows.filter((row) => rowSelection[row.node_id]),
    [filteredRows, rowSelection]
  );

  const counts = useMemo(
    () => ({
      total: rows.length,
      installable: rows.filter((row) => row.database_state === 'installable')
        .length,
      unregistered: rows.filter(
        (row) => row.database_state === 'unregistered_only'
      ).length,
      unusable: rows.filter(
        (row) =>
          !row.executor.registered ||
          !row.executor.reachable ||
          !row.executor.driver_healthy
      ).length,
      failing: rows.filter(countsAsFailing).length,
      automationEligible: rows.filter((row) => row.automation_eligible).length,
    }),
    [rows]
  );

  // The node an error elsewhere is about. A blocked node's reason has to be
  // followable to the scan that produced it, and that scan is on this page -- so the
  // destination is a row here, not a new view.
  const [searchParams] = useSearchParams();
  const focusNode = searchParams.get('node') ?? '';

  const table = useMaterialReactTable({
    columns,
    data: filteredRows,
    // The server-issued id, not MRT's default row index. These rows are refetched on a
    // timer and again whenever a refresh lands, so an insertion or a reorder would move
    // an open detail panel onto a different host. The run and cluster tables already
    // key on their own ids for the same reason.
    getRowId: (row) => row.node_id,
    enablePagination: false,
    enableDensityToggle: false,
    enableExpanding: true,
    enableRowActions: true,
    // A host already part of an in-flight bootstrap run cannot be selected
    // for another one -- see `busyExecutorHosts`'s own comment.
    enableRowSelection: (row) => !isHostBusy(row.original),
    // See FleetClustersTab: without these MRT sizes every column to its header's
    // chrome rather than its content, and the table overflows the ~980px the page
    // gets at 1440 with the nav open. The per-column menu's only verb beyond sorting
    // is "hide this column", which the chooser in the toolbar already does.
    layoutMode: 'grid',
    enableColumnActions: false,
    // The actions column has to hold "Scan" beside "Install MongoDB", and MRT's
    // default for it is narrower than that - so the install action was clipped at
    // the right edge, which is the same width problem all over again.
    displayColumnDefOptions: {
      'mrt-row-actions': { size: 290, grow: false },
      'mrt-row-select': { size: 50, grow: false },
      'mrt-row-expand': { size: 50, grow: false },
    },
    // Ours already shows the count and carries the actions, so MRT's banner was a
    // second bar saying the same thing.
    positionToolbarAlertBanner: 'none',
    positionActionsColumn: 'last',
    onRowSelectionChange: setRowSelection,
    state: { rowSelection },
    renderDetailPanel: ({ row }) => <HostDetail row={row.original} />,
    renderRowActions: ({ row }) => (
      <Stack direction="row" spacing={1}>
        {/* The span is load-bearing: MUI disables pointer events on a disabled
            ButtonBase, so a Tooltip wrapping the button directly never opens while
            a refresh is pending -- which is exactly when a reader wants to know why. */}
        <Tooltip title="Scan this node now. Starts a job and takes tens of seconds; the row updates when it lands.">
          <Box component="span">
            <Button
              size="small"
              disabled={refresh.isPending || refreshing}
              onClick={() => refresh.refreshHosts([row.original.node_id])}
            >
              Scan
            </Button>
          </Box>
        </Tooltip>
        <Tooltip
          title={
            isHostBusy(row.original)
              ? BUSY_TITLE
              : row.original.automation_eligible
                ? 'Install MongoDB on this node and initialize a single-member replica set.'
                : automationBlockedTitle(
                    row.original.automation_blocked_reasons
                  )
          }
        >
          <Box component="span">
            <Button
              size="small"
              disabled={
                !row.original.automation_eligible || isHostBusy(row.original)
              }
              onClick={() =>
                navigate(
                  `${omBase}/${OM_ROUTE_INSTALL}?nodes=${row.original.node_id}`
                )
              }
            >
              Install MongoDB
            </Button>
          </Box>
        </Tooltip>
        {/* Behind the ellipsis, not beside the daily actions: three text buttons
            did not fit the row, and Forget was the one falling off the right edge
            and it belongs behind a menu on its own account too, named for what
            it is for rather than for what it deletes. */}
        {/* No actions at all on PMM Server's own node. Forget would clear Operations'
            record of the machine PMM runs on, the next scan would put it straight
            back, and in between the fleet would be wrong - so the menu has nothing
            to show and is not rendered. */}
        {!row.original.is_pmm_server_node && (
          <RowOverflowMenu label={`More actions for ${row.original.name}`}>
            {(close) => [
              /* Wrapped in a span because a disabled MUI MenuItem fires no pointer
                 events, so the Tooltip would never open on the one state it exists
                 to explain - the same reason the Install button above has one. An
                 empty title renders no tooltip, so an idle node is unaffected. */
              <Tooltip
                key="forget"
                title={isHostBusy(row.original) ? BUSY_TITLE : ''}
              >
                <Box component="span">
                  <MenuItem
                    disabled={isHostBusy(row.original)}
                    onClick={() => {
                      setForgetting([row.original]);
                      close();
                    }}
                  >
                    Remove duplicate entry
                  </MenuItem>
                </Box>
              </Tooltip>,
            ]}
          </RowOverflowMenu>
        )}
      </Stack>
    ),
    initialState: {
      density: 'compact',
      columnVisibility: HIDDEN_BY_DEFAULT,
      sorting: [{ id: 'name', desc: false }],
      // Seeded from ?node=, so an error elsewhere can link to the one node it is
      // about and land on that row rather than on a fleet the reader has to search.
      // `initialState`, not `state`: it is a starting point, and clearing the search
      // box has to work.
      showGlobalFilter: focusNode !== '',
      globalFilter: focusNode,
    },
  });

  if (isPending) {
    return <LinearProgress />;
  }

  if (isError) {
    return (
      <Alert severity="error">
        {/* An error here means PMM Extensions is unwell, and it renders inside the page rather
            than replacing it. That is the whole point of reaching the fleet through
            pmm-managed: before the proxy, a sick PMM Extensions blanked the page entirely. */}
        {(error as Error)?.message ?? 'Could not load the nodes.'}
      </Alert>
    );
  }

  return (
    <Box>
      <OmHeader
        title="Nodes"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            Every node Operations knows about, including the ones with no
            database on them. This page says whether each node is ready to be
            scanned and installed on, not how its databases are doing: database
            health is on{' '}
            <Link component={RouterLink} to={omBase} underline="hover">
              Clusters
            </Link>
            .
          </Typography>
        }
        actions={
          <Stack direction="row" alignItems="center" gap={2}>
            <HostFilterChips value={hostFilter} onChange={setHostFilter} />
            <Tooltip title="Scan every node. Starts one job per node and takes tens of seconds.">
              <Box component="span">
                <Button
                  variant="contained"
                  startIcon={<PlayArrowIcon />}
                  disabled={refresh.isPending || refreshing}
                  onClick={() => refresh.refreshAll()}
                >
                  {refreshing ? 'Scanning…' : 'Scan all'}
                </Button>
              </Box>
            </Tooltip>
          </Stack>
        }
      />
      {/* A 409 is an expected answer, not a fault: another scan already holds these
          nodes, and the schedule starts one every ten minutes. */}
      {scanConflict && (
        <Alert
          severity="info"
          sx={{ mb: 2 }}
          action={
            runningScan && (
              <Button
                component={RouterLink}
                to={`${omBase}/${OM_ROUTE_AUTOMATIONS}?tab=scans&expand=${encodeURIComponent(
                  runningScan.run_id
                )}`}
                color="inherit"
                size="small"
              >
                Open the running scan
              </Button>
            )
          }
        >
          {scanConflict.message}
        </Alert>
      )}
      {refresh.isError && !scanConflict && (
        <Alert severity="error" sx={{ mb: 2 }}>
          Could not start a scan: {refresh.error.message}
        </Alert>
      )}
      <Stack direction="row" spacing={3} sx={{ mb: 2, alignItems: 'center' }}>
        <Typography variant="body2">
          <strong>{counts.total}</strong> {pluralize(counts.total, 'node')}
        </Typography>
        <Tooltip title="PMM Client is connected, and the node's automation agent is reachable and healthy.">
          <Typography variant="body2" sx={{ cursor: 'help' }}>
            <strong>{counts.automationEligible}</strong> eligible for automation
          </Typography>
        </Tooltip>
        <Tooltip title="No registered service and no mongod found - a node a database could be installed on.">
          <Typography variant="body2" sx={{ cursor: 'help' }}>
            <strong>{counts.installable}</strong> with no database
          </Typography>
        </Tooltip>
        {counts.unregistered > 0 && (
          <Tooltip title="A mongod is running that PMM has no service for. Not an empty node.">
            <Typography
              variant="body2"
              color="warning.main"
              sx={{ cursor: 'help' }}
            >
              <strong>{counts.unregistered}</strong> unregistered
            </Typography>
          </Tooltip>
        )}
        {counts.unusable > 0 && (
          <Typography variant="body2" color="error.main">
            <strong>{counts.unusable}</strong> cannot be scanned
          </Typography>
        )}
        {/* A count to act on, so it is the filter too - as on the Services tab. */}
        {(counts.failing > 0 || hostFilter === 'failing') && (
          <Chip
            size="small"
            color={hostFilter === 'failing' ? 'error' : 'default'}
            variant={hostFilter === 'failing' ? 'filled' : 'outlined'}
            label={`${counts.failing} failing`}
            onClick={() =>
              setHostFilter((current) =>
                current === 'failing' ? 'all' : 'failing'
              )
            }
          />
        )}
      </Stack>
      {selectedRows.length > 0 && (
        <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 2 }}>
          <Typography variant="body2">
            <strong>{selectedRows.length}</strong> selected
          </Typography>
          <Tooltip title="Scan every selected node. Starts one job per node and takes tens of seconds.">
            <Box component="span">
              <Button
                size="small"
                variant="outlined"
                disabled={refresh.isPending || refreshing}
                onClick={() => {
                  refresh.refreshHosts(selectedRows.map((row) => row.node_id));
                  setRowSelection({});
                }}
              >
                Scan selected
              </Button>
            </Box>
          </Tooltip>
          <Tooltip
            title={
              selectedRows.length !== 1 && selectedRows.length !== 3
                ? selectionCountTitle(selectedRows.length)
                : selectedRows.some((row) => isHostBusy(row))
                  ? 'A selected node is already part of an install in progress.'
                  : selectedRows.some((row) => !row.automation_eligible)
                    ? 'Every selected node must be eligible for automation.'
                    : 'Install MongoDB on the selected nodes and initialize them as one replica set.'
            }
          >
            <Box component="span">
              <Button
                size="small"
                variant="outlined"
                disabled={
                  (selectedRows.length !== 1 && selectedRows.length !== 3) ||
                  selectedRows.some((row) => !row.automation_eligible) ||
                  selectedRows.some((row) => isHostBusy(row))
                }
                onClick={() =>
                  navigate(
                    `${omBase}/${OM_ROUTE_INSTALL}?nodes=${selectedRows
                      .map((row) => row.node_id)
                      .join(',')}`
                  )
                }
              >
                Install MongoDB
              </Button>
            </Box>
          </Tooltip>
          <Button size="small" onClick={() => setForgetting(selectedRows)}>
            Remove duplicate entries
          </Button>
        </Stack>
      )}
      {filteredRows.length === 0 ? (
        <EmptyState
          title="No nodes to show"
          action={
            rows.length === 0
              ? {
                  label: 'Go to Scans',
                  to: `${omBase}/${OM_ROUTE_AUTOMATIONS}?tab=scans`,
                }
              : undefined
          }
        >
          {rows.length === 0
            ? 'This page lists every node Operations has scanned, including the ones with no database on them - which is where an install can go. There are no scan results yet: press Scan all, or check Scans if one is already running.'
            : 'Every node is filtered out by the filter above. Clear it to see them.'}
        </EmptyState>
      ) : (
        <MaterialReactTable table={table} />
      )}
      <ForgetDialog
        rows={forgetting}
        onClose={() => setForgetting([])}
        onForgotten={() => {
          enqueueSnackbar(forgottenMessage(forgetting), { variant: 'success' });
          setForgetting([]);
          setRowSelection({});
        }}
      />
    </Box>
  );
};
