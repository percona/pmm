import type { Theme } from '@mui/material/styles';
import { visuallyHidden as hidden } from 'components/preview-chip/PreviewChip.styles';

// Out of flow it is a block in browsers anyway; saying so makes every name computation space it off.
export const visuallyHidden = { ...hidden, display: 'block' } as const;

// Text that takes focus only so its tooltip is reachable from the keyboard.
export const focusableText = (theme: Theme) => ({
  borderRadius: `${theme.shape.borderRadiusXs}px`,
  '&:focus-visible': {
    outline: `2px solid ${theme.palette.primary.main}`,
    outlineOffset: 2,
  },
});

/** The hub page's container name, for layouts that follow its width rather than the viewport's. */
export const HUB_CONTAINER = 'hub';
