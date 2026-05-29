import { Button, Chip, Stack } from '@mui/material';
import FilterListIcon from '@mui/icons-material/FilterList';
import { FC, useMemo } from 'react';
import { useQanPanelActions, useQanPanelState } from '../hooks/useQanPanelState';
import { useQanFiltersDrawer } from '../hooks/useQanFiltersDrawer';
import {
  clearAllFilters,
  getActiveFilterChips,
  removeFilterChip,
} from '../utils/qanFilterChips';

export const QanControlsToolbar: FC = () => {
  const state = useQanPanelState();
  const actions = useQanPanelActions();
  const { toggle } = useQanFiltersDrawer();
  const chips = useMemo(() => getActiveFilterChips(state.labels), [state.labels]);

  return (
    <Stack
      direction="row"
      alignItems="center"
      spacing={1}
      sx={{ flexWrap: 'wrap', rowGap: 1 }}
      data-testid="qan-controls-toolbar"
    >
      <Button
        variant="outlined"
        size="small"
        startIcon={<FilterListIcon />}
        onClick={toggle}
        data-testid="qan-filters-button"
      >
        Filters
      </Button>
      {chips.map((chip) => (
        <Chip
          key={`${chip.key}-${chip.value}`}
          label={chip.label}
          size="small"
          onDelete={() => actions.setLabels(removeFilterChip(state.labels, chip.key, chip.value))}
        />
      ))}
      {chips.length ? (
        <Button
          size="small"
          onClick={() => actions.setLabels(clearAllFilters(state.labels))}
          data-testid="qan-clear-filters"
        >
          Clear all
        </Button>
      ) : null}
    </Stack>
  );
};
