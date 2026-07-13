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
      spacing={2}
      sx={{ flexWrap: 'wrap', rowGap: 1, flex: 1, minWidth: 0 }}
      data-testid="qan-controls-toolbar"
    >
      <Button
        variant="text"
        color="primary"
        size="small"
        startIcon={<FilterListIcon sx={{ fontSize: 18 }} />}
        onClick={toggle}
        sx={{
          fontWeight: 600,
          fontSize: 15,
          px: 1.25,
          py: 1,
          flexShrink: 0,
        }}
        data-testid="qan-filters-button"
      >
        Filters
      </Button>
      <Stack direction="row" alignItems="center" spacing={0.5} sx={{ flexWrap: 'wrap' }}>
        {chips.map((chip) => (
          <Chip
            key={`${chip.key}-${chip.value}`}
            label={chip.label}
            size="medium"
            onDelete={() =>
              actions.setLabels(removeFilterChip(state.labels, chip.key, chip.value))
            }
            sx={{
              borderRadius: '100px',
              bgcolor: 'action.selected',
              height: 'auto',
              py: 0.5,
              '& .MuiChip-label': { fontSize: 16, px: 0.5 },
            }}
          />
        ))}
        {chips.length ? (
          <Button
            variant="text"
            color="primary"
            size="small"
            onClick={() => actions.setLabels(clearAllFilters(state.labels))}
            sx={{ fontWeight: 600, fontSize: 13, minWidth: 'auto', px: 1 }}
            data-testid="qan-clear-filters"
          >
            Clear all
          </Button>
        ) : null}
      </Stack>
    </Stack>
  );
};
