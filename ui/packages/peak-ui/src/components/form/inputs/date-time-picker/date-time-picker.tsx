import { Controller, useFormContext } from 'react-hook-form';

import { kebabize } from '@/utils';
import {
  DateTimePickerInputProps,
  DateTimePickerValueFormat,
  DateTimePickerValueTransform,
} from './date-time-picker.types';
import { DateTimePicker, PickerValidDate } from '@mui/x-date-pickers';

const presets: Record<DateTimePickerValueFormat, DateTimePickerValueTransform<PickerValidDate>> = {
  date: {
    input: (v) => v as PickerValidDate | null,
    output: (v) => v,
  },
  'iso-string': {
    input: (v) =>
      typeof v === 'string' && v ? (new Date(v) as PickerValidDate) : null,
    output: (v) =>
      v instanceof Date && !Number.isNaN(v.getTime()) ? v.toISOString() : null,
  },
  'unix-ms': {
    input: (v) =>
      typeof v === 'number' ? (new Date(v) as PickerValidDate) : null,
    output: (v) =>
      v instanceof Date && !Number.isNaN(v.getTime()) ? v.getTime() : null,
  },
};

const DateTimePickerInput = <T extends PickerValidDate>({
  name,
  control,
  controllerProps,
  valueFormat = 'date',
  transform,
  ...dateTimePickerProps
}: DateTimePickerInputProps<T>) => {
  const formContext = useFormContext();
  const contextControl = formContext?.control;

  const { input, output } = (transform ?? presets[valueFormat]) as DateTimePickerValueTransform<T>;

  return (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field, fieldState: { error } }) => (
        <DateTimePicker
          name={field.name}
          inputRef={field.ref}
          onBlur={field.onBlur}
          value={input(field.value) as T | null}
          onChange={(value) => field.onChange(output(value))}
          slotProps={{
            textField: {
              error: !!error,
              helperText: error ? error.message : '',
              inputProps: {
                'data-testid': `date-time-picker-${kebabize(name)}`,
              },
            },
          }}
          {...dateTimePickerProps}
        />
      )}
      {...controllerProps}
    />
  );
};

export default DateTimePickerInput;
