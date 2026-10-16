import { listItemTextClasses } from '@mui/material/ListItemText';
import { Theme } from '@mui/material/styles';
import { appearOnOpenMixin, drawerTransition } from '../drawer/Drawer.styles';

export const getStyles = (theme: Theme, drawerOpen: boolean, level: number) => {
  // Items change shape in step with the drawer's own width transition.
  const drawerTiming = drawerTransition(theme, drawerOpen);
  // A heading inside a group only shows while its group is open, which needs
  // the drawer open, so it never turns into a divider. Keeping it fixed also
  // lets the group measure its full height when it opens.
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
    },
    itemEnd: drawerOpen ? { mr: itemEnd } : {},
    // Expanded, top-level items run flush to the left edge and round off on
    // the right. Collapsed, they fill the rail as square rows, with padding
    // that centres the 24px icon.
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
    // Sits over the item's rounded end like the group chevron, so it reads as
    // part of the item while staying a button of its own: an action or a
    // reorder handle.
    trailingButton: trailingButtonPlace,
    // An action comes and goes with the drawer, so it fades in as it opens.
    itemAction: {
      ...trailingButtonPlace,
      ...appearOnOpenMixin(theme),
    },
    // Fades out at once when collapsing, and in only once the drawer has
    // widened, so it never passes over the icon. Pinned to the item's first
    // row, like the icon, when the label wraps.
    groupToggle: {
      position: 'absolute',
      top: 0,
      right: theme.spacing(itemEnd),
      opacity: drawerOpen ? 1 : 0,
      visibility: drawerOpen ? 'visible' : 'hidden',
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
    listItemDivider: {
      px: drawerOpen ? 2 : 1.5,
      py: 0.5,
    },
    divider: {
      flex: 1,
    },
    listItemButton: {
      px: 2,
    },
    // A heading inside an open group labels a run of its children and inherits
    // the group's indent.
    // Collapsed, the row closes down to a divider with 8px either side of the
    // line.
    sectionRow: {
      // Clips sideways only, so a heading's button can spill out vertically.
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
    // Spans the heading's label line. A 40px icon button centres on it and
    // spills out above and below, into the heading's padding and just over
    // the first entry, so the heading keeps its height. Its right edge lines
    // the button up with the chevrons, actions and handles below.
    headingAccessory: {
      position: 'absolute',
      zIndex: 1,
      top: theme.spacing(sectionPaddingTop),
      bottom: theme.spacing(sectionPaddingBottom),
      right: theme.spacing(itemEnd),
      display: 'flex',
      alignItems: 'center',
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
    // While its list is edited, the entry is something to drag, not a link.
    disabledLink: {
      cursor: 'inherit',
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
