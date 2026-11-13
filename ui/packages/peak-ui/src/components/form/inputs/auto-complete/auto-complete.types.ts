import { AutocompleteProps, TextFieldProps } from '@mui/material';
import {
  Control,
  FieldPath,
  FieldValues,
  UseControllerProps,
} from 'react-hook-form';

export type AutoCompleteInputProps<
  TOption,
  TFieldValues extends FieldValues = FieldValues,
> = {
  /** Form field the value is stored under; also drives the test ids. */
  name: FieldPath<TFieldValues>;
  /** Choices offered in the dropdown; the user can type to narrow them. */
  options: TOption[];
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control<TFieldValues>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<TFieldValues>, 'name' | 'control'>;
  /** Floating label shown inside the field (that moves to the top when the field is focused/filled). */
  label?: string;
  /** MUI Autocomplete props forwarded to the dropdown, such as multiple, freeSolo, or getOptionLabel. */
  autoCompleteProps?: Omit<
    AutocompleteProps<
      TOption,
      boolean | undefined,
      boolean | undefined,
      boolean | undefined
    >,
    'options' | 'renderInput'
  >;
  /** MUI TextField props forwarded to the text box, such as placeholder or helperText. */
  textFieldProps?: TextFieldProps;
  /** Shows a spinner inside the field while the choices are being fetched. */
  loading?: boolean;
  /** Adds an asterisk to the label and makes the browser block form submit while the text box is empty; rules go in `controllerProps.rules`. */
  isRequired?: boolean;
  /** Prevents the user from opening the dropdown or typing. */
  disabled?: boolean;
  /** Complementary text shown inside a tooltip that appears above the field on hover. */
  tooltipText?: string;
  /** Called after the user picks or clears a choice, once the form value is already updated. */
  onChange?: () => void;
};
