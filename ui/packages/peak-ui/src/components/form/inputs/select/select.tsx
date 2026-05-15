import {
  CircularProgress,
  FormControl,
  FormHelperText,
  InputLabel,
  MenuItem,
  Select,
} from '@mui/material';
import { kebabize } from '@/utils';
import { Controller, FieldValues, useFormContext } from 'react-hook-form';
import { SelectInputProps } from './select.types';
import { Messages } from './select.messages';

const SelectInput = <T extends FieldValues = FieldValues>({
  name,
  control,
  label,
  helperText,
  controllerProps,
  selectFieldProps,
  formControlProps,
  loading,
  children,
}: SelectInputProps<T>) => {
  const formContext = useFormContext<T>();
  const contextControl = formContext?.control;

  return (
    <FormControl sx={{ mt: 3 }} size={formControlProps?.size || 'small'} {...formControlProps}>
      <InputLabel id={`${name}-input-label`}>{label}</InputLabel>
      <Controller
        name={name}
        control={control ?? contextControl}
        render={({ field, fieldState: { error } }) => (
          <Select
            {...field}
            label={label}
            labelId={`${name}-input-label`}
            variant="outlined"
            error={error !== undefined}
            data-testid={`select-${kebabize(name)}-button`}
            inputProps={{
              'data-testid': `select-input-${kebabize(name)}`,
              ...selectFieldProps?.inputProps,
            }}
            IconComponent={
              loading
                ? () => <CircularProgress color="inherit" size={20} sx={{ mr: 1 }} />
                : undefined
            }
            {...selectFieldProps}
          >
            {children}
            {(!children || (Array.isArray(children) && !children.length)) && (
              <MenuItem
                disabled
                key="noOptions"
                value=""
                data-testid="no-options-select"
                sx={{
                  fontWeight: '400',
                  '&.Mui-disabled.Mui-selected': {
                    backgroundColor: 'transparent',
                  },
                }}
              >
                {Messages.noOptions}
              </MenuItem>
            )}
          </Select>
        )}
        {...controllerProps}
      />
      {helperText && <FormHelperText>{helperText}</FormHelperText>}
    </FormControl>
  );
};

export default SelectInput;
