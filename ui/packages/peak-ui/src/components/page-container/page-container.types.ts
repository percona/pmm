import type { StackProps } from '@mui/material/Stack';

/**
 * Content width of a page.
 * - a `number`: max content width in pixels (applied at the `lg` breakpoint and up)
 * - `'full'`: content stretches to 100% of the available width
 */
export type PageContainerMaxWidth = number | 'full';

export interface PageContainerProps extends Omit<StackProps, 'maxWidth'> {
  /**
   * Controls the max content width.
   * @default 1000
   */
  maxWidth?: PageContainerMaxWidth;
}
