import React, {
  FC,
  Fragment,
  useMemo,
  useState,
} from 'react';
import { Icon, IconButton, useStyles2 } from '@grafana/ui';
import { Messages } from './AuroraTreePanel.messages';
import { getStyles } from './AuroraTreePanel.styles';
import {
  compareSortValues,
  formatStorage,
  formatThroughput,
  getEngineLabel,
  getGroupType,
  getGroupTypeLabel,
  getWriters,
} from './AuroraTreePanel.tools';
import { GroupColumn, GroupSort, InstanceGroup } from './AuroraTreePanel.types';
import { InstancesTable } from './InstancesTable';

interface GroupTableProps {
  groups: InstanceGroup[];
  autoExpanded: Set<string>;
  // Rows the user expanded or collapsed are forgotten when this changes (the search text).
  expansionKey: string;
}

const GroupName: FC<{ group: InstanceGroup }> = ({ group }) => {
  const styles = useStyles2(getStyles);

  if (group.kind === 'unknown') {
    return (
      <span className={styles.unknownGroup} title={Messages.groupTypeTitles.unavailable}>{group.name}</span>
    );
  }

  return <span className={styles.groupName}>{group.name}</span>;
};

const Writers: FC<{ group: InstanceGroup }> = ({ group }) => {
  const styles = useStyles2(getStyles);
  const { text, state } = getWriters(group);
  const classNames = { good: styles.good, warning: styles.warning, unknown: styles.muted };

  return <span className={classNames[state]}>{text}</span>;
};

const COLUMNS: GroupColumn[] = [
  {
    id: 'region', header: Messages.columns.region, render: (g) => g.region, sortValue: (g) => g.region,
  },
  {
    id: 'awsAccountId',
    header: Messages.columns.awsAccount,
    render: (g) => g.awsAccountId,
    sortValue: (g) => g.awsAccountId,
  },
  {
    id: 'name',
    header: Messages.columns.identifier,
    render: (g) => <GroupName group={g} />,
    sortValue: (g) => g.name,
  },
  {
    id: 'type',
    header: Messages.columns.type,
    render: (g) => <span title={Messages.groupTypeTitles[getGroupType(g)]}>{getGroupTypeLabel(g)}</span>,
    sortValue: (g) => getGroupTypeLabel(g),
  },
  {
    id: 'engine',
    header: Messages.columns.engine,
    render: (g) => getEngineLabel(g.engine),
    sortValue: (g) => g.engine && getEngineLabel(g.engine),
  },
  {
    id: 'monitoredInstances',
    header: Messages.columns.monitoredInstances,
    render: (g) => g.monitoredInstances,
    sortValue: (g) => g.monitoredInstances,
  },
  {
    id: 'exportersConnected',
    header: Messages.columns.exportersConnected,
    render: (g) => g.exportersConnected,
    sortValue: (g) => g.exportersConnected,
  },
  {
    id: 'writers', header: Messages.columns.writers, render: (g) => <Writers group={g} />, sortValue: (g) => g.writers,
  },
  {
    id: 'throughput',
    header: Messages.columns.throughput,
    description: Messages.columns.throughputDescription,
    render: (g) => formatThroughput(g.throughput, g.engine),
    sortValue: (g) => g.throughput,
  },
  {
    id: 'storage',
    header: Messages.columns.storage,
    description: Messages.columns.storageDescription,
    render: (g) => formatStorage(g.storage),
    sortValue: (g) => g.storage?.bytes,
  },
];

// Clicking a header cycles ascending, descending, unsorted.
const nextSort = (sort: GroupSort | undefined, id: string): GroupSort | undefined => {
  if (sort?.id !== id) {
    return { id, desc: false };
  }

  return sort.desc ? undefined : { id, desc: true };
};

const getAriaSort = (sort: GroupSort | undefined, id: string) => {
  if (sort?.id !== id) {
    return undefined;
  }

  return sort.desc ? 'descending' : 'ascending';
};

export const GroupTable: FC<GroupTableProps> = ({
  groups, autoExpanded, expansionKey,
}) => {
  const styles = useStyles2(getStyles);
  const [sort, setSort] = useState<GroupSort>();
  const [expansion, setExpansion] = useState({ key: expansionKey, overrides: {} as Record<string, boolean> });
  const overrides = expansion.key === expansionKey ? expansion.overrides : {};
  const isExpanded = (group: InstanceGroup) => overrides[group.key] ?? autoExpanded.has(group.key);
  const allExpanded = groups.every(isExpanded);
  const setExpanded = (changed: InstanceGroup[], expanded: boolean) => setExpansion({
    key: expansionKey,
    overrides: changed.reduce((acc, group) => ({ ...acc, [group.key]: expanded }), overrides),
  });
  const sortedGroups = useMemo(() => {
    const column = COLUMNS.find(({ id }) => id === sort?.id);

    return column && sort
      ? [...groups].sort((a, b) => compareSortValues(column.sortValue(a), column.sortValue(b), sort.desc))
      : groups;
  }, [groups, sort]);

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
              onClick={() => setExpanded(groups, !allExpanded)}
            />
          </th>
          {COLUMNS.map(({ id, header, description }) => (
            <th key={id} className={styles.header} aria-sort={getAriaSort(sort, id)}>
              <button
                type="button"
                className={styles.sortButton}
                title={description}
                onClick={() => setSort(nextSort(sort, id))}
              >
                {header}
                {sort?.id === id && <Icon name={sort.desc ? 'angle-down' : 'angle-up'} />}
              </button>
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {sortedGroups.map((group) => {
          const expanded = isExpanded(group);
          const contentId = `rds-group-${group.key.replace(/[^\w-]/g, '-')}`;

          return (
            <Fragment key={group.key}>
              <tr className={styles.row}>
                <td className={`${styles.cell} ${styles.expanderCell}`}>
                  <IconButton
                    name={expanded ? 'angle-down' : 'angle-right'}
                    aria-label={expanded ? Messages.collapseRow : Messages.expandRow}
                    aria-expanded={expanded}
                    aria-controls={expanded ? contentId : undefined}
                    size="lg"
                    onClick={() => setExpanded([group], !expanded)}
                  />
                </td>
                {COLUMNS.map(({ id, render }) => (
                  <td key={id} className={styles.cell}>{render(group)}</td>
                ))}
              </tr>
              {expanded && (
                <tr id={contentId}>
                  {/* eslint-disable-next-line jsx-a11y/control-has-associated-label -- holds a table */}
                  <td className={styles.expandedCell} colSpan={COLUMNS.length + 1}>
                    <InstancesTable group={group} />
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
