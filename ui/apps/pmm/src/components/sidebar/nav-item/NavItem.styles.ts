import { listItemIconClasses } from '@mui/material/ListItemIcon';
import { typographyClasses } from '@mui/material/Typography';
import { Theme } from '@mui/material/styles';

export const getStyles = (
  theme: Theme,
  active: boolean,
  drawerOpen: boolean,
  level: number
) => ({
  navItemRoot: {
    borderRadius: 0,
  },
  navItemRootCollapsible: {
    borderRadius: drawerOpen ? levelBorderRadius[level] : 0,
  },
  listItemButton: {
    px: 2,

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
    level === 0
      ? {
          pl: 4,
          pb: 2,
        }
      : level === 1
        ? {
            ml: 2.5,
            pl: '11px',
            pb: 2,
            borderLeft: 1,
            borderColor: theme.palette.divider,
          }
        : {},
  listItemIcon: {
    minWidth: 'auto',
    pr: drawerOpen ? 1 : 0,
  },
  listItemDivider: {
    px: 2,
  },
  divider: {
    flex: 1,
  },
  text: {
    whiteSpace: 'normal',
  },
});

const levelBorderRadius: Record<number, any> = {
  0: {
    borderTopRightRadius: 50,
    borderBottomRightRadius: 50,
  },
  1: 50,
  2: 50,
};
