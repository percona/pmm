import { FormControlProps, SelectProps } from '@mui/material';
import {
  Control,
  FieldPath,
  FieldValues,
  UseControllerProps,
} from 'react-hook-form';

export type SelectInputProps<T extends FieldValues = FieldValues> = {
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control<T>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<T>, 'name' | 'control'>;
  /** Form field the selected value is stored under; also drives the test ids. */
  name: FieldPath<T>;
  /** Floating label shown inside the field or at the top, depending on state. */
  label?: string;
  /** Hint shown under the field. Validation errors only turn the field red, so pass the message here. */
  helperText?: React.ReactNode;
  /** MUI Select props forwarded to the dropdown, such as multiple or renderValue. */
  selectFieldProps?: SelectProps;
  /** MUI FormControl props for the wrapper, such as fullWidth or size; size is small unless this object sets the `size` key. */
  formControlProps?: FormControlProps;
  /** The choices, as MUI MenuItem elements; with none, a disabled "No options" item is shown. */
  children?: React.ReactNode | React.ReactNode[];
  /** Adds an asterisk to the label and makes the browser block form submit while nothing is selected; rules go in `controllerProps.rules`. */
  isRequired?: boolean;
  /** Replaces the dropdown arrow with a spinner while the choices are being fetched. */
  loading?: boolean;
};
