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

export interface SelectFilter {
  id: string;
  label: string;
  placeholder: string;
  options: ComboboxOption[];
  value: string[];
  onChange: (value: string[]) => void;
}

interface ToolbarProps {
  search: string;
  onSearchChange: (search: string) => void;
  selects: SelectFilter[];
  shown: number;
  total: number;
}

export const Toolbar: FC<ToolbarProps> = ({
  search, onSearchChange, selects, shown, total,
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
      {selects.map(({
        id, label, placeholder, options, value, onChange,
      }) => (
        <InlineField key={id} label={label}>
          <MultiCombobox
            id={id}
            options={options}
            value={value}
            onChange={(selected) => onChange(selected.map((option) => option.value))}
            placeholder={placeholder}
            isClearable
            width="auto"
            minWidth={18}
            maxWidth={40}
          />
        </InlineField>
      ))}
      <span className={styles.count}>{Messages.shownGroups(shown, total)}</span>
    </div>
  );
};
