import { PickerValidDate } from '@mui/x-date-pickers';

export const toValidPickerDate = (value: Date): PickerValidDate | null =>
  Number.isNaN(value.getTime()) ? null : (value as PickerValidDate);
