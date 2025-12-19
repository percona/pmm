export type TextArrayProps = {
  fieldName: string;
  fieldKey: string;
  label?: string;
  placeholder?: string;
  handleBlur?: (value: string, fieldName: string, hasError: boolean) => void;
  onRemove?: (index: number) => void;
};
