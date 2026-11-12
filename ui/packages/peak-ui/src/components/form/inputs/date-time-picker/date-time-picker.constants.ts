import { PickerValidDate } from '@mui/x-date-pickers';
import { DateTimePickerValueFormat, DateTimePickerValueTransform } from './date-time-picker.types';
import { toValidPickerDate } from './date-time-picker.utils';

export const presets: Record<
  DateTimePickerValueFormat,
  DateTimePickerValueTransform<PickerValidDate>
> = {
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
    output: (v) => (v instanceof Date && !Number.isNaN(v.getTime()) ? v.toISOString() : null),
  },
  'unix-ms': {
    input: (v) => {
      if (typeof v !== 'number') return null;
      return toValidPickerDate(new Date(v));
    },
    output: (v) => (v instanceof Date && !Number.isNaN(v.getTime()) ? v.getTime() : null),
  },
};
