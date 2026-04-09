import { useLinkWithVariables } from 'hooks/utils/useLinkWithVariables';
import { FC, useCallback, useEffect, useMemo, useState } from 'react';
import { NavItemProps } from './NavItem.types';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import { getLinkProps, hasChildMatch, shouldShowBadge } from './NavItem.utils';
import { getStyles } from './NavItem.styles';
import { useTheme } from '@mui/material/styles';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import ListItem from '@mui/material/ListItem';
import List from '@mui/material/List';
import Collapse from '@mui/material/Collapse';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import Box from '@mui/material/Box';
import NavItemIcon from './nav-item-icon/NavItemIcon';
import NavItemTooltip from './nav-item-tooltip/NavItemTooltip';
import { DRAWER_WIDTH } from '../drawer/Drawer.constants';
import NavItemDot from './nav-item-dot/NavItemDot';
import NavItemBadge from './nav-item-badge/NavItemBadge';

const NavItem: FC<NavItemProps> = ({
  activeItem,
  item,
  drawerOpen,
  level = 0,
  onClick,
}) => {
  const active = useMemo(
    () => activeItem === item || hasChildMatch(item, activeItem),
    [activeItem, item]
  );
  const [open, setIsOpen] = useState(active);
  const url = useLinkWithVariables(
    item.children?.length ? item.children[0].url : item.url
  );
  const linkProps = getLinkProps(item, url);
  const theme = useTheme();
  const styles = getStyles(theme, active, drawerOpen, level);
  const dataTestid = `navitem-${item.id}`;
  const showBadge = shouldShowBadge(item, open);

  useEffect(() => {
    if (active && drawerOpen) {
      setIsOpen(true);
    } else if (level === 0) {
      setIsOpen(false);
    }
  }, [level, drawerOpen, active]);

  const handleToggle = useCallback(() => {
    if (!drawerOpen) {
      return;
    }

    setIsOpen((open) => !open);
  }, [drawerOpen]);

  const handleOpenCollapsible = () => {
    // prevent opening when sidebar collapsed
    if (drawerOpen) {
      setIsOpen(true);
    }
  };

  if (item.children?.length) {
    return (
      <>
        <NavItemTooltip
          key={item.url}
          drawerOpen={drawerOpen}
          item={item}
          level={level}
        >
          <Stack
            direction="row"
            alignItems="center"
            justifyContent="space-between"
            sx={{
              width: level === 0 ? DRAWER_WIDTH : undefined,
            }}
            data-testid={dataTestid + '-list-item'}
          >
            <ListItemButton
              color="primary.main"
              disableGutters
              sx={[
                styles.listItemButton,
                level === 0 && styles.navItemRootCollapsible,
              ]}
              onClick={handleOpenCollapsible}
              {...linkProps}
              data-testid={dataTestid}
              data-navlevel={level}
            >
              {item.icon && (
                <NavItemDot show={showBadge}>
                  <ListItemIcon sx={styles.listItemIcon}>
                    <NavItemIcon icon={item.icon} />
                  </ListItemIcon>
                </NavItemDot>
              )}
              <ListItemText
                primary={item.text}
                className="navitem-primary-text"
                sx={styles.text}
              />
              {item.badge && item.badgeAlwaysVisible && drawerOpen && (
                <NavItemBadge badge={item.badge} />
              )}
            </ListItemButton>
            {drawerOpen && (
              <IconButton
                data-testid={`${dataTestid}-toggle`}
                onClick={handleToggle}
                sx={{
                  mr: 1,
                }}
              >
                <KeyboardArrowDownIcon
                  sx={(theme) => ({
                    rotate: open ? '180deg' : 0,
                    transition: theme.transitions.create('rotate'),
                  })}
                />
              </IconButton>
            )}
          </Stack>
        </NavItemTooltip>
        <Collapse
          in={open}
          timeout={
            drawerOpen
              ? theme.transitions.duration.leavingScreen
              : theme.transitions.duration.enteringScreen
          }
          data-testid={`${dataTestid}-collapse`}
        >
          <List component="div" disablePadding sx={styles.listCollapsible}>
            {item.children.map((item) => (
              <NavItem
                key={item.id}
                item={item}
                activeItem={activeItem}
                drawerOpen={drawerOpen}
                level={level + 1}
                onClick={onClick}
              />
            ))}
          </List>
        </Collapse>
      </>
    );
  }

  if (item.type === 'menu-divider') {
    return (
      <ListItem
        data-testid={dataTestid + '-divider'}
        sx={styles.listItemDivider}
      >
        <Divider sx={styles.divider} />
      </ListItem>
    );
  }

  if (item.type === 'menu-text') {
    return (
      <ListItem
        key={item.id}
        data-testid={dataTestid + '-text-item'}
        disableGutters
        disablePadding
      >
        <ListItemText
          primary={item.text}
          secondary={item.secondaryText}
          sx={[
            styles.listItemButton,
            styles.leafItem,
            level === 0 && styles.navItemRoot,
            styles.textOnly,
          ]}
        />
      </ListItem>
    );
  }

  return (
    <NavItemTooltip
      key={item.url}
      drawerOpen={drawerOpen}
      item={item}
      level={level}
    >
      <ListItem
        disablePadding
        sx={{
          width: level === 0 ? DRAWER_WIDTH : undefined,
        }}
        data-testid={dataTestid + '-list-item'}
      >
        <ListItemButton
          disableGutters
          sx={[
            styles.listItemButton,
            styles.leafItem,
            level === 0 && styles.navItemRoot,
          ]}
          selected={active}
          {...linkProps}
          onClick={onClick}
          data-testid={dataTestid}
          data-navlevel={level}
        >
          {item.icon ? (
            <ListItemIcon sx={styles.listItemIcon}>
              <NavItemIcon icon={item.icon} />
            </ListItemIcon>
          ) : (
            <Box sx={{ mr: -1 }} />
          )}
          <ListItemText
            primary={item.text}
            secondary={item.secondaryText}
            className="navitem-primary-text"
            sx={styles.text}
          />
          {item.badge && <NavItemBadge badge={item.badge} />}
        </ListItemButton>
      </ListItem>
    </NavItemTooltip>
  );
};

export default NavItem;
