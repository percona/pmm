import React, { FC, useMemo } from 'react';
import {
  Badge,
  CellProps,
  InteractiveTable,
  useStyles2,
} from '@grafana/ui';
import { NO_VALUE, ROLE_COLORS } from './AuroraTreePanel.constants';
import { Messages } from './AuroraTreePanel.messages';
import { getStyles } from './AuroraTreePanel.styles';
import {
  formatStorage,
  formatThroughput,
  getInstanceDetails,
  getInstanceLinks,
} from './AuroraTreePanel.tools';
import { InstanceGroup, RdsInstance, TableColumn } from './AuroraTreePanel.types';

interface InstancesTableProps {
  group: InstanceGroup;
}

const getInstanceId = (instance: RdsInstance) => instance.serviceName;

const RoleCell = ({ row: { original } }: CellProps<RdsInstance>) => (
  <Badge text={original.role} color={ROLE_COLORS[original.role]} />
);

const StatusCell = ({ row: { original } }: CellProps<RdsInstance>) => {
  const styles = useStyles2(getStyles);

  if (original.up === undefined) {
    return <>{NO_VALUE}</>;
  }

  return original.up === 1
    ? <span className={styles.good}>{Messages.up}</span>
    : <span className={styles.bad}>{Messages.down}</span>;
};

// Plain Grafana URLs are enough: nginx redirects a top-level /graph navigation (the new tab) into /pmm-ui.
const NEW_TAB = { target: '_blank', rel: 'noopener noreferrer' } as const;

const LinksCell = ({ row: { original } }: CellProps<RdsInstance>) => {
  const styles = useStyles2(getStyles);

  return (
    <span className={styles.links}>
      {getInstanceLinks(original).dashboards.map(({ label, title, url }) => (
        <a key={label} className={styles.link} {...NEW_TAB} href={url} title={title} aria-label={title}>
          {label}
        </a>
      ))}
    </span>
  );
};

const ThroughputCell = ({ row: { original } }: CellProps<RdsInstance>) => (
  <>{formatThroughput(original.throughput, original.engine)}</>
);

const StorageCell = ({ row: { original } }: CellProps<RdsInstance>) => <>{formatStorage(original.storage)}</>;

export const InstancesTable: FC<InstancesTableProps> = ({ group }) => {
  const styles = useStyles2(getStyles);
  const columns = useMemo((): Array<TableColumn<RdsInstance>> => [
    {
      id: 'serviceName',
      header: Messages.columns.service,
      cell: ({ row: { original } }) => {
        const { service } = getInstanceLinks(original);
        const details = getInstanceDetails(original, group);

        return (
          <>
            <a className={styles.link} {...NEW_TAB} href={service.url} title={service.title}>
              {service.label}
            </a>
            {details && <span className={styles.instanceDetails}>{details}</span>}
          </>
        );
      },
    },
    { id: 'nodeName', header: Messages.columns.links, cell: LinksCell },
    { id: 'role', header: Messages.columns.role, cell: RoleCell },
    { id: 'up', header: Messages.columns.status, cell: StatusCell },
    { id: 'throughput', header: Messages.columns.throughput, cell: ThroughputCell },
    { id: 'storage', header: Messages.columns.storage, cell: StorageCell },
  ], [styles, group]);

  return <InteractiveTable columns={columns} data={group.instances} getRowId={getInstanceId} />;
};
