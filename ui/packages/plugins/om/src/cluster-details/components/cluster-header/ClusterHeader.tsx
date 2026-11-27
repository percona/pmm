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

import type { ReactElement, ReactNode } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Breadcrumbs from '@mui/material/Breadcrumbs';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import HubOutlinedIcon from '@mui/icons-material/HubOutlined';
import InsightsIcon from '@mui/icons-material/Insights';
import { MenuButton } from '@percona/peak-ui';
import { CLUSTER_TYPE_LABEL } from '../../../constants';
import { ClusterHealthBadge } from '../../../fleet-clusters-tab/components/cluster-health-badge';
import { TYPE_ICON } from '../../../fleet-clusters-tab/components/cluster-type-cell/ClusterTypeCell.constants';
import { ClusterVersionCell } from '../../../fleet-clusters-tab/components/cluster-version-cell';
import {
  clusterDisplayName,
  environmentLabel,
} from '../../../fleet-clusters-tab/FleetClustersTab.utils';
import type { OmClusterRow } from '../../../types';
import { useOmBase } from '../../../useOmBase';
import { AVAILABLE_CLUSTER_ACTIONS } from '../../ClusterDetails.actions';
import { Messages } from '../../ClusterDetails.messages';
import {
  buildConnectionString,
  clusterDashboardPath,
  shardCount,
  singleReplicaSet,
} from '../../ClusterDetails.utils';
import { ClusterSwitcher } from '../cluster-switcher';
import { ConnectionStringField } from '../connection-string-field';

const HeaderChip = ({
  icon,
  label,
  title,
}: {
  icon?: ReactElement;
  label: ReactNode;
  title?: string;
}) => (
  <Chip
    size="small"
    variant="outlined"
    icon={icon}
    label={label}
    title={title}
    sx={{ '& .MuiChip-icon': { color: 'text.secondary' } }}
  />
);

export const ClusterHeader = ({
  cluster,
  clusters,
}: {
  cluster: OmClusterRow;
  clusters: OmClusterRow[];
}) => {
  const base = useOmBase();
  const replicaSet = singleReplicaSet(cluster);
  const connectionString = buildConnectionString(cluster);
  const typeLabel = CLUSTER_TYPE_LABEL[cluster.type];
  const members = Messages.header.members(cluster.total_services);

  return (
    <Stack
      direction={{ xs: 'column', md: 'row' }}
      justifyContent="space-between"
      gap={2}
    >
      <Stack gap={1.5} sx={{ minWidth: 0 }}>
        <Breadcrumbs>
          <Link component={RouterLink} to={base} underline="hover">
            {Messages.fleet}
          </Link>
          <ClusterSwitcher current={cluster} clusters={clusters} />
        </Breadcrumbs>
        <Stack direction="row" alignItems="center" gap={1.5} flexWrap="wrap">
          <Typography variant="h4" component="h1">
            {clusterDisplayName(cluster)}
          </Typography>
          <ClusterHealthBadge health={cluster.health} />
        </Stack>
        <Stack
          direction="row"
          gap={1}
          flexWrap="wrap"
          data-testid="om-cluster-chips"
        >
          {typeLabel && (
            <HeaderChip icon={TYPE_ICON[cluster.type]} label={typeLabel} />
          )}
          {replicaSet && (
            <HeaderChip
              icon={<HubOutlinedIcon fontSize="small" />}
              label={replicaSet}
              title={Messages.header.replicaSet}
            />
          )}
          <HeaderChip
            label={
              cluster.type === 'CLUSTER_TYPE_SHARDED'
                ? `${Messages.header.shards(shardCount(cluster))} · ${members}`
                : members
            }
          />
          {cluster.versions.length > 0 && (
            <HeaderChip label={<ClusterVersionCell cluster={cluster} />} />
          )}
          <HeaderChip
            label={environmentLabel(cluster)}
            title={Messages.header.environment}
          />
        </Stack>
      </Stack>
      <Stack
        gap={1.5}
        alignItems={{ xs: 'flex-start', md: 'flex-end' }}
        sx={{ minWidth: 0 }}
      >
        <Stack direction="row" gap={1}>
          <Button
            component={RouterLink}
            to={clusterDashboardPath(cluster)}
            variant="outlined"
            startIcon={<InsightsIcon />}
          >
            {Messages.header.dashboard}
          </Button>
          {AVAILABLE_CLUSTER_ACTIONS.length > 0 && (
            <MenuButton buttonText={Messages.header.clusterActions}>
              {(handleClose: () => void) =>
                AVAILABLE_CLUSTER_ACTIONS.map((action) => (
                  <MenuItem key={action.id} onClick={handleClose}>
                    {action.label}
                  </MenuItem>
                ))
              }
            </MenuButton>
          )}
        </Stack>
        {connectionString && <ConnectionStringField value={connectionString} />}
      </Stack>
    </Stack>
  );
};
