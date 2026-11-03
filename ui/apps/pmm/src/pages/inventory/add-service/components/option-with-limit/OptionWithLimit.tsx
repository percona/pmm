import { RadioGroup } from '@percona/peak-ui';
import type { FC } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import { FieldGrid } from '../field-grid';
import { TextField } from '../fields';
import type { OptionWithLimitProps } from './OptionWithLimit.types';

export const OptionWithLimit: FC<OptionWithLimitProps> = ({
  name,
  label,
  options,
  editableOption,
  limitName,
  limitLabel,
  limitTooltip,
}) => {
  const { control } = useFormContext();
  const option = useWatch({ control, name });

  return (
    <FieldGrid>
      <RadioGroup name={name} label={label} options={options} />
      {option === editableOption && (
        <TextField
          name={limitName}
          kind="number"
          label={limitLabel}
          tooltip={limitTooltip}
        />
      )}
    </FieldGrid>
  );
};
