import React, { FC, useMemo } from 'react';
import { PanelProps } from '@grafana/data';
import { Button, useStyles2 } from '@grafana/ui';
import { Messages } from './AuroraTreePanel.messages';
import { getStyles } from './AuroraTreePanel.styles';
import {
  filterGroups,
  getAccountOptions,
  getEngineOptions,
  getGroupOptions,
  getRegionOptions,
  groupInstances,
  narrowSelection,
  readInstances,
} from './AuroraTreePanel.tools';
import { AuroraTreeOptions, GroupSelection } from './AuroraTreePanel.types';
import { GroupTable } from './GroupTable';
import { SelectFilter, Toolbar } from './Toolbar';
import { useGroupFilters } from './useGroupFilters';

export const AuroraTreePanel: FC<PanelProps<AuroraTreeOptions>> = ({ data, options }) => {
  const styles = useStyles2(getStyles);
  const {
    search, setSearch, filters, setSelection, clear,
  } = useGroupFilters();
  const groups = useMemo(() => groupInstances(readInstances(data.series, options)), [data.series, options]);
  const filtered = useMemo(() => filterGroups(groups, filters), [groups, filters]);
  const selects = useMemo((): SelectFilter[] => {
    const select = (changed: Partial<GroupSelection>) => setSelection(
      narrowSelection(groups, { ...filters, ...changed }),
    );

    return [
      {
        id: 'aurora-tree-region',
        label: Messages.columns.region,
        placeholder: Messages.allRegions,
        options: getRegionOptions(groups),
        value: filters.regions,
        onChange: (regions) => select({ regions }),
      },
      {
        id: 'aurora-tree-account',
        label: Messages.columns.awsAccount,
        placeholder: Messages.allAccounts,
        options: getAccountOptions(groups, filters.regions),
        value: filters.accounts,
        onChange: (accounts) => select({ accounts }),
      },
      {
        id: 'aurora-tree-engine',
        label: Messages.columns.engine,
        placeholder: Messages.allEngines,
        options: getEngineOptions(groups),
        value: filters.engines,
        onChange: (engines) => select({ engines }),
      },
      {
        id: 'aurora-tree-group',
        label: Messages.columns.group,
        placeholder: Messages.allGroups,
        options: getGroupOptions(groups, filters),
        value: filters.groups,
        onChange: (names) => select({ groups: names }),
      },
    ];
  }, [groups, filters, setSelection]);

  if (!groups.length) {
    return <div className={styles.empty}>{Messages.noData}</div>;
  }

  return (
    <div className={styles.wrapper}>
      <Toolbar
        search={search}
        onSearchChange={setSearch}
        selects={selects}
        shown={filtered.groups.length}
        total={groups.length}
      />
      {filtered.groups.length ? (
        <GroupTable
          groups={filtered.groups}
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
