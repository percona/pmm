import { type BoxProps } from '@mui/material/Box';

export type StatusIconSeverity =
  | 'success'
  | 'info'
  | 'indeterminate'
  | 'warning'
  | 'error';

export type StatusIconSize = 'medium' | 'small';

export interface StatusIconProps extends Omit<BoxProps, 'color' | 'children'> {
  /** Semantic status. Determines both the color and the (locked) glyph. */
  severity: StatusIconSeverity;
  /** medium = 32px (symbol inside a tinted circular wrapper), small = 22px (bare symbol). */
  size?: StatusIconSize;
  /** Elevation surface the icon sits on; backs the small mask and medium base so translucent surfaces composite. Defaults to theme paper. */
  surfaceColor?: string;
  /** Accessible name. When set, the icon is exposed as an image; otherwise it stays decorative (aria-hidden). */
  titleAccess?: string;
}
