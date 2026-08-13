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

import { useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Chip,
  CircularProgress,
  Collapse,
  IconButton,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowRightIcon from '@mui/icons-material/KeyboardArrowRight';
import { isProbeRunActive, usePomProbeRun } from '../probeHooks';
import { formatDuration } from '../format';
import { Unavailable } from './Unavailable';
import type { PomProbeFact, PomProbeNode, PomProbeRun } from '../types';

/** How a host was matched, spelled out. The raw values are terse to the point of coy. */
const RESOLUTION_LABEL: Record<string, string> = {
  name: 'by node name',
  address: 'by node address',
  orphaned: 'no executor',
};

/**
 * A sub-second host time still deserves a number.
 *
 * `formatDuration` floors to whole seconds, which is right for an oplog window of
 * days and wrong here: a host that failed in 0.28s would read as `0s`, hiding that it
 * failed instantly rather than after a wait.
 */
function formatHostDuration(seconds: number): string {
  if (seconds < 10) {
    return `${seconds.toFixed(2)}s`;
  }
  return formatDuration(seconds) || `${Math.round(seconds)}s`;
}

/** Render one fact's value, whatever JSON shape the probe read. */
function factValue(value: unknown): string {
  if (value === null || value === undefined) {
    return '—';
  }
  if (typeof value === 'string') {
    return value;
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  return JSON.stringify(value);
}

/** Every fact this service returned, in the order the probe reads them. */
function NodeFacts({ facts }: { facts: PomProbeFact[] }) {
  if (!facts.length) {
    return (
      <Typography variant="body2" color="text.secondary">
        No facts. The host did not answer for this service, or PMM has no id to
        key them by.
      </Typography>
    );
  }
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: 'minmax(140px, max-content) 1fr',
        columnGap: 2,
        rowGap: 0.5,
      }}
    >
      {facts.map((fact) => (
        <Box key={fact.field} sx={{ display: 'contents' }}>
          <Typography variant="body2" color="text.secondary">
            {fact.field}
          </Typography>
          <Typography
            variant="body2"
            sx={{ fontFamily: 'monospace', overflowWrap: 'anywhere' }}
          >
            {factValue(fact.value)}
          </Typography>
        </Box>
      ))}
    </Box>
  );
}

/** One service's row, unfolding to the facts the sweep stored for it. */
function NodeRow({
  node,
  facts,
}: {
  node: PomProbeNode;
  facts: PomProbeFact[];
}) {
  const [open, setOpen] = useState(false);
  const orphaned = node.resolution === 'orphaned';

  return (
    <>
      <TableRow hover>
        <TableCell padding="checkbox">
          <IconButton
            size="small"
            onClick={() => setOpen(!open)}
            aria-label={open ? 'Hide facts' : 'Show facts'}
            aria-expanded={open}
          >
            {open ? (
              <KeyboardArrowDownIcon fontSize="small" />
            ) : (
              <KeyboardArrowRightIcon fontSize="small" />
            )}
          </IconButton>
        </TableCell>
        <TableCell>{node.service_name}</TableCell>
        <TableCell>
          {node.executor_host ?? (
            // Orphaned is a fact about the estate, not a gap in the sweep: an
            // inventory row routinely outlives the executor that served it.
            <Unavailable reason="service_not_observed" />
          )}
        </TableCell>
        <TableCell>
          <Typography variant="body2" color="text.secondary">
            {RESOLUTION_LABEL[node.resolution] ?? node.resolution}
          </Typography>
        </TableCell>
        <TableCell>
          <Chip
            size="small"
            variant={node.answered ? 'filled' : 'outlined'}
            color={node.answered ? 'success' : orphaned ? 'default' : 'error'}
            label={
              node.answered ? 'Answered' : orphaned ? 'Orphaned' : 'No answer'
            }
          />
        </TableCell>
        <TableCell>
          {node.duration_seconds == null ? (
            <Unavailable reason="not_applicable" />
          ) : (
            <Tooltip title="Wall-clock for this host's dispatch, shared by every service it served">
              <Box component="span">
                {formatHostDuration(node.duration_seconds)}
              </Box>
            </Tooltip>
          )}
        </TableCell>
        <TableCell>{node.facts_collected}</TableCell>
      </TableRow>
      <TableRow>
        <TableCell sx={{ py: 0, borderBottom: 0 }} colSpan={7}>
          <Collapse in={open} unmountOnExit>
            <Stack gap={1} sx={{ py: 2, pl: 6, pr: 2 }}>
              {node.error && (
                <Alert severity="warning" variant="outlined">
                  {node.error}
                </Alert>
              )}
              <NodeFacts facts={facts} />
            </Stack>
          </Collapse>
        </TableCell>
      </TableRow>
    </>
  );
}

/**
 * What one sweep saw, service by service.
 *
 * Fetched on unfold rather than with the list: this is the per-run detail the list
 * endpoint deliberately leaves out. Every fact the probe read is here, including the
 * ones neither Overview nor Topology renders — the sweep collects a dozen fields per
 * service and POM's document maps three of them.
 */
export function RunNodes({ run }: { run: PomProbeRun }) {
  // Unfolding a sweep that is still running is the common case - it takes tens of
  // seconds - and its detail is empty until it ends, so that answer must not be
  // cached as if it were final.
  const { data, isPending, isError, error } = usePomProbeRun(
    run.run_id,
    true,
    isProbeRunActive(run.status)
  );

  const factsByService = useMemo(() => {
    const grouped = new Map<string, PomProbeFact[]>();
    for (const fact of data?.facts ?? []) {
      const existing = grouped.get(fact.service_id);
      if (existing) {
        existing.push(fact);
      } else {
        grouped.set(fact.service_id, [fact]);
      }
    }
    return grouped;
  }, [data]);

  if (isPending) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
        <CircularProgress size={20} />
      </Box>
    );
  }

  if (isError) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        Could not load this sweep: {(error as Error).message}
      </Alert>
    );
  }

  return (
    <Stack gap={2} sx={{ p: 2 }}>
      {data.error && (
        <Alert severity="warning" variant="outlined">
          {data.error}
        </Alert>
      )}
      {data.nodes.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          {/* Three different situations end up here, and saying "no detail" to all
              three reads as data loss when two of them have a plain explanation.
              The records are written when a sweep ends, so a running one has none
              yet; a sweep that failed before mapping anything never had any, and
              says why in the alert above; only the remaining case is the sweep
              recorded before this column existed, which cannot be reconstructed. */}
          {isProbeRunActive(data.status)
            ? 'This sweep is still running; per-service detail is recorded when it finishes.'
            : data.error
              ? 'This sweep failed before it mapped any service, so there is no per-service detail.'
              : 'This sweep recorded no per-service detail.'}
        </Typography>
      ) : (
        <Box sx={{ overflowX: 'auto' }}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell padding="checkbox" />
                <TableCell>Service</TableCell>
                <TableCell>Executor host</TableCell>
                <TableCell>Resolved</TableCell>
                <TableCell>Outcome</TableCell>
                <TableCell>Host took</TableCell>
                <TableCell>Facts</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {data.nodes.map((node) => (
                <NodeRow
                  key={node.service_name}
                  node={node}
                  facts={
                    node.service_id
                      ? (factsByService.get(node.service_id) ?? [])
                      : []
                  }
                />
              ))}
            </TableBody>
          </Table>
        </Box>
      )}
    </Stack>
  );
}
