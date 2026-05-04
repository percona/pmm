import { Control, UseControllerProps } from 'react-hook-form';
import { DateTimePickerProps, PickerValidDate } from '@mui/x-date-pickers';

export type DateTimePickerValueFormat = 'date' | 'iso-string' | 'unix-ms';

export interface DateTimePickerValueTransform<T> {
  /** Map the value stored in form state → the value the picker should receive. */
  input: (formValue: unknown) => T | null;
  /** Map the value the picker emits → the value to store in form state. */
  output: (pickerValue: T | null) => unknown;
}

export interface DateTimePickerInputProps<T extends PickerValidDate>
  extends DateTimePickerProps<T> {
  control?: Control;
  controllerProps?: UseControllerProps;
  name: string;
  onBlur?: React.FocusEventHandler<HTMLDivElement>;
  /**
   * Shortcut for common value shapes stored in form state.
   * - 'date'       → identity (form state holds a `Date`). **Default.**
   * - 'iso-string' → form state holds an ISO 8601 string (`"2025-01-01T12:00:00Z"`).
   * - 'unix-ms'    → form state holds epoch milliseconds.
   *
   * Presets assume the native `Date` adapter (`AdapterDateFns` / `AdapterDayjs`-as-Date).
   * For non-Date adapters (Luxon, Moment, Dayjs objects) supply `transform` instead.
   * Ignored when `transform` is supplied.
   */
  valueFormat?: DateTimePickerValueFormat;
  /** Custom transform pair. Takes precedence over `valueFormat`. */
  transform?: DateTimePickerValueTransform<T>;
}
