import { FormHelperTextProps, TextFieldProps } from '@mui/material';
import {
  Control,
  FieldPath,
  FieldValues,
  UseControllerProps,
} from 'react-hook-form';

export type TextInputProps<T extends FieldValues = FieldValues> = {
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control<T>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<T>, 'name' | 'control'>;
  /** Form field the text is stored under; also drives the test id. */
  name: FieldPath<T>;
  /** Label shown on the field's top border; it stays there even while the field is empty. */
  label?: string;
  /** MUI TextField props such as placeholder or multiline; an onBlur here must return the text to store, as its result replaces the value. */
  textFieldProps?: TextFieldProps;
  /** Adds an asterisk to the label and makes the browser block form submit while the field is empty; rules go in `controllerProps.rules`. */
  isRequired?: boolean;
  /** Styles for the helper or error text under the field; only `sx` is applied. */
  formHelperTextProps?: Partial<FormHelperTextProps>;
};
