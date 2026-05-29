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

export const QanManageColumns: FC = () => {
  const state = useQanPanelState();
  const actions = useQanPanelActions();
  const mainMetric = state.columns[0] ?? 'load';
  const { data } = useQanMetricNames(mainMetric, state.groupBy);

  const available = useMemo(() => {
    const names = data?.data?.map((m) => m.name) ?? DEFAULT_QAN_COLUMNS;
    return [...new Set([...DEFAULT_QAN_COLUMNS, ...names])];
  }, [data]);

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
        value={state.columns}
        onChange={onChange}
        input={<OutlinedInput label="Columns" />}
        renderValue={(selected) => selected.join(', ')}
      >
        {available.map((name) => (
          <MenuItem key={name} value={name}>
            <Checkbox checked={state.columns.includes(name)} />
            <ListItemText primary={name} />
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
};
