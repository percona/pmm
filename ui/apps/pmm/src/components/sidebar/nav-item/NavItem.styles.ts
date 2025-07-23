import { buttonBaseClasses } from '@mui/material/ButtonBase';
import { listItemIconClasses } from '@mui/material/ListItemIcon';
import { typographyClasses } from '@mui/material/Typography';
import { Theme } from '@mui/material/styles';

export const getStyles = (
  theme: Theme,
  active: boolean,
  drawerOpen: boolean,
  level: number
) => ({
  listItemButton: {
    pl: drawerOpen ? levelPadding[level] || 0 : 0,
    ml: drawerOpen ? levelMargin[level] || 0 : 0,
    borderRadius: drawerOpen ? levelBorderRadius[level] : 0,

    [`.${typographyClasses.root}`]: {
      fontWeight: 600,
    },

    [`&, .${listItemIconClasses.root}`]: {
      color: active ? theme.palette.primary.main : theme.palette.text.primary,
    },

    justifyContent: drawerOpen ? undefined : 'center',
  },
  listItemButtonCollapsible: {
    backgroundColor: active
      ? theme.components?.MuiListItem?.styleOverrides?.selected
      : 'initial',
  },
  listCollapsible:
    level !== 1
      ? {
          ml: 1.5,
          mb: 2,
        }
      : {
          pl: 1,
          ml: 6,
          mb: 2,
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
  1: 2.5,
  3: 3.5,
};

const levelMargin: Record<number, number> = {
  1: 3,
};

const levelBorderRadius: Record<number, any> = {
  0: {
    borderTopRightRadius: 50,
    borderBottomRightRadius: 50,
  },
  1: 50,
  2: 50,
};
