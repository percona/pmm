import {
  Control,
  FieldPath,
  FieldValues,
  UseControllerProps,
} from 'react-hook-form';
import { CheckboxProps as MUICheckboxProps } from '@mui/material';
import { LabeledContentProps } from '../../..';

export type CheckboxProps<T extends FieldValues = FieldValues> = {
  name: FieldPath<T>;
  label?: string;
  control?: Control<T>;
  controllerProps?: Omit<UseControllerProps<T>, 'name'>;
  checkboxProps?: MUICheckboxProps;
  labelProps?: LabeledContentProps;
  disabled?: boolean;
};
