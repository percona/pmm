import { Control } from 'react-hook-form';

export type TextArrayProps = {
  /** Form field holding the list, as an array of rows. */
  fieldName: string;
  /** Property of each row that holds the text, so a row is `{ [fieldKey]: string }`. */
  fieldKey: string;
  /** Heading shown above the list, next to the "Add new" button. */
  label?: string;
  /** Placeholder shown in every empty text field. */
  placeholder?: string;
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control;
  /** Called when a field loses focus, with its text, its form path, and whether it currently has an error. */
  handleBlur?: (value: string, fieldName: string, hasError: boolean) => void;
  /** Called with the row's index after the user deletes it. */
  onRemove?: (index: number) => void;
};
