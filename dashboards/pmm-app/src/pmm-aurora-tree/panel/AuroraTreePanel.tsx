import React, { FC, useMemo } from 'react';
import { PanelProps } from '@grafana/data';
import { Button, useStyles2 } from '@grafana/ui';
import { Messages } from './AuroraTreePanel.messages';
import { getStyles } from './AuroraTreePanel.styles';
import {
  filterClusters,
  getAccountOptions,
  getClusterOptions,
  getRegionOptions,
  groupClusters,
  narrowSelection,
  readInstances,
} from './AuroraTreePanel.tools';
import { AuroraTreeOptions, ClusterSelection } from './AuroraTreePanel.types';
import { ClusterTable } from './ClusterTable';
import { SelectFilter, Toolbar } from './Toolbar';
import { useClusterFilters } from './useClusterFilters';

export const AuroraTreePanel: FC<PanelProps<AuroraTreeOptions>> = ({ data, options }) => {
  const styles = useStyles2(getStyles);
  const {
    search, setSearch, filters, setSelection, clear,
  } = useClusterFilters();
  const clusters = useMemo(() => groupClusters(readInstances(data.series, options)), [data.series, options]);
  const filtered = useMemo(() => filterClusters(clusters, filters), [clusters, filters]);
  const selects = useMemo((): SelectFilter[] => {
    const select = (changed: Partial<ClusterSelection>) => setSelection(
      narrowSelection(clusters, { ...filters, ...changed }),
    );

    return [
      {
        id: 'aurora-tree-region',
        label: Messages.columns.region,
        placeholder: Messages.allRegions,
        options: getRegionOptions(clusters),
        value: filters.regions,
        onChange: (regions) => select({ regions }),
      },
      {
        id: 'aurora-tree-account',
        label: Messages.columns.awsAccount,
        placeholder: Messages.allAccounts,
        options: getAccountOptions(clusters, filters.regions),
        value: filters.accounts,
        onChange: (accounts) => select({ accounts }),
      },
      {
        id: 'aurora-tree-cluster',
        label: Messages.columns.cluster,
        placeholder: Messages.allClusters,
        options: getClusterOptions(clusters, filters),
        value: filters.clusters,
        onChange: (ids) => select({ clusters: ids }),
      },
    ];
  }, [clusters, filters, setSelection]);

  if (!clusters.length) {
    return <div className={styles.empty}>{Messages.noData}</div>;
  }

  return (
    <div className={styles.wrapper}>
      <Toolbar
        search={search}
        onSearchChange={setSearch}
        selects={selects}
        shown={filtered.clusters.length}
        total={clusters.length}
      />
      {filtered.clusters.length ? (
        <ClusterTable
          clusters={filtered.clusters}
          autoExpanded={filtered.autoExpanded}
          expansionKey={filters.query}
        />
      ) : (
        <div className={styles.empty}>
          {Messages.noMatch}
          <Button variant="secondary" fill="text" size="sm" icon="times" onClick={clear}>
            {Messages.clearFilters}
          </Button>
        </div>
      )}
    </div>
  );
};
