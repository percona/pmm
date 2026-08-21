import { PropsWithChildren, ReactNode } from 'react';
import type { PageContainerMaxWidth } from '@percona/peak-ui';
import { OrgRole } from 'types/user.types';

export interface PageProps extends PropsWithChildren {
  title?: string;
  footer?: ReactNode;
  topBar?: ReactNode;
  /**
   * Max content width: a pixel number, or `'full'` for 100% width.
   * @default 1000
   */
  maxWidth?: PageContainerMaxWidth;
  /**
   * @deprecated Use `maxWidth="full"` instead. Kept as an alias that maps to
   * `maxWidth="full"` when `maxWidth` is not set.
   */
  fullWidth?: boolean;
  surface?: 'default' | 'paper';
  roles?: OrgRole[];
}
