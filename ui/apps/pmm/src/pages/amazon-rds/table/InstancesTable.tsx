import type { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { Chip } from '@percona/peak-ui';
import { NO_VALUE } from '../AmazonRds.constants';
import { Messages } from '../AmazonRds.messages';
import type {
  InstanceGroup,
  InstanceLink,
  InstanceRole,
  RdsInstance,
} from '../AmazonRds.types';
import {
  formatStorage,
  formatThroughput,
  getInstanceDetails,
  getInstanceLinks,
} from '../AmazonRds.utils';

const ROLE_CHIPS: Record<
  InstanceRole,
  { color: 'info' | 'default'; variant: 'filled' | 'outlined' }
> = {
  writer: { color: 'info', variant: 'filled' },
  reader: { color: 'default', variant: 'filled' },
  unknown: { color: 'default', variant: 'outlined' },
};

// Dashboards open in a new tab, inside the PMM shell like every other in-app path. A short
// label (MySQL, OS) is named after its dashboard for assistive technologies.
const DashboardLink: FC<{ link: InstanceLink; named?: boolean }> = ({
  link,
  named,
}) => (
  <Link
    component={RouterLink}
    to={link.url}
    target="_blank"
    rel="noopener noreferrer"
    title={link.title}
    aria-label={named ? link.title : undefined}
  >
    {link.label}
  </Link>
);

const Status: FC<{ up?: number }> = ({ up }) => {
  if (up === undefined) {
    return <>{NO_VALUE}</>;
  }

  return up === 1 ? (
    <Typography variant="body2" color="success.main">
      {Messages.up}
    </Typography>
  ) : (
    <Typography variant="body2" color="error.main">
      {Messages.down}
    </Typography>
  );
};

const InstanceRow: FC<{ instance: RdsInstance; group: InstanceGroup }> = ({
  instance,
  group,
}) => {
  const { service, dashboards } = getInstanceLinks(instance);
  const details = getInstanceDetails(instance, group);

  return (
    <TableRow>
      <TableCell>
        <Stack direction="row" alignItems="baseline" gap={1}>
          <DashboardLink link={service} />
          {details && (
            <Typography variant="body2" color="text.secondary">
              {details}
            </Typography>
          )}
        </Stack>
      </TableCell>
      <TableCell>
        <Stack direction="row" gap={1.5}>
          {dashboards.map((link) => (
            <DashboardLink key={link.label} link={link} named />
          ))}
        </Stack>
      </TableCell>
      <TableCell>
        <Chip
          size="small"
          label={instance.role}
          {...ROLE_CHIPS[instance.role]}
        />
      </TableCell>
      <TableCell>
        <Status up={instance.up} />
      </TableCell>
      <TableCell>
        {formatThroughput(instance.throughput, instance.engine)}
      </TableCell>
      <TableCell>{formatStorage(instance.storage)}</TableCell>
    </TableRow>
  );
};

export const InstancesTable: FC<{ group: InstanceGroup }> = ({ group }) => (
  <Table size="small" aria-label={group.name}>
    <TableHead>
      <TableRow>
        <TableCell>{Messages.columns.service}</TableCell>
        <TableCell>{Messages.columns.links}</TableCell>
        <TableCell>{Messages.columns.role}</TableCell>
        <TableCell>{Messages.columns.status}</TableCell>
        <TableCell>{Messages.columns.throughput}</TableCell>
        <TableCell>{Messages.columns.storage}</TableCell>
      </TableRow>
    </TableHead>
    <TableBody>
      {group.instances.map((instance) => (
        <InstanceRow
          key={instance.serviceName}
          instance={instance}
          group={group}
        />
      ))}
    </TableBody>
  </Table>
);
