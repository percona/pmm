import {
  Box,
  Checkbox,
  FormControlLabel,
  Stack,
  Typography,
} from '@mui/material';
import { FC, useMemo } from 'react';
import { useQanFilters } from 'hooks/api/useQan';
import { useQanPanelActions, useQanPanelState } from '../hooks/useQanPanelState';
import { getLabelQueryParams } from '../utils/qanTools';
import { asLabelValueList, asStringList } from '../utils/qanNormalize';
import type { QanLabelsMap } from 'types/qan.types';

export const QanFiltersPanel: FC = () => {
  const state = useQanPanelState();
  const actions = useQanPanelActions();
  const mainMetric = state.columns[0] ?? 'load';

  const filterParams = useMemo(
    () => ({
      labels: getLabelQueryParams(state.labels),
      mainMetricName: mainMetric,
      periodStartFrom: state.from,
      periodStartTo: state.to,
    }),
    [state.labels, mainMetric, state.from, state.to]
  );

  const { data, isLoading } = useQanFilters(filterParams);

  const toggleLabel = (key: string, value: string, checked: boolean) => {
    const next: QanLabelsMap = { ...state.labels };
    const current = asStringList(next[key]);
    if (checked) {
      next[key] = [...new Set([...current.filter((v) => v !== '$__all'), value])];
    } else {
      const filtered = current.filter((v) => v !== value);
      next[key] = filtered.length ? filtered : ['$__all'];
    }
    actions.setLabels(next);
  };

  return (
    <Box sx={{ overflow: 'auto', maxHeight: '100%', py: 1, px: 2 }} data-testid="qan-filters-panel">
      <Typography variant="subtitle2" sx={{ mb: 1 }}>
        Filters
      </Typography>
      {isLoading ? (
        <Typography variant="body2" color="text.secondary">
          Loading…
        </Typography>
      ) : null}
      {data?.labels
        ? Object.entries(data.labels).map(([key, group]) => (
            <Box key={key} sx={{ mb: 2 }}>
              <Typography variant="caption" color="text.secondary">
                {key}
              </Typography>
              <Stack>
                {asLabelValueList(group.name).slice(0, 20).map((item) => {
                  const val = item.value ?? '';
                  const selected = asStringList(state.labels[key]).includes(val);
                  return (
                    <FormControlLabel
                      key={`${key}-${val}`}
                      control={
                        <Checkbox
                          size="small"
                          checked={selected}
                          onChange={(_, c) => toggleLabel(key, val, c)}
                        />
                      }
                      label={
                        <Typography variant="body2" noWrap title={val}>
                          {val || '(empty)'}
                        </Typography>
                      }
                    />
                  );
                })}
              </Stack>
            </Box>
          ))
        : null}
    </Box>
  );
};
