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

import { formatCompactDuration, pluralize } from '../format';
import type { OmInventoryRun } from '../types';

export const describeScanProgress = (
  run: OmInventoryRun,
  expectedSeconds: number | null,
  now: number
): string => {
  const parts: string[] = [];
  const total = run.counts.probeable_hosts;
  if (total > 0) {
    parts.push(
      `${run.counts.finished_hosts} of ${total} ${pluralize(total, 'node')} done`
    );
  }
  const elapsed = Math.max(0, (now - Date.parse(run.start_time)) / 1000);
  parts.push(`${formatCompactDuration(elapsed) || '0s'} so far`);
  if (expectedSeconds != null) {
    parts.push(
      `usually about ${formatCompactDuration(expectedSeconds) || '1s'}`
    );
  }
  return parts.join(' · ');
};
