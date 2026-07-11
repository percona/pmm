import { FC } from 'react';
import MenuItem from '@mui/material/MenuItem';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import { SelectInput } from '@percona/percona-ui';
import { Template } from 'types/alert-templates.types';
import { Messages } from '../CreateAlertFromTemplate.messages';

interface Props {
  templates: Template[];
}

export const TemplateSelectSection: FC<Props> = ({ templates }) => (
  <Stack gap={1}>
    <Typography variant="h6">{Messages.sections.template}</Typography>
    <SelectInput
      name="template"
      label={Messages.fields.template}
      isRequired
      selectFieldProps={{ displayEmpty: true }}
    >
      {templates.map((template) => (
        <MenuItem key={template.name} value={template.name}>
          {template.summary || template.name}
        </MenuItem>
      ))}
    </SelectInput>
  </Stack>
);

export default TemplateSelectSection;
