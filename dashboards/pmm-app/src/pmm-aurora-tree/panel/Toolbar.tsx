import React, { FC } from 'react';
import {
  ComboboxOption,
  FilterInput,
  InlineField,
  MultiCombobox,
  useStyles2,
} from '@grafana/ui';
import { Messages } from './AuroraTreePanel.messages';
import { getStyles } from './AuroraTreePanel.styles';

interface ToolbarProps {
  search: string;
  onSearchChange: (search: string) => void;
  accounts: string[];
  accountOptions: ComboboxOption[];
  onAccountsChange: (accounts: string[]) => void;
  clusters: string[];
  clusterOptions: ComboboxOption[];
  onClustersChange: (clusters: string[]) => void;
  shown: number;
  total: number;
}

const toValues = (options: ComboboxOption[]) => options.map(({ value }) => value);

export const Toolbar: FC<ToolbarProps> = ({
  search,
  onSearchChange,
  accounts,
  accountOptions,
  onAccountsChange,
  clusters,
  clusterOptions,
  onClustersChange,
  shown,
  total,
}) => {
  const styles = useStyles2(getStyles);

  return (
    <div className={styles.toolbar}>
      <div className={styles.search}>
        <FilterInput
          value={search}
          onChange={onSearchChange}
          escapeRegex={false}
          placeholder={Messages.searchPlaceholder}
          aria-label={Messages.searchPlaceholder}
        />
      </div>
      <InlineField label={Messages.columns.awsAccount}>
        <MultiCombobox
          id="aurora-tree-account"
          options={accountOptions}
          value={accounts}
          onChange={(options) => onAccountsChange(toValues(options))}
          placeholder={Messages.allAccounts}
          isClearable
          width="auto"
          minWidth={20}
          maxWidth={40}
        />
      </InlineField>
      <InlineField label={Messages.columns.cluster}>
        <MultiCombobox
          id="aurora-tree-cluster"
          options={clusterOptions}
          value={clusters}
          onChange={(options) => onClustersChange(toValues(options))}
          placeholder={Messages.allClusters}
          isClearable
          width="auto"
          minWidth={20}
          maxWidth={40}
        />
      </InlineField>
      <span className={styles.count}>{Messages.shownClusters(shown, total)}</span>
    </div>
  );
};
