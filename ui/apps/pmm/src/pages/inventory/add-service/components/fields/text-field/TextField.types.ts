import type { ReactNode } from 'react';

export interface TextFieldProps {
  name: string;
  label: string;
  tooltip?: ReactNode;
  placeholder?: string;
  helperText?: ReactNode;
  kind?: 'text' | 'number' | 'textarea';
  type?: string;
  endAdornment?: ReactNode;
  disabled?: boolean;
}
