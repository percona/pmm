import type { TextFieldProps } from '../text-field/TextField.types';

export type PasswordFieldProps = Pick<
  TextFieldProps,
  'name' | 'label' | 'tooltip' | 'placeholder'
>;
