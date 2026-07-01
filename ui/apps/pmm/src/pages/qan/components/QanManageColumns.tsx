import {
  Checkbox,
  FormControl,
  InputLabel,
  ListItemText,
  MenuItem,
  OutlinedInput,
  Select,
  SelectChangeEvent,
} from '@mui/material';
import { FC, useMemo } from 'react';
import { useQanMetricNames } from 'hooks/api/useQan';
import { useQanPanelActions, useQanPanelState } from '../hooks/useQanPanelState';
import { DEFAULT_QAN_COLUMNS } from '../utils/qanTools';
import { metricNamesFromResponse } from '../utils/qanNormalize';

export const QanManageColumns: FC = () => {
  const state = useQanPanelState();
  const actions = useQanPanelActions();
  const mainMetric = state.columns[0] ?? 'load';
  const { data } = useQanMetricNames(mainMetric, state.groupBy);

  const available = useMemo(() => {
    const names = metricNamesFromResponse(data);
    return [...new Set([...DEFAULT_QAN_COLUMNS, ...names])];
  }, [data]);

  const selectedColumns = Array.isArray(state.columns) ? state.columns : DEFAULT_QAN_COLUMNS;

  const onChange = (e: SelectChangeEvent<string[]>) => {
    const value = e.target.value;
    const cols = typeof value === 'string' ? value.split(',') : value;
    if (cols.length) actions.setColumns(cols);
  };

  return (
    <FormControl size="small" sx={{ minWidth: 200 }}>
      <InputLabel>Columns</InputLabel>
      <Select
        multiple
        value={selectedColumns}
        onChange={onChange}
        input={<OutlinedInput label="Columns" />}
        renderValue={(selected) => selected.join(', ')}
      >
        {available.map((name) => (
          <MenuItem key={name} value={name}>
            <Checkbox checked={selectedColumns.includes(name)} />
            <ListItemText primary={name} />
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
};
