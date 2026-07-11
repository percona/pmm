import { FC } from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import TextField from '@mui/material/TextField';
import { Template } from 'types/alert-templates.types';
import { Messages } from '../CreateAlertFromTemplate.messages';

interface Props {
  template: Template | null;
}

export const AdvancedSection: FC<Props> = ({ template }) => {
  if (!template) {
    return null;
  }

  return (
    <Stack gap={1}>
      <Typography variant="h6">{Messages.sections.advanced}</Typography>
      <TextField
        label={Messages.fields.expr}
        value={template.expr}
        multiline
        minRows={3}
        InputProps={{ readOnly: true }}
        data-testid="rule-expr"
      />
    </Stack>
  );
};

export default AdvancedSection;
