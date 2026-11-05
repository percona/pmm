import React, {
  FC,
  Fragment,
  useMemo,
  useState,
} from 'react';
import { Icon, IconButton, useStyles2 } from '@grafana/ui';
import {
  QPS_DECIMALS,
  QPS_UNIT,
  STORAGE_UNIT,
  UNKNOWN_CLUSTER,
} from './AuroraTreePanel.constants';
import { Messages } from './AuroraTreePanel.messages';
import { getStyles } from './AuroraTreePanel.styles';
import { compareSortValues, formatValue, getWritersText } from './AuroraTreePanel.tools';
import { AuroraCluster, ClusterColumn, ClusterSort } from './AuroraTreePanel.types';
import { InstancesTable } from './InstancesTable';

interface ClusterTableProps {
  clusters: AuroraCluster[];
  autoExpanded: Set<string>;
  // Rows the user expanded or collapsed are forgotten when this changes (the search text).
  expansionKey: string;
}

const ClusterName: FC<{ cluster: AuroraCluster }> = ({ cluster }) => {
  const styles = useStyles2(getStyles);

  return cluster.clusterId === UNKNOWN_CLUSTER
    ? <span className={styles.unknownCluster}>{cluster.clusterId}</span>
    : <>{cluster.clusterId}</>;
};

const Writers: FC<{ cluster: AuroraCluster }> = ({ cluster }) => {
  const styles = useStyles2(getStyles);

  return (
    <span className={cluster.writers === 1 ? styles.good : styles.warning}>
      {getWritersText(cluster.writers)}
    </span>
  );
};

const COLUMNS: ClusterColumn[] = [
  {
    id: 'region', header: Messages.columns.region, render: (c) => c.region, sortValue: (c) => c.region,
  },
  {
    id: 'awsAccountId',
    header: Messages.columns.awsAccount,
    render: (c) => c.awsAccountId,
    sortValue: (c) => c.awsAccountId,
  },
  {
    id: 'clusterId',
    header: Messages.columns.cluster,
    render: (c) => <ClusterName cluster={c} />,
    sortValue: (c) => c.clusterId,
  },
  {
    id: 'monitoredInstances',
    header: Messages.columns.monitoredInstances,
    render: (c) => c.monitoredInstances,
    sortValue: (c) => c.monitoredInstances,
  },
  {
    id: 'exportersConnected',
    header: Messages.columns.exportersConnected,
    render: (c) => c.exportersConnected,
    sortValue: (c) => c.exportersConnected,
  },
  {
    id: 'writers', header: Messages.columns.writers, render: (c) => <Writers cluster={c} />, sortValue: (c) => c.writers,
  },
  {
    id: 'qps',
    header: Messages.columns.qps,
    render: (c) => formatValue(c.qps, QPS_UNIT, QPS_DECIMALS),
    sortValue: (c) => c.qps,
  },
  {
    id: 'storageBytes',
    header: Messages.columns.storage,
    render: (c) => formatValue(c.storageBytes, STORAGE_UNIT),
    sortValue: (c) => c.storageBytes,
  },
];

// Clicking a header cycles ascending, descending, unsorted.
const nextSort = (sort: ClusterSort | undefined, id: string): ClusterSort | undefined => {
  if (sort?.id !== id) {
    return { id, desc: false };
  }

  return sort.desc ? undefined : { id, desc: true };
};

const getAriaSort = (sort: ClusterSort | undefined, id: string) => {
  if (sort?.id !== id) {
    return undefined;
  }

  return sort.desc ? 'descending' : 'ascending';
};

export const ClusterTable: FC<ClusterTableProps> = ({
  clusters, autoExpanded, expansionKey,
}) => {
  const styles = useStyles2(getStyles);
  const [sort, setSort] = useState<ClusterSort>();
  const [expansion, setExpansion] = useState({ key: expansionKey, overrides: {} as Record<string, boolean> });
  const overrides = expansion.key === expansionKey ? expansion.overrides : {};
  const isExpanded = (cluster: AuroraCluster) => overrides[cluster.key] ?? autoExpanded.has(cluster.key);
  const allExpanded = clusters.every(isExpanded);
  const setExpanded = (changed: AuroraCluster[], expanded: boolean) => setExpansion({
    key: expansionKey,
    overrides: changed.reduce((acc, cluster) => ({ ...acc, [cluster.key]: expanded }), overrides),
  });
  const sortedClusters = useMemo(() => {
    const column = COLUMNS.find(({ id }) => id === sort?.id);

    return column && sort
      ? [...clusters].sort((a, b) => compareSortValues(column.sortValue(a), column.sortValue(b), sort.desc))
      : clusters;
  }, [clusters, sort]);

  return (
    <table className={styles.table}>
      <thead>
        <tr>
          <th className={`${styles.header} ${styles.expanderCell}`}>
            <IconButton
              name={allExpanded ? 'table-collapse-all' : 'table-expand-all'}
              aria-label={allExpanded ? Messages.collapseAll : Messages.expandAll}
              variant="secondary"
              size="lg"
              onClick={() => setExpanded(clusters, !allExpanded)}
            />
          </th>
          {COLUMNS.map(({ id, header }) => (
            <th key={id} className={styles.header} aria-sort={getAriaSort(sort, id)}>
              <button type="button" className={styles.sortButton} onClick={() => setSort(nextSort(sort, id))}>
                {header}
                {sort?.id === id && <Icon name={sort.desc ? 'angle-down' : 'angle-up'} />}
              </button>
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {sortedClusters.map((cluster) => {
          const expanded = isExpanded(cluster);
          const contentId = `aurora-cluster-${cluster.key.replace(/[^\w-]/g, '-')}`;

          return (
            <Fragment key={cluster.key}>
              <tr className={styles.row}>
                <td className={`${styles.cell} ${styles.expanderCell}`}>
                  <IconButton
                    name={expanded ? 'angle-down' : 'angle-right'}
                    aria-label={expanded ? Messages.collapseRow : Messages.expandRow}
                    aria-expanded={expanded}
                    aria-controls={expanded ? contentId : undefined}
                    size="lg"
                    onClick={() => setExpanded([cluster], !expanded)}
                  />
                </td>
                {COLUMNS.map(({ id, render }) => (
                  <td key={id} className={styles.cell}>{render(cluster)}</td>
                ))}
              </tr>
              {expanded && (
                <tr id={contentId}>
                  {/* eslint-disable-next-line jsx-a11y/control-has-associated-label -- holds a table */}
                  <td className={styles.expandedCell} colSpan={COLUMNS.length + 1}>
                    <InstancesTable instances={cluster.instances} />
                  </td>
                </tr>
              )}
            </Fragment>
          );
        })}
      </tbody>
    </table>
  );
};
