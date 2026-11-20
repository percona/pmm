import type { ReactNode } from 'react';

export interface TextFieldProps {
  name: string;
  label: string;
  tooltip?: ReactNode;
  placeholder?: string;
  helperText?: ReactNode;
  kind?: 'text' | 'number' | 'textarea';
  type?: string;
  autoComplete?: string;
  endAdornment?: ReactNode;
  disabled?: boolean;
  required?: boolean;
  // Replaces peak-ui's text-input-<name>, for a name that doesn't make a readable id.
  testId?: string;
}
