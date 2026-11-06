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
  // Flex basis in px: the narrowest width that still shows the label in full.
  width: number;
}

interface RdsToolbarProps {
  search: string;
  onSearchChange: (search: string) => void;
  selects: SelectFilter[];
  onSelectChange: (id: keyof GroupSelection, value: string[]) => void;
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
}) => (
  <Stack direction="row" flexWrap="wrap" alignItems="center" gap={1}>
    <TextField
      size="small"
      value={search}
      onChange={(event) => onSearchChange(event.target.value)}
      placeholder={Messages.search}
      sx={{ flex: '2 1 180px' }}
      slotProps={{
        input: {
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon fontSize="small" />
            </InputAdornment>
          ),
        },
        htmlInput: { 'aria-label': Messages.searchLabel },
      }}
    />
    {selects.map(({ id, label, placeholder, options, value, width }) => (
      <Autocomplete
        key={id}
        id={`rds-filter-${id}`}
        multiple
        size="small"
        disableCloseOnSelect
        disableClearable
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
        // Text, not chips: a selection never makes the toolbar wrap.
        renderValue={(selected) => (
          <Typography
            variant="body2"
            noWrap
            title={selected.map((option) => option.label).join(', ')}
          >
            {Messages.selectedValues(selected[0].label, selected.length - 1)}
          </Typography>
        )}
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
        sx={{
          flex: `1 1 ${width}px`,
          minWidth: 0,
          '& .MuiAutocomplete-inputRoot': { flexWrap: 'nowrap' },
          // The typing area takes room only while the filter is open.
          '&:not(.Mui-focused) .MuiAutocomplete-input': { minWidth: 0 },
        }}
      />
    ))}
  </Stack>
);
