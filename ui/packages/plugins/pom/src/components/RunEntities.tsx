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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import { usePomInventoryRun } from '../inventoryHooks';
import { formatDuration } from '../format';
import { Unavailable } from './Unavailable';
import type { PomInventoryRun } from '../types';

/** How a host was matched, spelled out. The raw values are terse to the point of coy. */
const RESOLUTION_LABEL: Record<string, string> = {
  name: 'by node name',
  address: 'by node address',
  orphaned: 'no executor',
};

/**
 * A sub-second host time still deserves a number.
 *
 * `formatDuration` floors to whole seconds, which is right for an oplog window of days
 * and wrong here: a host that failed in 0.28s would read as `0s`, hiding that it
 * failed instantly rather than after a wait.
 */
function formatHostDuration(seconds: number): string {
  if (seconds < 10) {
    return `${seconds.toFixed(2)}s`;
  }
  return formatDuration(seconds) || `${Math.round(seconds)}s`;
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
export function RunEntities({ run }: { run: PomInventoryRun }) {
  const { data, isPending, isError, error } = usePomInventoryRun(run.run_id);

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
            <TableCell>Service</TableCell>
            <TableCell>Executor host</TableCell>
            <TableCell>Matched</TableCell>
            <TableCell>Answered</TableCell>
            <TableCell>Host time</TableCell>
            <TableCell>Error</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {entities.map((entity, index) => (
            <TableRow
              key={`${entity.service_id ?? entity.service_name}-${index}`}
            >
              <TableCell>{entity.service_name}</TableCell>
              <TableCell>
                {entity.executor_host ?? (
                  <Unavailable reason="not_applicable" />
                )}
              </TableCell>
              <TableCell>
                <Tooltip
                  title={
                    entity.resolution === 'orphaned'
                      ? 'No executor host matched this service, so nothing was dispatched for it. Not an error.'
                      : 'How the service was matched to the host its probe ran on.'
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
                  <Tooltip title="The host's wall clock, dispatch to collected output. Shared by every service on that host, because one dispatch covers them all.">
                    <Box component="span">
                      {formatHostDuration(entity.duration_seconds)}
                    </Box>
                  </Tooltip>
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
