import { ButtonProps } from '@mui/material/Button';

export interface TextSelectOption<T> {
  label: string;
  value: T;
}

export interface TextSelectProps<T> {
  value: T;
  label?: string;
  options: TextSelectOption<T>[];
  onChange: (value: T) => void;
  disabled?: boolean;
  disabledValue?: string;
  startIcon?: ButtonProps['startIcon'];
  'data-testid-button'?: string;
}
