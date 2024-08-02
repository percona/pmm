import {
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
} from '@mui/material';
import { FC } from 'react';
import { VersionsFilter } from '../UpdateClients.types';
import { ClientsFilterProps } from './ClientsFilter.types';
import { Messages } from '../UpdateClients.messages';

export const ClientsFilter: FC<ClientsFilterProps> = ({ value, onChange }) => (
  <Stack sx={{ width: 175 }}>
    <FormControl fullWidth>
      <InputLabel>{Messages.filter.label}</InputLabel>
      <Select
        label={Messages.filter.label}
        value={value}
        onChange={(e) => onChange(e.target.value as VersionsFilter)}
      >
        <MenuItem value={VersionsFilter.All}>{Messages.filter.all}</MenuItem>
        <MenuItem value={VersionsFilter.Required}>
          {Messages.filter.update}
        </MenuItem>
        <MenuItem value={VersionsFilter.Critical}>
          {Messages.filter.critical}
        </MenuItem>
      </Select>
    </FormControl>
  </Stack>
);
