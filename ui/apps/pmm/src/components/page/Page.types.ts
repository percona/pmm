import { PropsWithChildren, ReactNode } from 'react';
import { OrgRole } from 'types/user.types';

export interface PageProps extends PropsWithChildren {
  title?: string;
  footer?: ReactNode;
  topBar?: ReactNode;
  fullWidth?: boolean;
  // removes the default max-width cap so the page spans the whole viewport
  wide?: boolean;
  surface?: 'default' | 'paper';
  roles?: OrgRole[];
}
