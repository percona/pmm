import { forwardRef, type ComponentType } from 'react';
import Box from '@mui/material/Box';
import type { SvgIconProps } from '@mui/material/SvgIcon';
import Typography from '@mui/material/Typography';
import {
  NothingFoundIllustration,
  RadarIllustration,
  RealTimeDatabaseOffIllustration,
  UnknownErrorIllustration,
} from '../illustrations';
import type { EmptyStateIllustration, EmptyStateProps, EmptyStateSize } from './empty-state.types';

export const EMPTY_STATE_SIZES: Record<EmptyStateSize, { py: number; illustration: number }> = {
  small: { py: 4, illustration: 80 },
  medium: { py: 6, illustration: 120 },
};

export const EMPTY_STATE_ILLUSTRATIONS: Record<
  EmptyStateIllustration,
  ComponentType<SvgIconProps>
> = {
  'nothing-found': NothingFoundIllustration,
  radar: RadarIllustration,
  'unknown-error': UnknownErrorIllustration,
  'real-time-database-off': RealTimeDatabaseOffIllustration,
};

const EmptyState = forwardRef<HTMLDivElement, EmptyStateProps>(
  (
    { title, description, illustration, actions, titleComponent = 'h3', size = 'medium', ...rest },
    ref
  ) => {
    const { py, illustration: illustrationSize } = EMPTY_STATE_SIZES[size];
    const Illustration = illustration && EMPTY_STATE_ILLUSTRATIONS[illustration];

    return (
      <Box
        ref={ref}
        display="flex"
        flexDirection="column"
        alignItems="center"
        gap={1}
        py={py}
        px={2}
        textAlign="center"
        {...rest}
      >
        {Illustration && <Illustration aria-hidden sx={{ fontSize: illustrationSize }} />}
        <Typography variant="subtitle1" component={titleComponent}>
          {title}
        </Typography>
        {description && (
          <Typography variant="body2" component="div" color="text.secondary" sx={{ maxWidth: 520 }}>
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
