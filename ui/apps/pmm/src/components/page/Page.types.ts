import { PropsWithChildren, ReactNode } from 'react';

export interface PageProps extends PropsWithChildren {
  title?: string;
  footer?: ReactNode;
  topBar?: ReactNode;
  /** Use full horizontal width instead of the default ~1000px centered column (e.g. ADRE chat). */
  fullWidth?: boolean;
}
