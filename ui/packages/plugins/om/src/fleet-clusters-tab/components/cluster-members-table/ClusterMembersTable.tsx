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

import Box from '@mui/material/Box';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { StatusBadge } from '../../../components/HealthBadge';
import { MemberState } from '../../../components/MemberState';
import { Duration } from '../../../components/Metric';
import { ServiceLink } from '../../../components/ServiceLink';
import { Unavailable } from '../../../components/Unavailable';
import { PROCESS_ROLE_LABEL } from '../../../constants';
import { NESTED_TABLE_WRAPPER } from '../../../nestedTable';
import { downFirst } from '../../../topology';
import type { OmClusterRow } from '../../../types';
import { Messages } from '../../FleetClustersTab.messages';

const { columns } = Messages.members;

export const ClusterMembersTable = ({ cluster }: { cluster: OmClusterRow }) => {
  if (!cluster.services.length) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
        {Messages.members.empty}
      </Typography>
    );
  }
  return (
    <Box sx={NESTED_TABLE_WRAPPER}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>{columns.service}</TableCell>
            <TableCell>{columns.node}</TableCell>
            <TableCell>{columns.status}</TableCell>
            <TableCell>{columns.memberState}</TableCell>
            <TableCell>{columns.process}</TableCell>
            <TableCell>{columns.version}</TableCell>
            <TableCell>{columns.replicationLag}</TableCell>
            <TableCell>{columns.oplogWindow}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {downFirst(cluster.services).map((service) => (
            <TableRow key={service.service_name}>
              <TableCell>
                <ServiceLink serviceName={service.service_name} />
              </TableCell>
              <TableCell>
                {service.host ?? <Unavailable reason="service_not_observed" />}
              </TableCell>
              <TableCell>
                <StatusBadge
                  status={service.status}
                  lastUpAt={service.last_up_at}
                />
              </TableCell>
              <TableCell>
                <MemberState service={service} />
              </TableCell>
              <TableCell>
                {PROCESS_ROLE_LABEL[service.process_role] ??
                  service.process_role}
              </TableCell>
              <TableCell>
                {service.version ?? (
                  <Unavailable reason="service_not_observed" />
                )}
              </TableCell>
              <TableCell>
                <Duration value={service.replication_lag_seconds} />
              </TableCell>
              <TableCell>
                <Duration value={service.oplog_window_seconds} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Box>
  );
};
