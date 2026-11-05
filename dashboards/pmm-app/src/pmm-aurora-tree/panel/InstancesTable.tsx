import React, { FC, useMemo } from 'react';
import { TimeRange } from '@grafana/data';
import {
  Badge,
  CellProps,
  InteractiveTable,
  useStyles2,
} from '@grafana/ui';
import {
  NO_VALUE,
  QPS_DECIMALS,
  QPS_UNIT,
  ROLE_COLORS,
} from './AuroraTreePanel.constants';
import { Messages } from './AuroraTreePanel.messages';
import { getStyles } from './AuroraTreePanel.styles';
import { formatValue, getServiceUrl } from './AuroraTreePanel.tools';
import { AuroraInstance, TableColumn } from './AuroraTreePanel.types';

interface InstancesTableProps {
  instances: AuroraInstance[];
  timeRange: TimeRange;
}

const getInstanceId = (instance: AuroraInstance) => instance.serviceName;

const RoleCell = ({ row: { original } }: CellProps<AuroraInstance>) => (
  <Badge text={original.role} color={ROLE_COLORS[original.role]} />
);

const StatusCell = ({ row: { original } }: CellProps<AuroraInstance>) => {
  const styles = useStyles2(getStyles);

  if (original.up === undefined) {
    return <>{NO_VALUE}</>;
  }

  return original.up === 1
    ? <span className={styles.good}>{Messages.up}</span>
    : <span className={styles.bad}>{Messages.down}</span>;
};

const QpsCell = ({ row: { original } }: CellProps<AuroraInstance>) => (
  <>{formatValue(original.qps, QPS_UNIT, QPS_DECIMALS)}</>
);

export const InstancesTable: FC<InstancesTableProps> = ({ instances, timeRange }) => {
  const styles = useStyles2(getStyles);
  const columns = useMemo((): Array<TableColumn<AuroraInstance>> => [
    {
      id: 'serviceName',
      header: Messages.columns.service,
      cell: ({ row: { original } }) => (
        <a className={styles.link} href={getServiceUrl(original.serviceName, timeRange)}>
          {original.serviceName}
        </a>
      ),
    },
    { id: 'role', header: Messages.columns.role, cell: RoleCell },
    { id: 'up', header: Messages.columns.status, cell: StatusCell },
    { id: 'qps', header: Messages.columns.qps, cell: QpsCell },
  ], [styles, timeRange]);

  return <InteractiveTable columns={columns} data={instances} getRowId={getInstanceId} />;
};
