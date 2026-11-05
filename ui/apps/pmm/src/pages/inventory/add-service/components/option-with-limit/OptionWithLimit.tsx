import MuiTextField from '@mui/material/TextField';
import { kebabize, RadioGroup } from '@percona/peak-ui';
import type { FC } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import { FieldGrid } from '../field-grid';
import { FieldLabel, TextField } from '../fields';
import type { OptionWithLimitProps } from './OptionWithLimit.types';

export const OptionWithLimit: FC<OptionWithLimitProps> = ({
  name,
  label,
  options,
  editableOption,
  limitName,
  limitLabel,
  limitTooltip,
  presets = {},
}) => {
  const { control } = useFormContext();
  const option = useWatch({ control, name });

  return (
    <FieldGrid>
      <RadioGroup name={name} label={label} options={options} />
      {option === editableOption ? (
        <TextField
          name={limitName}
          kind="number"
          label={limitLabel}
          tooltip={limitTooltip}
          required
        />
      ) : (
        // Same look as the peak-ui text field, but not part of the form.
        <MuiTextField
          label={<FieldLabel label={limitLabel} tooltip={limitTooltip} />}
          value={presets[option] ?? ''}
          disabled
          fullWidth
          size="small"
          variant="outlined"
          sx={{ mt: 3 }}
          slotProps={{
            inputLabel: { shrink: true },
            htmlInput: { 'data-testid': `text-input-${kebabize(limitName)}` },
          }}
        />
      )}
    </FieldGrid>
  );
};
