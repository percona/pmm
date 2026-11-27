import {
  FormControlLabelProps as MuiFormControlLabelProps,
  SwitchProps,
} from '@mui/material';
import {
  Control,
  FieldPath,
  FieldValues,
  UseControllerProps,
} from 'react-hook-form';

type FormControlLabelProps = MuiFormControlLabelProps;

export type SwitchInputProps<T extends FieldValues = FieldValues> = {
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control<T>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<T>, 'name' | 'control'>;
  /** MUI FormControlLabel props for the label row, such as labelPlacement. */
  formControlLabelProps?: Omit<FormControlLabelProps, 'control' | 'label'>;
  /** Form field the on/off state is stored under; also drives the test ids. */
  name: FieldPath<T>;
  /** Text shown right after the switch. */
  label: string;
  /** Smaller explanatory text, similar to the helper text in other inputs, shown under the label. */
  labelCaption?: string;
  /** MUI Switch props forwarded to the toggle, such as size or color; its onChange fires too. */
  switchFieldProps?: SwitchProps;
};
