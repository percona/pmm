import { forwardRef } from 'react';
import Box from '@mui/material/Box';
import { svgIconClasses } from '@mui/material/SvgIcon';
import Typography from '@mui/material/Typography';
import { mergeSx } from '@/utils';
import type { EmptyStateProps } from './empty-state.types';

const SIZES = {
  small: { py: 4, illustration: 48 },
  medium: { py: 6, illustration: 64 },
} as const;

const EmptyState = forwardRef<HTMLDivElement, EmptyStateProps>(
  (
    {
      title,
      description,
      illustration,
      actions,
      titleComponent = 'h3',
      size = 'medium',
      sx,
      ...rest
    },
    ref
  ) => {
    const { py, illustration: illustrationSize } = SIZES[size];

    return (
      <Box
        ref={ref}
        sx={mergeSx(
          {
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 1,
            py,
            px: 2,
            textAlign: 'center',
          },
          sx
        )}
        {...rest}
      >
        {illustration && (
          <Box
            aria-hidden
            sx={{
              display: 'flex',
              [`& > .${svgIconClasses.root}`]: { fontSize: illustrationSize },
            }}
          >
            {illustration}
          </Box>
        )}
        <Typography variant="subtitle1" component={titleComponent}>
          {title}
        </Typography>
        {description && (
          <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 520 }}>
            {description}
          </Typography>
        )}
        {actions && (
          <Box
            sx={{
              display: 'flex',
              flexWrap: 'wrap',
              justifyContent: 'center',
              gap: 1,
              mt: 1,
            }}
          >
            {actions}
          </Box>
        )}
      </Box>
    );
  }
);

EmptyState.displayName = 'EmptyState';

export default EmptyState;
