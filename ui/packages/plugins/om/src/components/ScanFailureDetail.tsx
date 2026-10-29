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

import { Box, Link, Typography } from '@mui/material';
import { Link as RouterLink } from 'react-router-dom';
import { formatTimestamp } from '@pmm-extensions/framework';
import { OM_ROUTE_AUTOMATIONS } from '../constants';
import { formatCompactDuration } from '../format';
import type { ScanFailure } from '../inventory';
import { useOmBase } from '../useOmBase';

/**
 * A failing row's failure, in full: a node's detail panel, a service's drawer.
 *
 * The row can only say it in short. This is the whole of it: the raw error untrimmed
 * and with its line breaks (a traceback is unreadable folded onto one line), what kind
 * of failure that is, and what to do about it. The hint sits below the error because
 * some of them refer to "the excerpt above". The run link is only drawn when the
 * server names the run; a link to the scan history in general would send the reader
 * hunting through it, which is the trip this panel exists to save.
 */
export const ScanFailureDetail = ({
  failure,
  neutral = false,
}: {
  failure: ScanFailure;
  /** True for the PMM Server's own node; see NodesPage's countsAsFailing. */
  neutral?: boolean;
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
