import type { FC } from 'react';
import Autocomplete from '@mui/material/Autocomplete';
import Box from '@mui/material/Box';
import Checkbox from '@mui/material/Checkbox';
import InputAdornment from '@mui/material/InputAdornment';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import SearchIcon from '@mui/icons-material/Search';
import { Messages } from '../AmazonRds.messages';
import type { FilterOption, GroupSelection } from '../AmazonRds.types';

export interface SelectFilter {
  id: keyof GroupSelection;
  label: string;
  placeholder: string;
  options: FilterOption[];
  value: string[];
}

interface RdsToolbarProps {
  search: string;
  onSearchChange: (search: string) => void;
  selects: SelectFilter[];
  onSelectChange: (id: keyof GroupSelection, value: string[]) => void;
  shown: number;
  total: number;
}

// A selected value the data no longer offers still shows, under its own name.
const toSelected = (options: FilterOption[], value: string[]) =>
  value.map(
    (item) =>
      options.find((option) => option.value === item) ?? {
        label: item,
        value: item,
      }
  );

export const RdsToolbar: FC<RdsToolbarProps> = ({
  search,
  onSearchChange,
  selects,
  onSelectChange,
  shown,
  total,
}) => (
  <Stack direction="row" flexWrap="wrap" alignItems="center" gap={1.5}>
    <TextField
      size="small"
      value={search}
      onChange={(event) => onSearchChange(event.target.value)}
      placeholder={Messages.searchPlaceholder}
      sx={{ flex: '1 1 320px', maxWidth: 420 }}
      slotProps={{
        input: {
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon fontSize="small" />
            </InputAdornment>
          ),
        },
        htmlInput: { 'aria-label': Messages.searchPlaceholder },
      }}
    />
    {selects.map(({ id, label, placeholder, options, value }) => (
      <Autocomplete
        key={id}
        id={`rds-filter-${id}`}
        multiple
        size="small"
        disableCloseOnSelect
        limitTags={1}
        options={options}
        value={toSelected(options, value)}
        onChange={(_, selected) =>
          onSelectChange(
            id,
            selected.map((option) => option.value)
          )
        }
        getOptionLabel={(option) => option.label}
        isOptionEqualToValue={(option, selected) =>
          option.value === selected.value
        }
        renderOption={({ key, ...props }, option, { selected }) => (
          <Box component="li" key={key} {...props}>
            <Checkbox size="small" checked={selected} sx={{ mr: 1, p: 0 }} />
            <ListItemText
              primary={option.label}
              secondary={option.description}
            />
          </Box>
        )}
        renderInput={(params) => (
          <TextField
            {...params}
            label={label}
            placeholder={value.length ? undefined : placeholder}
          />
        )}
        sx={{ flex: '0 1 220px', minWidth: 180 }}
      />
    ))}
    <Typography variant="body2" color="text.secondary" sx={{ ml: 'auto' }}>
      {Messages.shownGroups(shown, total)}
    </Typography>
  </Stack>
);
