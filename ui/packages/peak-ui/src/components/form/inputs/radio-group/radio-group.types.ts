import {
  RadioGroupProps as MuiRadioGroupProps,
  RadioProps,
} from '@mui/material';
import {
  Control,
  FieldPath,
  FieldValues,
  UseControllerProps,
} from 'react-hook-form';
import { LabeledContentProps } from '../../../labeled-content';

export type RadioGroupOptions = {
  /** Text shown right after the radio. */
  label: string;
  /** Value of this option; the form stores it as text (`1` becomes `"1"`), so prefer string values. */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  value: any;
  /** Prevents the user from picking this option while keeping its state. */
  disabled?: boolean;
  /** MUI Radio props forwarded to this option's radio, such as size or color. */
  radioProps?: RadioProps;
};

export type RadioGroupProps<T extends FieldValues = FieldValues> = {
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control<T>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<T>, 'name' | 'control'>;
  /** Form field the picked value is stored under; each radio's test id comes from its option's `value`. */
  name: FieldPath<T>;
  /** Heading shown above the options; without it only the radios render. */
  label?: string;
  /** Settings for the heading rendered by `LabeledContent`, such as a caption. */
  labelProps?: LabeledContentProps;
  /** MUI RadioGroup props; pass `{ row: false }` to stack the options vertically. */
  radioGroupFieldProps?: MuiRadioGroupProps;
  /** Adds a required asterisk to the heading; validation itself goes in `controllerProps.rules`. */
  isRequired?: boolean;
  /** Choices to show, one radio each. */
  options: RadioGroupOptions[];
};
