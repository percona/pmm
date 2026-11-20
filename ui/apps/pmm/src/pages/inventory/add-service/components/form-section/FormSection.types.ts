import type { PropsWithChildren, ReactNode } from 'react';

export interface FormSectionProps extends PropsWithChildren {
  title: string;
  description?: ReactNode;
}
