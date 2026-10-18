import { PropsWithChildren, ReactNode } from 'react';
import { OrgRole } from 'types/user.types';

export interface PageProps extends PropsWithChildren {
  title?: string;
  /** Sets the document title without showing the page heading. */
  hideTitle?: boolean;
  footer?: ReactNode;
  topBar?: ReactNode;
  /** Use full horizontal width instead of the default ~1000px centered column (e.g. ADRE chat). */
  fullWidth?: boolean;
  /** Drop the ~1000px column limit but keep the normal page layout (padding, scrolling, footer); for wide tables. */
  wide?: boolean;
  surface?: 'default' | 'paper';
  roles?: OrgRole[];
}
