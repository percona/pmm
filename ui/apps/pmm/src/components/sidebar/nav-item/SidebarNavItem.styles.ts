import { listItemTextClasses } from '@mui/material/ListItemText';
import { Theme } from '@mui/material';

export const getStyles = (
  theme: Theme,
  drawerOpen: boolean,
  level: number
) => ({
  navItemRoot: {
    borderRadius: 0,
  },
  leafItem: drawerOpen
    ? {
        mr: level > 0 ? 1 : 0,
      }
    : {},
  navItemRootCollapsible: {
    borderTopLeftRadius: 0,
    borderBottomLeftRadius: 0,
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
    px: drawerOpen ? 2 : 1,
  },
  divider: {
    flex: 1,
  },
  listItemButton: {
    px: 2,
  },
  // A heading inside an open group labels a run of its children; it sits
  // tighter than a top-level section and inherits the group's indent.
  sectionRow:
    level === 0
      ? {
          pl: 2,
          pr: 1,
          pt: 3,
          pb: 0.5,
          gap: 0.5,
        }
      : {
          pl: 0,
          pr: 1,
          pt: 1.5,
          pb: 0.5,
          gap: 0.5,
        },
  sectionHeading: {
    m: 0,
    flex: 'none',

    [`.${listItemTextClasses.primary}`]: {
      ...theme.typography.overline,
      color: theme.palette.text.secondary,
    },
  },
  pinHandle: {
    color: theme.palette.text.secondary,
    cursor: 'grab',
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
});
