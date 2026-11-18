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
  // Hidden in the collapsed drawer: leaving it to the drawer's edge depends on NavItem's padding
  navItemLabel: {
    [`.${listItemTextClasses.root}`]: {
      opacity: drawerOpen ? 1 : 0,
      transition: theme.transitions.create('opacity', {
        easing: theme.transitions.easing.sharp,
        duration: drawerOpen
          ? theme.transitions.duration.enteringScreen
          : theme.transitions.duration.leavingScreen,
      }),
    },
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
