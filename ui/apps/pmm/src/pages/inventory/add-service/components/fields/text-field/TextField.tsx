import { TextInput } from '@percona/peak-ui';
import type { FC } from 'react';
import { FieldLabel } from '../field-label';
import type { TextFieldProps } from './TextField.types';

export const TextField: FC<TextFieldProps> = ({
  name,
  label,
  tooltip,
  placeholder,
  helperText,
  kind = 'text',
  type = kind === 'number' ? 'number' : undefined,
  autoComplete,
  endAdornment,
  disabled,
}) => (
  <TextInput
    name={name}
    textFieldProps={{
      // peak-ui's own label prop is a string; the tooltip needs a node.
      label: <FieldLabel label={label} tooltip={tooltip} />,
      type,
      autoComplete,
      placeholder,
      helperText,
      disabled,
      fullWidth: true,
      ...(kind === 'textarea' && { multiline: true, minRows: 3 }),
      ...(endAdornment && { slotProps: { input: { endAdornment } } }),
    }}
  />
);
