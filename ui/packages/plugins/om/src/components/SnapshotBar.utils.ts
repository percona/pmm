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

import { formatAge, formatTimestamp } from '../format';
import type { OmLastScan } from '../inventoryHooks';

export const describeLastScan = ({
  status,
  finishedAt,
}: OmLastScan): { label: string; tooltip: string } => {
  if (status === 'pending') {
    return {
      label: 'Reading scan history…',
      tooltip: 'The scan history has not been read yet.',
    };
  }
  if (status === 'unavailable') {
    return {
      label: 'Scan history unavailable',
      tooltip:
        'The scan history could not be read, so when the nodes were last scanned is not known.',
    };
  }
  return finishedAt
    ? {
        label: `Nodes last scanned ${formatAge(finishedAt)}`,
        tooltip: `The last scan of every node finished ${formatTimestamp(finishedAt)}.`,
      }
    : {
        label: 'Nodes not scanned yet',
        tooltip: 'No scan of every node has finished yet.',
      };
};
