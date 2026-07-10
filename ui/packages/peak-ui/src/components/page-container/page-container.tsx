import { forwardRef } from 'react';
import Stack from '@mui/material/Stack';
import type { PageContainerProps } from './page-container.types';

/** Default max content width in pixels. */
export const DEFAULT_PAGE_MAX_WIDTH = 1000;

/**
 * Centered page content container.
 *
 * Constrains its children to a max width and centers them horizontally.
 * On screens below `lg` the content is always full width (only side padding
 * applies); the `maxWidth` cap kicks in from `lg` up.
 *
 * Pass `maxWidth="full"` to let content stretch edge to edge (with side
 * padding), or a pixel number to cap it.
 */
const PageContainer = forwardRef<HTMLDivElement, PageContainerProps>(
  ({ maxWidth = DEFAULT_PAGE_MAX_WIDTH, sx, children, ...rest }, ref) => {
    const isFull = maxWidth === 'full';

    return (
      <Stack
        ref={ref}
        sx={{
          flex: 1,
          width: '100%',
          maxWidth: isFull ? '100%' : { lg: maxWidth },
          p: {
            xs: 2,
          },
          px: {
            md: isFull ? 4 : undefined,
          },
          mx: 'auto',
          gap: 2,
          mt: 1,
          ...sx,
        }}
        {...rest}
      >
        {children}
      </Stack>
    );
  }
);

PageContainer.displayName = 'PageContainer';

export default PageContainer;
