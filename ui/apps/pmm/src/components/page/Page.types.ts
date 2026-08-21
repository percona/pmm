import type { PropsWithChildren, ReactNode } from 'react';
import type { OrgRole } from 'types/user.types';

export interface PageProps extends PropsWithChildren {
  title?: string;
  footer?: ReactNode;
  topBar?: ReactNode;
  fullWidth?: boolean;
  surface?: 'default' | 'paper';
  roles?: OrgRole[];
}
