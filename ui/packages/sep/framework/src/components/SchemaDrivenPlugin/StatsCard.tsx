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

import { Alert, Box, Paper, Skeleton, Typography } from '@mui/material';
import { ApiError } from '@sep/api';
import { useTaskStats } from '../../hooks/useTaskStats';

const PLACEHOLDER = '—';

function formatSeconds(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return PLACEHOLDER;
  }
  return `${value.toFixed(3)}s`;
}

const RELATIVE_THRESHOLDS: Array<[number, Intl.RelativeTimeFormatUnit]> = [
  [60, 'second'],
  [60, 'minute'],
  [24, 'hour'],
  [7, 'day'],
  [4.345, 'week'],
  [12, 'month'],
  [Number.POSITIVE_INFINITY, 'year'],
];

function formatRelative(iso: string | null | undefined): string {
  if (!iso || typeof iso !== 'string') {
    return PLACEHOLDER;
  }
  const date = new Date(iso);
  const ms = date.getTime();
  if (!Number.isFinite(ms)) {
    return PLACEHOLDER;
  }
  let diff = (ms - Date.now()) / 1000;
  const formatter = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });
  for (const [bound, unit] of RELATIVE_THRESHOLDS) {
    if (Math.abs(diff) < bound) {
      return formatter.format(Math.round(diff), unit);
    }
    diff /= bound;
  }
  return formatter.format(Math.round(diff), 'year');
}

function SectionShell({ children }: { children: React.ReactNode }) {
  return (
    <Paper variant="outlined" sx={{ p: 3, mb: 3 }}>
      <Typography variant="h6" sx={{ mb: 2 }}>
        Stats
      </Typography>
      {children}
    </Paper>
  );
}

function StatField({
  label,
  value,
}: {
  label: string;
  value: React.ReactNode;
}) {
  return (
    <Box sx={{ minWidth: 140 }}>
      <Typography
        variant="caption"
        color="text.secondary"
        sx={{ display: 'block', mb: 0.5 }}
      >
        {label}
      </Typography>
      <Typography variant="body1">{value}</Typography>
    </Box>
  );
}

interface StatsCardProps {
  taskName: string | undefined;
}

export function StatsCard({ taskName }: StatsCardProps) {
  const trimmed = taskName?.trim();
  const enabled = Boolean(trimmed);
  const { data, isLoading, isError, error } = useTaskStats(trimmed, enabled);

  if (!enabled) {
    return null;
  }

  if (isLoading) {
    return (
      <SectionShell>
        <Skeleton variant="rectangular" height={80} />
      </SectionShell>
    );
  }

  if (isError) {
    const status = error instanceof ApiError ? error.status : undefined;
    if (status === 404) {
      return (
        <SectionShell>
          <Typography color="text.secondary">
            No execution history yet
          </Typography>
        </SectionShell>
      );
    }
    return (
      <SectionShell>
        <Alert severity="info">Could not load execution stats</Alert>
      </SectionShell>
    );
  }

  if (!data || !data.total) {
    return (
      <SectionShell>
        <Typography color="text.secondary">No execution history yet</Typography>
      </SectionShell>
    );
  }

  const succeeded = data.status?.pass ?? 0;
  const failed = data.status?.fail ?? 0;

  return (
    <SectionShell>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 3 }}>
        <StatField label="Executions" value={data.total} />
        <StatField label="Succeeded" value={succeeded} />
        <StatField label="Failed" value={failed} />
        <StatField
          label="Last Finished"
          value={formatRelative(data.last_finished_at)}
        />
        <StatField
          label="Avg Duration"
          value={formatSeconds(data.duration?.average_seconds)}
        />
        <StatField
          label="Last Duration"
          value={formatSeconds(data.duration?.last_seconds)}
        />
      </Box>
    </SectionShell>
  );
}
