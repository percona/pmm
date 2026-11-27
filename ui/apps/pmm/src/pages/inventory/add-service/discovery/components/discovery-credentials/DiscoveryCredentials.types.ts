import type { ReactNode } from 'react';
import type { FieldValues, UseFormReturn } from 'react-hook-form';

export interface DiscoveryCredentialsProps<T extends FieldValues> {
  form: UseFormReturn<T>;
  onSubmit: (values: T) => void;
  isDiscovering: boolean;
  testId: string;
  children: ReactNode;
}
