import {
  buttonBaseClasses,
  listItemIconClasses,
  Theme,
  typographyClasses,
} from '@mui/material';

export const getStyles = (
  theme: Theme,
  active: boolean,
  drawerOpen: boolean,
  level: number
) => ({
  listItemButton: {
    pl: drawerOpen ? levelPadding[level] || 0 : 0,
    borderRadius: drawerOpen ? 50 : 0,

    [`.${typographyClasses.root}`]: {
      fontWeight: 600,
    },

    [`&, .${listItemIconClasses.root}`]: {
      color: active ? theme.palette.primary.main : theme.palette.text.primary,
    },

    [`&:hover, &:hover .${listItemIconClasses.root} svg`]: {
      color: 'primary.main',
    },

    justifyContent: drawerOpen ? undefined : 'center',
  },
  listItemButtonCollapsible: {
    backgroundColor: 'initial',
  },
  listCollapsible:
    level !== 1
      ? { ml: 1.5 }
      : {
          pl: 1,
          ml: 6,
          borderLeft: 1,
          borderColor: theme.palette.divider,

          [`.${buttonBaseClasses.root}`]: {
            px: 2.5,
          },
        },
  listItemIcon: {
    minWidth: 'auto',
    pr: drawerOpen ? 1 : 0,
  },
  listItemDivider: drawerOpen
    ? {}
    : {
        justifyContent: 'center',
      },
  divider: drawerOpen
    ? {
        mr: 1,
        ml: 2,
        flex: 1,
      }
    : {
        flex: 1,
      },
  text: {
    whiteSpace: 'normal',
  },
});

const levelPadding: Record<number, number> = {
  0: 3,
  1: 5.5,
  3: 3.5,
};
