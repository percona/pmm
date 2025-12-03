export interface TextSelectOption<T> {
  label: string;
  value: T;
}

export interface TextSelectProps<T> {
  value: T;
  label?: string;
  options: TextSelectOption<T>[];
  onChange: (value: T) => void;
}
