import { FC } from 'react';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { SelectInput, TextInput } from '@percona/percona-ui';
import { SEVERITY_OPTIONS } from '../CreateAlertFromTemplate.constants';
import { Messages } from '../CreateAlertFromTemplate.messages';

export const DetailsSection: FC = () => (
  <Stack gap={2}>
    <Typography variant="h6">{Messages.sections.details}</Typography>
    <TextInput name="name" label={Messages.fields.name} isRequired />
    <SelectInput name="severity" label={Messages.fields.severity} isRequired>
      {SEVERITY_OPTIONS.map((option) => (
        <MenuItem key={option.value} value={option.value}>
          {option.label}
        </MenuItem>
      ))}
    </SelectInput>
    <TextInput
      name="duration"
      label={Messages.fields.duration}
      isRequired
      textFieldProps={{ type: 'number' }}
    />
  </Stack>
);

export default DetailsSection;
