import { Control, FieldPath, FieldValues, UseControllerProps } from 'react-hook-form';
import { CheckboxProps as MUICheckboxProps } from '@mui/material';
import { LabeledContentProps } from '../../..';

export type CheckboxProps<T extends FieldValues = FieldValues> = {
  /** Form field the checked state is stored under; also drives the test id. */
  name: FieldPath<T>;
  /** Heading shown above the checkbox; without it only the box renders. */
  label?: string;
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control<T>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<T>, 'name' | 'control'>;
  /** MUI Checkbox props forwarded to the box, such as size or color. */
  checkboxProps?: MUICheckboxProps;
  /** Settings for the heading rendered by `LabeledContent`, such as a caption or required asterisk. */
  labelProps?: LabeledContentProps;
  /** Prevents the user from toggling the box. */
  disabled?: boolean;
};
