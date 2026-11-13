import { CSSObject, Theme, keyframes } from '@mui/material/styles';
import {
  DRAWER_CLOSE_DURATION,
  DRAWER_OPEN_DURATION,
  drawerWidth,
} from './Drawer.constants';

// The one timing every part of the sidebar moves with when it opens or
// closes, so nothing arrives before or after the drawer's own width.
export const drawerTransition = (theme: Theme, open: boolean) => ({
  easing: theme.transitions.easing.sharp,
  duration: open ? DRAWER_OPEN_DURATION : DRAWER_CLOSE_DURATION,
});

export const openedMixin = (theme: Theme): CSSObject => ({
  width: drawerWidth(true),
  transition: theme.transitions.create('width', drawerTransition(theme, true)),
  overflowX: 'hidden',
});

export const closedMixin = (theme: Theme): CSSObject => ({
  transition: theme.transitions.create('width', drawerTransition(theme, false)),
  overflowX: 'hidden',
  width: drawerWidth(false),
});

export const logoMixin = (theme: Theme, open: boolean): CSSObject => ({
  transition: theme.transitions.create(
    ['left', 'top', 'height', 'color'],
    drawerTransition(theme, open)
  ),
});

const appear = keyframes`
  from {
    opacity: 0;
  }
`;

// Controls that exist only while the drawer is open fade in once it has
// mostly widened, like the group chevrons, so they never pass over an icon.
export const appearOnOpenMixin = (theme: Theme) => ({
  animation: `${appear} ${theme.transitions.duration.shorter}ms ${
    DRAWER_OPEN_DURATION / 2
  }ms both`,
});

// Opening and closing still change the layout, just without the motion.
export const reducedMotionMixin: CSSObject = {
  '@media (prefers-reduced-motion: reduce)': {
    '&, & *': {
      transitionDuration: '0s !important',
      animationDuration: '0s !important',
      animationDelay: '0s !important',
    },
  },
};

// The list still scrolls by wheel, touchpad and keyboard, but shows no
// scrollbar, so nothing reserves space beside the items in either state.
export const hiddenScrollbarMixin: CSSObject = {
  scrollbarWidth: 'none',

  '&::-webkit-scrollbar': {
    display: 'none',
  },
};

// Fades the list's content out toward each edge that has more to scroll, so
// items dissolve into the sidebar background. A mask rather than an overlay,
// so highlighted items fade too.
export const scrollFadeMixin = (
  fadeTop: boolean,
  fadeBottom: boolean
): CSSObject => {
  const size = 48;
  // A faint midpoint keeps the edge mostly gone for longer, so the fade reads
  // clearly rather than as a soft blur.
  const fadeIn = `transparent, rgba(0, 0, 0, 0.1) ${size / 2}px, black ${size}px`;
  const fadeOut = `black calc(100% - ${size}px), rgba(0, 0, 0, 0.1) calc(100% - ${size / 2}px), transparent`;
  const mask = `linear-gradient(to bottom, ${fadeTop ? fadeIn : 'black'}, ${fadeBottom ? fadeOut : 'black'})`;

  return {
    maskImage: mask,
    WebkitMaskImage: mask,
    // Keyboard focus stops short of the faded edges, where the scroll buttons
    // also sit, so a focused item is never hidden.
    scrollPaddingBlock: `${size}px`,
  };
};
