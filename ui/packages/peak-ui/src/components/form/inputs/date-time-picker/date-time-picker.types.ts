import { Control, UseControllerProps } from 'react-hook-form';
import { DateTimePickerProps } from '@mui/x-date-pickers/DateTimePicker';
import { PickerValidDate } from '@mui/x-date-pickers';

export interface DateTimePickerInputProps<T extends PickerValidDate>
  extends DateTimePickerProps<T> {
  control?: Control;
  controllerProps?: UseControllerProps;
  name: string;
}
