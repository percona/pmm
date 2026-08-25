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

import { Box, Tooltip } from '@mui/material';
import {
  ageSeconds,
  missingRowReason,
  type OmEstateStatus,
} from '../inventory';
import { formatCompactDuration } from '../format';
import { Unavailable } from './Unavailable';
import type { OmInventoryService } from '../types';

/**
 * One value the probe collected, shown with its age when that age matters.
 *
 * `Unavailable` covers "there is no value and here is why". The estate needs a third
 * answer it cannot express: *there is a value, it is old, and the last attempt to
 * refresh it failed*. Rendering that as a dash would throw away the only information
 * anyone has about the service, and rendering it plainly would present three-day-old
 * facts as current.
 *
 * So a failing row keeps its value and gains its age. That is the design the API
 * commits to - stale rows are served with their ages rather than filtered out,
 * because the consumer decides - and this is where a reader sees it.
 */
export function ProbeValue({
  inventory,
  value,
  estate = 'ready',
}: {
  /** The estate row, or null when OM has never held one for this service. */
  inventory: OmInventoryService | null;
  /** The field being shown, read off that row by the caller. */
  value: string | number | null | undefined;
  /**
   * What the estate query is doing, which decides what a missing row means.
   *
   * Without it a missing row is reported as "not in the inventory", which is a
   * statement about the estate this page is in no position to make while it is still
   * reading it - or when it could not read it at all.
   */
  estate?: OmEstateStatus;
}) {
  // No row at all: PMM registered this service after the last sweep, so nothing has
  // ever been dispatched for it. Different from a probe that ran and found nothing.
  if (!inventory) {
    return <Unavailable reason={missingRowReason(estate)} />;
  }
  if (value === null || value === undefined || value === '') {
    // A row exists. Whether it has ever answered is the useful distinction: never is
    // an onboarding problem, once-but-not-this-field is a probe that ran and could
    // not read it.
    return (
      <Unavailable
        reason={
          inventory.freshness.last_success_at
            ? 'metric_not_collected'
            : 'probe_never_succeeded'
        }
      />
    );
  }

  const failing = inventory.freshness.failing_since != null;
  if (!failing) {
    return <>{value}</>;
  }

  const age = ageSeconds(inventory.freshness.last_success_at);
  const since = ageSeconds(inventory.freshness.failing_since);
  const detail = [
    age == null
      ? 'This service has never answered a probe.'
      : `Last collected ${formatCompactDuration(age)} ago.`,
    since == null
      ? null
      : `Failing for ${formatCompactDuration(since)} (${inventory.freshness.consecutive_failures} attempts).`,
    inventory.freshness.last_error,
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <Tooltip title={detail}>
      {/* Focusable so the detail -- which carries last_error -- is reachable without a
          pointer. MUI opens a Tooltip on focus, but only if the anchor can take it. */}
      <Box
        component="span"
        tabIndex={0}
        sx={{
          cursor: 'help',
          '&:focus-visible': {
            outline: '2px solid',
            outlineColor: 'primary.main',
            outlineOffset: 2,
            borderRadius: 0.5,
          },
        }}
      >
        {value}
        <Box
          component="span"
          sx={{ color: 'warning.main', ml: 0.5, whiteSpace: 'nowrap' }}
        >
          {age == null ? '(stale)' : `(${formatCompactDuration(age)} old)`}
        </Box>
      </Box>
    </Tooltip>
  );
}
