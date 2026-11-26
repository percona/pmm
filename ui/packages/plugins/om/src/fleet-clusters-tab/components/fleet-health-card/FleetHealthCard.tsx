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

import FilterListIcon from '@mui/icons-material/FilterList';
import MonitorHeartOutlinedIcon from '@mui/icons-material/MonitorHeartOutlined';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { MRT_TableInstance } from 'material-react-table';
import { CLUSTER_HEALTH_LABEL } from '../../../constants';
import type { OmClusterRow } from '../../../types';
import { HEALTH_FILTER_VALUES } from '../../FleetClustersTab.constants';
import { Messages } from '../../FleetClustersTab.messages';
import { countValues } from '../../FleetClustersTab.utils';
import { ClusterHealthIcon } from '../cluster-health-icon';
import {
  HEALTH_BAR_COLOR,
  UNHEALTHY_STATES,
} from './FleetHealthCard.constants';

const { healthCard } = Messages;

/** How many clusters are not healthy, of the whole fleet, and a way to list them. */
export const FleetHealthCard = ({
  table,
  rows,
}: {
  table: MRT_TableInstance<OmClusterRow>;
  rows: OmClusterRow[];
}) => {
  const byHealth = countValues(rows, (row) => row.health);
  const present = HEALTH_FILTER_VALUES.filter((health) => byHealth[health]);
  const unhealthy = present.filter((health) => health !== 'healthy');
  const unhealthyCount = unhealthy.reduce((sum, h) => sum + byHealth[h], 0);

  return (
    <Paper
      variant="outlined"
      sx={{ p: 2, width: { xs: '100%', md: 1 / 3 }, minWidth: 320 }}
      data-testid="om-fleet-health-card"
    >
      <Stack gap={1.5}>
        <Stack direction="row" justifyContent="space-between">
          <Typography variant="overline" color="text.secondary">
            {healthCard.title}
          </Typography>
          <MonitorHeartOutlinedIcon fontSize="small" color="action" />
        </Stack>
        <Stack direction="row" alignItems="baseline" gap={1}>
          <Typography variant="h4" component="span">
            {unhealthyCount}
          </Typography>
          <Typography variant="h6" component="span" color="text.secondary">
            / {rows.length}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {healthCard.unhealthy(rows.length)}
          </Typography>
        </Stack>
        <Stack direction="row" gap={0.5} aria-hidden>
          {present.map((health) => (
            <Box
              key={health}
              sx={{
                flexGrow: byHealth[health],
                height: 6,
                borderRadius: 1,
                bgcolor: HEALTH_BAR_COLOR[health],
              }}
            />
          ))}
        </Stack>
        <Box
          sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', rowGap: 0.5 }}
        >
          {present.map((health) => (
            <Stack
              key={health}
              direction="row"
              alignItems="center"
              gap={0.75}
              data-testid={`om-fleet-health-${health}`}
            >
              <ClusterHealthIcon health={health} />
              <Typography variant="body2" color="text.secondary">
                {Messages.summary.health(
                  byHealth[health],
                  CLUSTER_HEALTH_LABEL[health]
                )}
              </Typography>
            </Stack>
          ))}
        </Box>
        {unhealthyCount > 0 && (
          <>
            <Divider />
            <Box>
              <Button
                size="small"
                endIcon={<FilterListIcon />}
                onClick={() =>
                  table.getColumn('health').setFilterValue(UNHEALTHY_STATES)
                }
              >
                {healthCard.showUnhealthy}
              </Button>
            </Box>
          </>
        )}
      </Stack>
    </Paper>
  );
};
