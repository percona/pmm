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

import {
  Alert,
  Box,
  Chip,
  CircularProgress,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import { useOmInventoryRun } from '../inventoryHooks';
import { formatCompactDuration } from '../format';
import { Unavailable } from './Unavailable';
import type { OmExecutorResolution, OmInventoryRun } from '../types';

/** How a host was matched, spelled out. The raw values are terse to the point of coy. */
// Keyed by the wire enum, not by its suffix. These read `EXECUTOR_RESOLUTION_NAME` and
// friends, so the short keys this once used never matched and every row fell through to
// showing the raw enum value.
const RESOLUTION_LABEL: Record<OmExecutorResolution, string> = {
  EXECUTOR_RESOLUTION_UNSPECIFIED: 'not resolved',
  EXECUTOR_RESOLUTION_NAME: 'by node name',
  EXECUTOR_RESOLUTION_ADDRESS: 'by node address',
  EXECUTOR_RESOLUTION_ORPHANED: 'no executor',
};

/**
 * A sub-second host time still deserves a number.
 *
 * `formatCompactDuration` floors to whole seconds, which is right for an oplog window of days
 * and wrong here: a host that failed in 0.28s would read as `0s`, hiding that it
 * failed instantly rather than after a wait.
 */
function formatHostDuration(seconds: number): string {
  if (seconds < 10) {
    return `${seconds.toFixed(2)}s`;
  }
  return formatCompactDuration(seconds) || `${Math.round(seconds)}s`;
}

/**
 * What one refresh attempted, and what came of each entity.
 *
 * **Outcomes, never observations.** What the probe found lives on the estate, where it
 * is upserted and stays current; if this panel ever starts showing collected
 * attributes, the receipt has grown a second copy of the estate that goes stale on the
 * next refresh. The counters on the row are this list's summary - "5 of 14 answered"
 * cannot say which five, on which host, or which host took a minute, and each of those
 * is the first question asked of a partial refresh.
 */
export function RunEntities({ run }: { run: OmInventoryRun }) {
  const { data, isPending, isError, error } = useOmInventoryRun(run.run_id);

  if (isPending) {
    return (
      <Box sx={{ p: 2 }}>
        <CircularProgress size={20} />
      </Box>
    );
  }
  if (isError) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        Could not load this refresh: {(error as Error).message}
      </Alert>
    );
  }

  const entities = data?.entities ?? [];
  if (entities.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
        This refresh attempted nothing. Either it was scoped to hosts with no
        services on them, or enumeration found none.
      </Typography>
    );
  }

  return (
    <Box sx={{ p: 2 }}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Host</TableCell>
            <TableCell>Executor host</TableCell>
            <TableCell>Matched</TableCell>
            <TableCell>Answered</TableCell>
            <TableCell>Host time</TableCell>
            <TableCell>Probe log</TableCell>
            <TableCell>Services</TableCell>
            <TableCell>Error</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {entities.map((entity) => (
            <TableRow key={entity.node_id}>
              <TableCell>{entity.host_name ?? entity.node_id}</TableCell>
              <TableCell>
                {entity.executor_host ?? (
                  <Unavailable reason="not_applicable" />
                )}
              </TableCell>
              <TableCell>
                <Tooltip
                  title={
                    entity.resolution === 'EXECUTOR_RESOLUTION_ORPHANED'
                      ? 'No executor client matched this host, so nothing was dispatched to it. Not an error.'
                      : 'How the host was matched to the executor client its probe ran on.'
                  }
                >
                  <Box component="span">
                    {RESOLUTION_LABEL[entity.resolution] ?? entity.resolution}
                  </Box>
                </Tooltip>
              </TableCell>
              <TableCell>
                <Chip
                  size="small"
                  variant="outlined"
                  color={entity.answered ? 'success' : 'default'}
                  label={entity.answered ? 'yes' : 'no'}
                />
              </TableCell>
              <TableCell>
                {entity.duration_seconds == null ? (
                  <Unavailable reason="not_applicable" />
                ) : (
                  <Tooltip title="The host's wall clock, dispatch to collected output. One dispatch covers every service on it, so this is measured once.">
                    <Box component="span">
                      {formatHostDuration(entity.duration_seconds)}
                    </Box>
                  </Tooltip>
                )}
              </TableCell>
              <TableCell>
                {entity.task_history_id == null ? (
                  <Unavailable reason="not_applicable" />
                ) : (
                  // Text, not a link: the OM plugin only reaches SEP through
                  // pmm-managed's inventory proxy, which does not carry the task-log
                  // surface (app/sep/routes/download_files.py). A second route
                  // straight from the browser into SEP is a security decision, not
                  // one to make here -- the id is still most of the value.
                  <Tooltip title="The task history id of this attempt's own probe run. Its raw output is in SEP's task log, not reachable from this page.">
                    <Box component="span">{entity.task_history_id}</Box>
                  </Tooltip>
                )}
              </TableCell>
              <TableCell>
                {/* Empty is the answer, not a gap: a host with a PMM client and no
                    database is the case this receipt exists to be able to show. */}
                {entity.services.length === 0 ? (
                  <Tooltip title="No MongoDB service PMM knows about on this host.">
                    <Box component="span" sx={{ color: 'text.disabled' }}>
                      none
                    </Box>
                  </Tooltip>
                ) : (
                  <Stack gap={0.25}>
                    {entity.services.map((service) => (
                      <Tooltip
                        key={service.service_id ?? service.service_name}
                        title={
                          service.error ??
                          (service.answered
                            ? 'Answered this refresh.'
                            : 'Did not answer.')
                        }
                      >
                        <Box
                          component="span"
                          sx={{
                            color: service.answered
                              ? 'text.primary'
                              : 'error.main',
                            cursor: 'help',
                          }}
                        >
                          {service.service_name ?? service.service_id}
                          {service.answered ? '' : ' (no answer)'}
                        </Box>
                      </Tooltip>
                    ))}
                  </Stack>
                )}
              </TableCell>
              <TableCell>
                {entity.error ? (
                  <Typography variant="body2" color="error">
                    {entity.error}
                  </Typography>
                ) : (
                  <Unavailable reason="not_applicable" />
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <Typography variant="caption" color="text.secondary">
        What the probe found is on the Services and Hosts pages. This is only
        what this refresh attempted.
      </Typography>
    </Box>
  );
}
