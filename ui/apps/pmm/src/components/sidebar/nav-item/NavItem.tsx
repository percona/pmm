import { useLinkWithVariables } from 'hooks/utils/useLinkWithVariables';
import { isActive } from 'lib/utils/navigation.utils';
import { FC, useCallback, useEffect, useState } from 'react';
import type { ComponentProps } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { NavItemProps } from './NavItem.types';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import { getLinkProps } from './NavItem.utils';
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
import NavItemIcon from './nav-item-icon/NavItemIcon';
import IconButton from '@mui/material/IconButton';
import NavItemTooltip from './nav-item-tooltip/NavItemTooltip';
import { DRAWER_WIDTH } from '../drawer/Drawer.constants';
import { useSetTheme } from 'themes/setTheme';

const NavItem: FC<NavItemProps> = ({ item, drawerOpen, level = 0 }) => {
  const location = useLocation();
  const active = isActive(item, location.pathname);
  const [open, setIsOpen] = useState(active);
  const url = useLinkWithVariables(item.url);
  const linkProps = getLinkProps(item, url);
  const theme = useTheme();
  const styles = getStyles(theme, active, drawerOpen, level);
  const children = item.children?.filter((i) => !i.hidden);
  const dataTestid = `navitem-${item.id}`;
  const navigate = useNavigate();

  // Detect "theme-toggle" item and compute dynamic label/icon from current palette mode.
  const isThemeToggle = item.id === 'theme-toggle';
  const paletteMode = (theme.palette?.mode ?? 'light') as 'light' | 'dark';
  const themeToggleLabel =
    paletteMode === 'dark' ? 'Change to Light Theme' : 'Change to Dark Theme';
  const themeToggleIcon = paletteMode === 'dark' ? 'theme-light' : 'theme-dark';

  // Use the exact prop type from NavItemIcon to keep the union type.
  type IconName = NonNullable<ComponentProps<typeof NavItemIcon>['icon']>;

  // Resolve icon for this item; make sure it's never undefined when passed down.
  const resolvedIcon: IconName | undefined = isThemeToggle
    ? (themeToggleIcon as IconName)
    : (item.icon as IconName | undefined);

  const { setTheme } = useSetTheme();

  // Handle click for the action item: instant UI update + persist + iframe sync.
  const handleThemeToggleClick = useCallback(async () => {
    try {
      const next: 'light' | 'dark' = paletteMode === 'dark' ? 'light' : 'dark';
      await setTheme(next);
    } catch (err) {
      console.warn('[NavItem] Theme toggle failed:', err);
    }
  }, [paletteMode, setTheme]);

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
    const firstChild = (item.children || [])[0];

    // prevent opening when sidebar collapsed
    if (drawerOpen) {
      setIsOpen(true);
    }

    if (firstChild?.url) {
      navigate(firstChild.url);
    }
  };

  if (children?.length) {
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
            sx={{ width: level === 0 ? DRAWER_WIDTH : undefined }}
          >
            <ListItemButton
              color="primary.main"
              disableGutters
              sx={[styles.listItemButton, level === 0 && styles.navItemRootCollapsible]}
              onClick={handleOpenCollapsible}
              data-testid={dataTestid}
              data-navlevel={level}
            >
              {item.icon && (
                <ListItemIcon sx={styles.listItemIcon}>
                  <NavItemIcon icon={item.icon as IconName} />
                </ListItemIcon>
              )}
              <ListItemText
                primary={item.text}
                className="navitem-primary-text"
                sx={styles.text}
              />
            </ListItemButton>
            {drawerOpen && (
              <IconButton
                data-testid={`${dataTestid}-toggle`}
                onClick={handleToggle}
                sx={{ mr: 1 }}
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
            {children.map((item) => (
              <NavItem
                key={item.id}
                item={item}
                drawerOpen={drawerOpen}
                level={level + 1}
              />
            ))}
          </List>
        </Collapse>
      </>
    );
  }

  if (item.isDivider) {
    return (
      <ListItem sx={styles.listItemDivider}>
        <Divider sx={styles.divider} />
      </ListItem>
    );
  }

  return (
    <NavItemTooltip
      key={item.url || item.id /* ensure stable key for action items */}
      drawerOpen={drawerOpen}
      item={item}
      level={level}
    >
      <ListItem disablePadding sx={{ width: level === 0 ? DRAWER_WIDTH : undefined }}>
        <ListItemButton
          disableGutters
          sx={[styles.listItemButton, styles.leafItem, level === 0 && styles.navItemRoot]}
          // Action items must not be highlighted as "selected"
          selected={!isThemeToggle && active}
          // Links keep linkProps; action item uses explicit onClick
          {...(!isThemeToggle ? linkProps : {})}
          onClick={isThemeToggle ? handleThemeToggleClick : linkProps?.onClick}
          data-testid={dataTestid}
          data-navlevel={level}
        >
          {resolvedIcon && (
            <ListItemIcon sx={styles.listItemIcon}>
              <NavItemIcon icon={resolvedIcon} />
            </ListItemIcon>
          )}
          <ListItemText
            primary={isThemeToggle ? themeToggleLabel : item.text}
            className="navitem-primary-text"
            sx={styles.text}
          />
        </ListItemButton>
      </ListItem>
    </NavItemTooltip>
  );
};

export default NavItem;
