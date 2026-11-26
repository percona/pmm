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

import ScheduleIcon from '@mui/icons-material/Schedule';
import Stack from '@mui/material/Stack';
import { Age } from '../../../components/Age';
import { Unavailable } from '../../../components/Unavailable';
import { formatCompactDuration } from '../../../format';
import { LAST_SEEN_STALE_SECONDS } from '../../FleetClustersTab.constants';
import { Messages } from '../../FleetClustersTab.messages';

/** When the cluster was last observed, with a clock once that is stale. */
export const LastSeenCell = ({ lastSeenAt }: { lastSeenAt: string | null }) => {
  if (!lastSeenAt) {
    return <Unavailable reason="service_not_observed" />;
  }
  const stale =
    Date.now() - Date.parse(lastSeenAt) > LAST_SEEN_STALE_SECONDS * 1000;
  if (!stale) {
    return <Age value={lastSeenAt} />;
  }
  return (
    <Stack direction="row" alignItems="center" gap={0.5}>
      <ScheduleIcon
        fontSize="inherit"
        titleAccess={Messages.lastSeenStale(
          formatCompactDuration(LAST_SEEN_STALE_SECONDS)
        )}
        sx={{ color: 'text.secondary' }}
        data-testid="om-last-seen-stale"
      />
      <Age value={lastSeenAt} />
    </Stack>
  );
};
