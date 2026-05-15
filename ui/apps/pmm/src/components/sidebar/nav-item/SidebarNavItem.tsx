import { useLinkWithVariables } from 'hooks/utils/useLinkWithVariables';
import { FC, useCallback, useEffect, useMemo, useState } from 'react';
import { NavItemProps } from './SidebarNavItem.types';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import { getLinkProps, hasChildMatch, shouldShowBadge } from './SidebarNavItem.utils';
import { getStyles } from './SidebarNavItem.styles';
import { useTheme } from '@mui/material/styles';
import { NavItem } from '@percona/percona-ui';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import ListItem from '@mui/material/ListItem';
import List from '@mui/material/List';
import Collapse from '@mui/material/Collapse';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import NavItemIcon from './nav-item-icon/NavItemIcon';
import NavItemTooltip from './nav-item-tooltip/NavItemTooltip';
import { DRAWER_WIDTH } from '../drawer/Drawer.constants';
import NavItemBadge from './nav-item-badge/NavItemBadge';

const SidebarNavItem: FC<NavItemProps> = ({
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
  const styles = getStyles(theme, drawerOpen, level);
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

    if (onClick) {
      onClick();
    }
  };

  const handleItemClick = () => {
    if (linkProps.onClick) {
      linkProps.onClick();
    }

    if (onClick) {
      onClick();
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
              width: level === 0 ? DRAWER_WIDTH : '100%',
            }}
            data-testid={dataTestid + '-list-item'}
          >
            <NavItem
              text={item.text ?? ''}
              icon={item.icon ? <NavItemIcon icon={item.icon} /> : undefined}
              showDot={showBadge && !!item.icon}
              badge={
                item.badge && item.badgeAlwaysVisible && drawerOpen
                  ? <NavItemBadge badge={item.badge} />
                  : undefined
              }
              selected={active}
              sx={[
                level === 0 && styles.navItemRootCollapsible,
                !drawerOpen && { justifyContent: 'center' },
              ]}
              {...(linkProps as Omit<typeof linkProps, 'component'> & { component?: React.ElementType })}
              onClick={handleOpenCollapsible}
              data-testid={dataTestid}
              data-navlevel={level}
            />
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
              <SidebarNavItem
                key={item.id}
                item={item}
                activeItem={activeItem}
                drawerOpen={drawerOpen}
                level={level + 1}
                onClick={handleItemClick}
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
          width: level === 0 ? DRAWER_WIDTH : '100%',
        }}
        data-testid={dataTestid + '-list-item'}
      >
        <NavItem
          text={item.text ?? ''}
          secondaryText={item.secondaryText}
          icon={item.icon ? <NavItemIcon icon={item.icon} /> : undefined}
          badge={item.badge ? <NavItemBadge badge={item.badge} /> : undefined}
          selected={active}
          sx={[
            styles.leafItem,
            level === 0 && styles.navItemRoot,
            !drawerOpen && { justifyContent: 'center' },
          ]}
          {...(linkProps as Omit<typeof linkProps, 'component'> & { component?: React.ElementType })}
          onClick={handleItemClick}
          data-testid={dataTestid}
          data-navlevel={level}
        />
      </ListItem>
    </NavItemTooltip>
  );
};

export default SidebarNavItem;
