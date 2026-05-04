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
    input: (v) => {
      if (v == null) return null;
      if (v instanceof Date) return toValidPickerDate(v);
      // Pass through non-Date adapter objects (Dayjs/Luxon/Moment).
      // Primitives like '' or numbers are rejected to avoid invalid-value warnings.
      if (typeof v === 'object') return v as PickerValidDate;
      return null;
    },
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
  const {
    slotProps: consumerSlotProps,
    onChange: consumerOnChange,
    ...restPickerProps
  } = dateTimePickerProps;

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const resolveConsumerTextField = (ownerState: any) => {
    const tf = consumerSlotProps?.textField;
    if (typeof tf === 'function') return tf(ownerState) ?? {};
    return tf ?? {};
  };

  return (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field, fieldState: { error } }) => (
        <DateTimePicker
          {...restPickerProps}
          name={field.name}
          inputRef={field.ref}
          value={input(field.value) as TDate | null}
          onChange={(value, context) => {
            field.onChange(output(value));
            consumerOnChange?.(value, context);
          }}
          slotProps={{
            ...consumerSlotProps,
            textField: (ownerState) => {
              const consumerTextField = resolveConsumerTextField(ownerState);
              return {
                ...consumerTextField,
                error: !!error,
                helperText: error ? error.message : '',
                onBlur: (event: React.FocusEvent<HTMLInputElement>) => {
                  consumerTextField.onBlur?.(event);
                  field.onBlur();
                },
                inputProps: {
                  ...consumerTextField.inputProps,
                  'data-testid': `date-time-picker-${kebabize(name)}`,
                },
              };
            },
          }}
        />
      )}
      {...controllerProps}
    />
  );
};

export default DateTimePickerInput;
