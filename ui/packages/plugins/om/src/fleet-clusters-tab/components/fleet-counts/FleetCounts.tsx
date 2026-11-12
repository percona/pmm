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

import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { pluralize } from '../../../format';
import type { OmTopologySummary } from '../../../types';

export const FleetCounts = ({ summary }: { summary: OmTopologySummary }) => {
  const { environments, clusters, total_services, up_services, down_services } =
    summary;
  return (
    <Stack direction="row" spacing={3} flexWrap="wrap">
      <Typography variant="body2">
        <strong>{environments}</strong> {pluralize(environments, 'environment')}
      </Typography>
      <Typography variant="body2">
        <strong>{clusters}</strong> {pluralize(clusters, 'cluster')}
      </Typography>
      <Typography variant="body2">
        <strong>{total_services}</strong> {pluralize(total_services, 'service')}
      </Typography>
      <Typography variant="body2">
        <strong>{up_services}</strong> up
      </Typography>
      <Typography
        variant="body2"
        color={down_services ? 'error.main' : undefined}
      >
        <strong>{down_services}</strong> down
      </Typography>
    </Stack>
  );
};
