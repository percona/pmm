import { PropsWithChildren, ReactNode } from 'react';

export interface PageProps extends PropsWithChildren {
  title?: string;
  footer?: ReactNode;
  topBar?: ReactNode;
}
