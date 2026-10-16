import { forwardRef } from 'react';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Badge from '@mui/material/Badge';
import Box from '@mui/material/Box';
import { listItemIconClasses } from '@mui/material/ListItemIcon';
import { listItemTextClasses } from '@mui/material/ListItemText';
import type { NavItemProps } from './nav-item.types';

// Row the 24px icon occupies; the label and trailing slot centre on it.
const ICON_ROW_HEIGHT = 24;

const NavItem = forwardRef<HTMLDivElement, NavItemProps>(
  (
    {
      text,
      secondaryText,
      icon,
      badge,
      showDot,
      dotColor = 'warning',
      selected,
      hasSelectedChild,
      sx,
      ...rest
    },
    ref
  ) => (
    <ListItemButton
      ref={ref}
      disableGutters
      selected={selected}
      sx={[
        (theme) => ({
          px: 2,
          py: 1,
          minHeight: 40,
          // The icon and trailing slot hold the first line's row when the label wraps.
          alignItems: 'flex-start',
          borderRadius: `${theme.shape.borderRadiusXl}px`,
          fontFamily: theme.typography.menuText.fontFamily,

          [`.${listItemTextClasses.root}`]: {
            my: 0,
            minHeight: ICON_ROW_HEIGHT,
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
          },

          // Two lines at most; sets its own wrapping so a nowrap parent can't run it under the badge.
          [`.${listItemTextClasses.primary}`]: {
            ...theme.typography.menuText,
            whiteSpace: 'normal',
            overflowWrap: 'anywhere',
            display: '-webkit-box',
            WebkitBoxOrient: 'vertical',
            WebkitLineClamp: 2,
            overflow: 'hidden',
          },

          // A group holding the selected item takes its colour but not its background,
          // so only one item in the navigation ever reads as selected.
          [`&, .${listItemIconClasses.root}`]: {
            color:
              selected || hasSelectedChild
                ? theme.palette.primary.main
                : theme.palette.text.primary,
          },

          // A trigger keeps its hover look while the panel it opens, such as a NavFlyout, is showing.
          '&[aria-expanded="true"]': {
            backgroundColor: theme.palette.primary.hover,

            [`&, .${listItemIconClasses.root}`]: {
              color: theme.palette.primary.main,
            },
          },

          [`.${listItemIconClasses.root} .MuiSvgIcon-root`]: {
            fontSize: theme.typography.pxToRem(24),
          },

          [`.${listItemTextClasses.secondary}`]: {
            ...theme.typography.helperText,
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            color: selected || hasSelectedChild ? 'inherit' : theme.palette.text.secondary,
          },

          [`&:hover .${listItemTextClasses.secondary}`]: {
            color: 'inherit',
          },
        }),
        ...(Array.isArray(sx) ? sx : sx ? [sx] : []),
      ]}
      aria-current={selected ? 'page' : undefined}
      {...rest}
    >
      {icon ? (
        showDot ? (
          <Badge variant="dot" color={dotColor}>
            <ListItemIcon sx={{ minWidth: 'auto' }}>{icon}</ListItemIcon>
          </Badge>
        ) : (
          <ListItemIcon sx={{ minWidth: 'auto' }}>{icon}</ListItemIcon>
        )
      ) : (
        <Box sx={{ mr: -1.75 }} />
      )}

      <ListItemText primary={text} secondary={secondaryText} sx={{ pl: 1 }} />

      {badge && (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            flexShrink: 0,
            minHeight: ICON_ROW_HEIGHT,
            ml: 1,
          }}
        >
          {badge}
        </Box>
      )}
    </ListItemButton>
  )
);

NavItem.displayName = 'NavItem';

export default NavItem;
