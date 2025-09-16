import { listItemIconClasses } from '@mui/material/ListItemIcon';
import { listItemTextClasses } from '@mui/material/ListItemText';
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
  leafItem: drawerOpen
    ? {
        mr: level > 0 ? 1 : 0,
      }
    : {
        py: 1.5,
      },
  navItemRootCollapsible: {
    borderTopLeftRadius: 0,
    borderBottomLeftRadius: 0,
  },
  listItemButton: {
    px: 2,
    height: 48,
    borderRadius: 50,
    justifyContent: drawerOpen ? undefined : 'center',

    [`.${typographyClasses.root}`]: {
      fontWeight: 600,
    },

    [`&, .${listItemIconClasses.root}`]: {
      color: active ? theme.palette.primary.main : theme.palette.text.primary,
    },
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
  },
  listItemDivider: {
    px: drawerOpen ? 2 : 1,
  },
  divider: {
    flex: 1,
  },
  text: {
    pl: 2,

    [`.${listItemTextClasses.secondary}`]: {
      ...theme.typography.helperText,
      textOverflow: 'ellipsis',
      whiteSpace: 'nowrap',
      overflow: 'hidden',
    },
  },
});
