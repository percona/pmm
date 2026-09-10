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

import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import type { StreamError } from '../../hooks/useTaskLogs';

interface ExecutorGoneDetail {
  message?: string;
  resource_type?: string;
  resource_id?: string;
  job_id?: string;
  evaluation_id?: string;
  executor_name?: string;
  detail?: string;
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function formatGenericDetail(detail: unknown): string {
  if (typeof detail === 'string') {
    return detail;
  }
  if (detail === null || detail === undefined) {
    return 'Unknown stream error';
  }
  try {
    return JSON.stringify(detail, null, 2);
  } catch {
    return String(detail);
  }
}

function ExecutorGoneBlock({ detail }: { detail: ExecutorGoneDetail }) {
  const summary =
    detail.message || 'This run is no longer available in the task executor.';
  const rows: [string, string][] = [];
  if (detail.resource_type) {
    rows.push(['Resource type', detail.resource_type]);
  }
  if (detail.resource_id) {
    rows.push(['Resource', detail.resource_id]);
  }
  if (detail.job_id) {
    rows.push(['Job ID', detail.job_id]);
  }
  if (detail.evaluation_id) {
    rows.push(['Evaluation ID', detail.evaluation_id]);
  }
  if (detail.executor_name) {
    rows.push(['Executor', detail.executor_name]);
  }

  return (
    <Alert severity="warning" variant="outlined" sx={{ mb: 1 }} role="alert">
      <Typography variant="body2" sx={{ mb: rows.length ? 1 : 0 }}>
        {summary}
      </Typography>
      {rows.length > 0 && (
        <Box component="ul" sx={{ m: 0, pl: 2 }}>
          {rows.map(([k, v]) => (
            <Box component="li" key={k} sx={{ fontSize: '0.85rem' }}>
              <strong>{k}: </strong>
              {v}
            </Box>
          ))}
        </Box>
      )}
      {detail.detail && detail.detail !== summary && (
        <Typography variant="caption" component="p" sx={{ mt: 1 }}>
          {detail.detail}
        </Typography>
      )}
    </Alert>
  );
}

export interface StreamErrorBlockProps {
  error: StreamError;
}

export function StreamErrorBlock({ error }: StreamErrorBlockProps) {
  const { code, detail } = error;
  if (code === 410 && isObject(detail)) {
    return <ExecutorGoneBlock detail={detail as ExecutorGoneDetail} />;
  }
  return (
    <Alert severity="error" variant="outlined" sx={{ mb: 1 }} role="alert">
      <Box
        component="pre"
        sx={{
          m: 0,
          fontFamily: "'Roboto Mono', monospace",
          fontSize: '0.8rem',
          whiteSpace: 'pre-wrap',
        }}
      >
        {formatGenericDetail(detail)}
      </Box>
    </Alert>
  );
}
