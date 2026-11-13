import { CSSObject, Theme, alpha, keyframes } from '@mui/material/styles';
import {
  DRAWER_BORDER_WIDTH,
  DRAWER_CLOSE_DURATION,
  DRAWER_CLOSED_WIDTH,
  DRAWER_OPEN_DURATION,
  DRAWER_WIDTH,
} from './Drawer.constants';

// The one timing every part of the sidebar moves with when it opens or
// closes, so nothing arrives before or after the drawer's own width.
export const drawerTransition = (theme: Theme, open: boolean) => ({
  easing: theme.transitions.easing.sharp,
  duration: open ? DRAWER_OPEN_DURATION : DRAWER_CLOSE_DURATION,
});

export const openedMixin = (theme: Theme): CSSObject => ({
  width: DRAWER_WIDTH + DRAWER_BORDER_WIDTH,
  transition: theme.transitions.create('width', drawerTransition(theme, true)),
  overflowX: 'hidden',
});

export const closedMixin = (theme: Theme): CSSObject => ({
  transition: theme.transitions.create('width', drawerTransition(theme, false)),
  overflowX: 'hidden',
  width: DRAWER_CLOSED_WIDTH + DRAWER_BORDER_WIDTH,
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

// Soft shadows at the list's top and bottom edges whenever there is more to
// scroll that way. The shadows are fixed to the edges, while covers in the
// paper colour scroll with the content and hide each shadow once its end is
// reached. Pure CSS: no scroll listeners.
export const scrollShadowMixin = (theme: Theme): CSSObject => {
  const paper = theme.palette.background.paper;
  // The covers are twice the shadow's height, fully opaque over the first
  // half, so a shadow is hidden completely before it fades in.
  const shadowSize = 24;
  const coverSize = shadowSize * 2;
  const shadow = alpha(
    theme.palette.common.black,
    theme.palette.mode === 'dark' ? 0.8 : 0.1
  );

  return {
    background: [
      `linear-gradient(${paper} 50%, transparent) top / 100% ${coverSize}px no-repeat local`,
      `linear-gradient(transparent, ${paper} 50%) bottom / 100% ${coverSize}px no-repeat local`,
      `linear-gradient(${shadow}, transparent) top / 100% ${shadowSize}px no-repeat scroll`,
      `linear-gradient(transparent, ${shadow}) bottom / 100% ${shadowSize}px no-repeat scroll`,
      paper,
    ].join(', '),
  };
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
