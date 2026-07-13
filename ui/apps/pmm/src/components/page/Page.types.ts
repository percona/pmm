import { PropsWithChildren, ReactNode } from 'react';
import { OrgRole } from 'types/user.types';

export interface PageProps extends PropsWithChildren {
  title?: string;
  footer?: ReactNode;
  topBar?: ReactNode;
  /** Use full horizontal width instead of the default ~1000px centered column (e.g. ADRE chat). */
  fullWidth?: boolean;
  surface?: 'default' | 'paper';
  roles?: OrgRole[];
}
