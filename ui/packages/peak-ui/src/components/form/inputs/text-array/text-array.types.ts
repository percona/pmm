import { Control } from 'react-hook-form';

export type TextArrayProps = {
  fieldName: string;
  fieldKey: string;
  label?: string;
  placeholder?: string;
  control?: Control;
  handleBlur?: (value: string, fieldName: string, hasError: boolean) => void;
  onRemove?: (index: number) => void;
};
