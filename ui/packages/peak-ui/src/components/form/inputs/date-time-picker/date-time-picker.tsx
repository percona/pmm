import { Controller, FieldValues, useFormContext } from 'react-hook-form';

import { kebabize } from '@/utils';
import {
  DateTimePickerInputProps,
  DateTimePickerValueFormat,
  DateTimePickerValueTransform,
} from './date-time-picker.types';
import { DateTimePicker, PickerValidDate } from '@mui/x-date-pickers';

const toValidPickerDate = (value: Date): PickerValidDate | null =>
  Number.isNaN(value.getTime()) ? null : (value as PickerValidDate);

const presets: Record<DateTimePickerValueFormat, DateTimePickerValueTransform<PickerValidDate>> = {
  date: {
    input: (v) => (v == null ? null : (v as PickerValidDate)),
    output: (v) => v,
  },
  'iso-string': {
    input: (v) => {
      if (typeof v !== 'string' || !v) return null;
      return toValidPickerDate(new Date(v));
    },
    output: (v) =>
      v instanceof Date && !Number.isNaN(v.getTime()) ? v.toISOString() : null,
  },
  'unix-ms': {
    input: (v) => {
      if (typeof v !== 'number') return null;
      return toValidPickerDate(new Date(v));
    },
    output: (v) =>
      v instanceof Date && !Number.isNaN(v.getTime()) ? v.getTime() : null,
  },
};

const DateTimePickerInput = <
  TDate extends PickerValidDate,
  TFieldValues extends FieldValues = FieldValues,
>({
  name,
  control,
  controllerProps,
  valueFormat = 'date',
  transform,
  ...dateTimePickerProps
}: DateTimePickerInputProps<TDate, TFieldValues>) => {
  const formContext = useFormContext<TFieldValues>();
  const contextControl = formContext?.control;

  const { input, output } = (transform ?? presets[valueFormat]) as DateTimePickerValueTransform<TDate>;
  const { slotProps: consumerSlotProps, ...restPickerProps } = dateTimePickerProps;
  const consumerTextField =
    typeof consumerSlotProps?.textField === 'object' ? consumerSlotProps.textField : undefined;

  return (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field, fieldState: { error } }) => (
        <DateTimePicker
          {...restPickerProps}
          name={field.name}
          inputRef={field.ref}
          onBlur={field.onBlur}
          value={input(field.value) as TDate | null}
          onChange={(value) => field.onChange(output(value))}
          slotProps={{
            ...consumerSlotProps,
            textField: {
              ...consumerTextField,
              error: !!error,
              helperText: error ? error.message : '',
              inputProps: {
                ...consumerTextField?.inputProps,
                'data-testid': `date-time-picker-${kebabize(name)}`,
              },
            },
          }}
        />
      )}
      {...controllerProps}
    />
  );
};

export default DateTimePickerInput;
