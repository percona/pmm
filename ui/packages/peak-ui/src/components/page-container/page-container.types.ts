import type { StackProps } from '@mui/material/Stack';

/**
 * Width of a page container.
 * - a `number`: max container width in pixels (applied at the `lg` breakpoint and
 *   up). With MUI's default `box-sizing: border-box` the padding is included, so
 *   the content area is slightly narrower than this value.
 * - `'full'`: container stretches to 100% of the available width
 */
export type PageContainerMaxWidth = number | 'full';

export interface PageContainerProps extends Omit<StackProps, 'maxWidth'> {
  /**
   * Controls the max container width.
   * @default 1000
   */
  maxWidth?: PageContainerMaxWidth;
}
