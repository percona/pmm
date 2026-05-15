import { forwardRef } from 'react';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Badge from '@mui/material/Badge';
import Box from '@mui/material/Box';
import { listItemIconClasses } from '@mui/material/ListItemIcon';
import { listItemTextClasses } from '@mui/material/ListItemText';
import type { NavItemProps } from './nav-item.types';

const NavItem = forwardRef<HTMLDivElement, NavItemProps>(
  (
    { text, secondaryText, icon, badge, showDot, dotColor = 'warning', selected, sx, ...rest },
    ref
  ) => (
    <ListItemButton
      ref={ref}
      disableGutters
      selected={selected}
      sx={[
        (theme) => ({
          px: 2,
          py: 0.75,
          minHeight: 44,
          borderRadius: 5.5,
          fontFamily: theme.typography.menuText.fontFamily,

          [`.${listItemTextClasses.primary}`]: {
            fontWeight: theme.typography.menuText.fontWeight,
            fontFamily: 'inherit',
          },

          [`&, .${listItemIconClasses.root}`]: {
            color: selected ? theme.palette.primary.main : theme.palette.text.primary,
          },

          [`.${listItemTextClasses.secondary}`]: {
            ...theme.typography.helperText,
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            color: selected ? 'inherit' : theme.palette.text.secondary,
          },

          [`&:hover .${listItemTextClasses.secondary}`]: {
            color: 'inherit',
          },
        }),
        ...(Array.isArray(sx) ? sx : sx ? [sx] : []),
      ]}
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

      <ListItemText primary={text} secondary={secondaryText} sx={{ pl: 2 }} />

      {badge}
    </ListItemButton>
  )
);

NavItem.displayName = 'NavItem';

export default NavItem;
