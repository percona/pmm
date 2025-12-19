import { Control, UseControllerProps } from 'react-hook-form';
import { CheckboxProps as MUICheckboxProps } from '@mui/material';
import { LabeledContentProps } from '../../..';

export type CheckboxProps = {
  name: string;
  label?: string;
  control?: Control;
  controllerProps?: Omit<UseControllerProps, 'name'>;
  checkboxProps?: MUICheckboxProps;
  labelProps?: LabeledContentProps;
  disabled?: boolean;
};
