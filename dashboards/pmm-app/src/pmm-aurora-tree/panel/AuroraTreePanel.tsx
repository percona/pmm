import React, { FC, useMemo } from 'react';
import { PanelProps } from '@grafana/data';
import { Button, useStyles2 } from '@grafana/ui';
import { Messages } from './AuroraTreePanel.messages';
import { getStyles } from './AuroraTreePanel.styles';
import {
  filterClusters,
  getAccountOptions,
  getClusterOptions,
  groupClusters,
  readInstances,
} from './AuroraTreePanel.tools';
import { AuroraTreeOptions } from './AuroraTreePanel.types';
import { ClusterTable } from './ClusterTable';
import { Toolbar } from './Toolbar';
import { useClusterFilters } from './useClusterFilters';

export const AuroraTreePanel: FC<PanelProps<AuroraTreeOptions>> = ({ data, options, timeRange }) => {
  const styles = useStyles2(getStyles);
  const {
    search, setSearch, filters, setAccounts, setClusters, clear,
  } = useClusterFilters();
  const clusters = useMemo(() => groupClusters(readInstances(data.series, options)), [data.series, options]);
  const accountOptions = useMemo(() => getAccountOptions(clusters), [clusters]);
  const clusterOptions = useMemo(
    () => getClusterOptions(clusters, filters.accounts),
    [clusters, filters.accounts],
  );
  const filtered = useMemo(() => filterClusters(clusters, filters), [clusters, filters]);

  // Selected clusters outside the selected accounts would hide everything, so they are dropped.
  const onAccountsChange = (accounts: string[]) => {
    const allowed = getClusterOptions(clusters, accounts).map(({ value }) => value);

    setAccounts(accounts);
    setClusters(filters.clusters.filter((cluster) => allowed.includes(cluster)));
  };

  if (!clusters.length) {
    return <div className={styles.empty}>{Messages.noData}</div>;
  }

  return (
    <div className={styles.wrapper}>
      <Toolbar
        search={search}
        onSearchChange={setSearch}
        accounts={filters.accounts}
        accountOptions={accountOptions}
        onAccountsChange={onAccountsChange}
        clusters={filters.clusters}
        clusterOptions={clusterOptions}
        onClustersChange={setClusters}
        shown={filtered.clusters.length}
        total={clusters.length}
      />
      {filtered.clusters.length ? (
        <ClusterTable
          clusters={filtered.clusters}
          autoExpanded={filtered.autoExpanded}
          expansionKey={filters.query}
          timeRange={timeRange}
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
