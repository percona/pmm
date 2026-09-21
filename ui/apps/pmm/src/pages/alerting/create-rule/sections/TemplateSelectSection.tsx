import { FC } from 'react';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import { Template } from 'types/alert-templates.types';
import { Messages } from '../CreateAlertFromTemplate.messages';
import { Controller, useFormContext } from 'react-hook-form';
import Autocomplete from '@mui/material/Autocomplete';
import TextField from '@mui/material/TextField';

interface Props {
  selectedTemplate: Template | null;
  templates: Template[];
}

export const TemplateSelectSection: FC<Props> = ({
  selectedTemplate,
  templates,
}) => {
  const { control } = useFormContext();

  return (
    <Stack gap={1}>
      <Typography variant="h6">{Messages.sections.template}</Typography>
      <Controller
        name="template"
        control={control}
        render={({ field }) => (
          <Autocomplete
            options={templates}
            value={selectedTemplate}
            getOptionLabel={(option) => option.name}
            getOptionKey={(option) => option.name}
            onChange={(_, newValue) => {
              field.onChange(newValue?.name);
            }}
            renderInput={(params) => (
              <TextField
                {...params}
                label={Messages.fields.template}
                size="small"
                variant="outlined"
              />
            )}
            sx={{
              mt: 2,
            }}
          />
        )}
      />
    </Stack>
  );
};

export default TemplateSelectSection;
