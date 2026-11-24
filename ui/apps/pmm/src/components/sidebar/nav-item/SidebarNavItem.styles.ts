import { listItemTextClasses } from '@mui/material/ListItemText';
import { Theme } from '@mui/material/styles';
import { appearOnOpenMixin, drawerTransition } from '../drawer/Drawer.styles';

// The rail row's height: NavItem's 24px icon line.
const RAIL_LABEL_HEIGHT = 24;

export const getStyles = (theme: Theme, drawerOpen: boolean, level: number) => {
  const drawerTiming = drawerTransition(theme, drawerOpen);
  // A heading inside a group always shows, since a group opens only in the
  // open drawer; that also lets the group measure its full height.
  const showHeading = drawerOpen || level > 0;
  const sectionPaddingTop = 3;
  const sectionPaddingBottom = 1;
  // Gap between an expanded item's end and the edge of its list, at every
  // level; the group chevron and item actions sit at that same inset.
  const itemEnd = 0.5;
  const radius = theme.shape.borderRadiusXl;

  const trailingButtonPlace = {
    position: 'absolute',
    top: 0,
    right: theme.spacing(itemEnd),
  } as const;

  return {
    // The label keeps its line and is clipped by the item rather than
    // squeezed, fading out as the drawer narrows.
    navItem: {
      overflow: 'hidden',
      transition: [
        theme.transitions.create(
          ['padding', 'margin', 'border-radius'],
          drawerTiming
        ),
        theme.transitions.create('background-color', {
          duration: theme.transitions.duration.shortest,
        }),
      ].join(', '),

      [`.${listItemTextClasses.root}`]: {
        opacity: drawerOpen ? 1 : 0,
        transition: theme.transitions.create('opacity', drawerTiming),
      },

      // A two-line label eases down to the rail's 24px icon row and back, in
      // step with the drawer, instead of changing the item's height at once.
      [`.${listItemTextClasses.primary}`]: {
        maxHeight: drawerOpen ? '2lh' : RAIL_LABEL_HEIGHT,
        transition: theme.transitions.create('max-height', drawerTiming),
      },
    },
    itemEnd: drawerOpen ? { mr: itemEnd } : {},
    // Open, top-level items run flush left and round off on the right;
    // collapsed, they are square rows that centre the 24px icon.
    navItemRoot: drawerOpen
      ? { borderRadius: `0 ${radius}px ${radius}px 0` }
      : { borderRadius: 0, px: 1.5 },
    // Keeps the label clear of a button laid over the item's end: a group's
    // chevron or an item's action.
    navItemWithButton: drawerOpen ? { pr: 5.5 } : {},
    // Optically aligns a child's icon with its siblings' labels: the padding
    // matches NavItem's 8px icon-to-label gap.
    childIcon: drawerOpen ? { pl: 1 } : {},
    groupRow: {
      position: 'relative',
      overflow: 'hidden',
    },
    itemAction: {
      ...trailingButtonPlace,
      ...appearOnOpenMixin(theme),
    },
    // Fades out at once on collapse and in once the drawer has widened, so it
    // never passes over the icon; pinned to the first row when a label wraps.
    // Open, it inherits rather than forcing visible, so a toggle inside a closed
    // group stays hidden and out of the tab order.
    groupToggle: {
      position: 'absolute',
      top: 0,
      right: theme.spacing(itemEnd),
      opacity: drawerOpen ? 1 : 0,
      visibility: drawerOpen ? 'inherit' : 'hidden',
      transition: [
        theme.transitions.create(['opacity', 'visibility'], {
          duration: theme.transitions.duration.shorter,
          delay: drawerOpen ? drawerTiming.duration / 2 : 0,
        }),
        theme.transitions.create('background-color', {
          duration: theme.transitions.duration.shortest,
        }),
      ].join(', '),
    },
    listCollapsible:
      level === 0
        ? {
            pl: 4.75,
            pb: 2,
          }
        : level === 1
          ? {
              ml: 3.5,
              pl: 1,
              borderLeft: 1,
              borderColor: theme.palette.divider,
            }
          : {},
    listItemButton: {
      px: 2,
    },
    // Inside an open group a heading takes the group's indent; collapsed, the
    // row closes to a divider with 8px either side.
    sectionRow: {
      overflowX: 'clip',
      // Nested headings line up with the label of an icon-less NavItem: 16px
      // button padding, -14px spacer, 8px text padding.
      pl: level === 0 ? 2 : 1.25,
      pr: 1,
      pt: showHeading ? sectionPaddingTop : 1,
      pb: sectionPaddingBottom,
      minHeight: 17,
      gap: 0.5,
      transition: theme.transitions.create('padding', drawerTiming),
    },
    // Animating grid rows between 1fr and 0fr eases the label's height to and
    // from nothing, which a height of auto can't do.
    sectionLabel: {
      display: 'grid',
      gridTemplateRows: showHeading ? '1fr' : '0fr',
      minWidth: 0,
      opacity: showHeading ? 1 : 0,
      transition: theme.transitions.create(
        ['grid-template-rows', 'opacity'],
        drawerTiming
      ),
    },
    sectionLine: {
      position: 'absolute',
      top: theme.spacing(1),
      left: theme.spacing(1),
      right: theme.spacing(1),
      borderTop: 1,
      borderColor: 'divider',
      opacity: showHeading ? 0 : 1,
      transition: theme.transitions.create('opacity', drawerTiming),
    },
    headingEnd: {
      whiteSpace: 'nowrap',
    },
    // In the chip's own lettering, not the heading's.
    sectionBadge: {
      display: 'inline-flex',
      verticalAlign: 'middle',
      ml: 1,
      my: -0.25,
      textTransform: 'none',
      letterSpacing: 'normal',
    },
    // A heading with a badge wraps rather than lose its end under it, with
    // room between its lines for the badge.
    headingWraps: {
      [`.${listItemTextClasses.primary}`]: {
        whiteSpace: 'normal',
        lineHeight: '16px',
      },
    },
    sectionHeading: {
      m: 0,
      minHeight: 0,
      overflow: 'hidden',

      [`.${listItemTextClasses.primary}`]: {
        ...theme.typography.overline,
        fontSize: theme.typography.pxToRem(11),
        letterSpacing: '0.125em',
        color: theme.palette.text.secondary,
        whiteSpace: 'nowrap',
      },
    },
    // A heading inside the panel starts in line with the entries' labels.
    flyoutSection: {
      pl: 2,
    },
    // Already there when the panel opens, 8px in from its edge like a menu
    // row's action.
    flyoutAction: {
      ...trailingButtonPlace,
      right: theme.spacing(1),
    },
    textOnly: {
      m: 0,
      pl: 3,

      [`.${listItemTextClasses.primary}`]: {
        fontSize: 12,
        fontWeight: 500,
        color: theme.palette.text.secondary,
        fontFamily: theme.typography.body1.fontFamily,
      },

      [`.${listItemTextClasses.secondary}`]: {
        fontSize: 14,
        fontWeight: 475,
        color: theme.palette.text.secondary,
        fontFamily: 'Roboto Mono, monospace',
      },
    },
  };
};
