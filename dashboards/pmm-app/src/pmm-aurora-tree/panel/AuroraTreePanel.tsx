import React, { FC, useCallback, useMemo } from 'react';
import { PanelProps } from '@grafana/data';
import { CellProps, InteractiveTable, useStyles2 } from '@grafana/ui';
import {
  QPS_DECIMALS,
  QPS_UNIT,
  STORAGE_UNIT,
  UNKNOWN_CLUSTER,
} from './AuroraTreePanel.constants';
import { Messages } from './AuroraTreePanel.messages';
import { getStyles } from './AuroraTreePanel.styles';
import {
  formatValue,
  getWritersText,
  groupClusters,
  readInstances,
} from './AuroraTreePanel.tools';
import { AuroraCluster, AuroraTreeOptions, TableColumn } from './AuroraTreePanel.types';
import { InstancesTable } from './InstancesTable';

const getClusterId = (cluster: AuroraCluster) => cluster.key;

const ClusterCell = ({ row: { original } }: CellProps<AuroraCluster>) => {
  const styles = useStyles2(getStyles);

  return original.clusterId === UNKNOWN_CLUSTER
    ? <span className={styles.unknownCluster}>{original.clusterId}</span>
    : <>{original.clusterId}</>;
};

const WritersCell = ({ row: { original } }: CellProps<AuroraCluster>) => {
  const styles = useStyles2(getStyles);

  return (
    <span className={original.writers === 1 ? styles.good : styles.warning}>
      {getWritersText(original.writers)}
    </span>
  );
};

const CLUSTER_COLUMNS: Array<TableColumn<AuroraCluster>> = [
  { id: 'region', header: Messages.columns.region, sortType: 'alphanumeric' },
  { id: 'awsAccountId', header: Messages.columns.awsAccount, sortType: 'alphanumeric' },
  {
    id: 'clusterId',
    header: Messages.columns.cluster,
    sortType: 'alphanumeric',
    cell: ClusterCell,
  },
  { id: 'monitoredInstances', header: Messages.columns.monitoredInstances, sortType: 'number' },
  { id: 'exportersConnected', header: Messages.columns.exportersConnected, sortType: 'number' },
  { id: 'writers', header: Messages.columns.writers, cell: WritersCell },
  {
    id: 'qps',
    header: Messages.columns.qps,
    cell: ({ row: { original } }) => formatValue(original.qps, QPS_UNIT, QPS_DECIMALS),
  },
  {
    id: 'storageBytes',
    header: Messages.columns.storage,
    cell: ({ row: { original } }) => formatValue(original.storageBytes, STORAGE_UNIT),
  },
];

export const AuroraTreePanel: FC<PanelProps<AuroraTreeOptions>> = ({ data, options, timeRange }) => {
  const styles = useStyles2(getStyles);
  const clusters = useMemo(() => groupClusters(readInstances(data.series, options)), [data.series, options]);
  const renderExpandedRow = useCallback(
    (cluster: AuroraCluster) => <InstancesTable instances={cluster.instances} timeRange={timeRange} />,
    [timeRange],
  );

  if (!clusters.length) {
    return <div className={styles.empty}>{Messages.noData}</div>;
  }

  return (
    <div className={styles.wrapper}>
      <InteractiveTable
        columns={CLUSTER_COLUMNS}
        data={clusters}
        getRowId={getClusterId}
        renderExpandedRow={renderExpandedRow}
        showExpandAll
      />
    </div>
  );
};
