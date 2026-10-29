import { listItemButtonClasses } from '@mui/material/ListItemButton';
import { Theme } from '@mui/material/styles';
import { DRAWER_WIDTH } from '../drawer/Drawer.constants';

export const getStyles = (theme: Theme) => {
  const quick = { duration: theme.transitions.duration.shorter };

  return {
    // The edit button stays out of the way: hidden until the section is
    // hovered or holds keyboard focus, half-strength then, and full strength
    // when it is hovered or focused itself. Focus left by a mouse click does
    // not count, so the button goes once the pointer leaves. Screens that
    // cannot hover show it at half.
    root: {
      '& [data-reveal]': {
        opacity: 0,
        transition: theme.transitions.create('opacity', quick),
      },
      '&:hover [data-reveal], &:has(:focus-visible) [data-reveal]': {
        opacity: 0.5,
      },
      '@media (hover: none)': {
        '& [data-reveal]': {
          opacity: 0.5,
        },
      },
      '& [data-reveal]:hover, & [data-reveal]:focus-visible': {
        opacity: 1,
      },
    },
    editing: {
      userSelect: 'none',

      '& [data-item-id]': {
        cursor: 'grab',
        touchAction: 'none',
        transition: theme.transitions.create(
          ['transform', 'box-shadow'],
          quick
        ),
      },

      // The row being dragged follows the pointer directly and lifts above the
      // others.
      '& [data-dragging]': {
        position: 'relative',
        zIndex: 1,
        cursor: 'grabbing',
        transition: 'none',

        [`.${listItemButtonClasses.root}`]: {
          backgroundColor: theme.palette.background.paper,
          boxShadow: theme.shadows[4],
        },
      },
    },
    // A dashed Paper as tall as its note, lined up with the heading. It keeps
    // the open drawer's width, so the drawer opening or closing over it
    // reveals or covers the note instead of rewrapping it.
    empty: {
      boxSizing: 'border-box',
      width: DRAWER_WIDTH,
      px: 2,
      py: 0.5,
    },
    emptyNote: {
      display: 'flex',
      alignItems: 'flex-start',
      gap: 1,
      // The dashed line's 2px, then 8px.
      p: '10px',
    },
    // The sidebar keeps its labels on one line while it animates; the note wraps.
    emptyText: {
      minWidth: 0,
      whiteSpace: 'normal',
      textWrap: 'pretty',
    },
    // Centred on the note's first 18px line.
    emptyStar: {
      flexShrink: 0,
      my: '-1px',
      color: 'text.secondary',
    },
    visuallyHidden: {
      position: 'absolute',
      width: 1,
      height: 1,
      m: '-1px',
      p: 0,
      border: 0,
      overflow: 'hidden',
      clip: 'rect(0 0 0 0)',
      whiteSpace: 'nowrap',
    },
  };
};
