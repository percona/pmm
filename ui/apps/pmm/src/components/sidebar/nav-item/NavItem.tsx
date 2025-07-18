import { useLinkWithVariables } from 'hooks/utils/useLinkWithVariables';
import { isActive } from 'lib/utils/navigation.utils';
import { FC, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
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

  useEffect(() => {
    if (active) {
      setIsOpen(true);
    }
  }, [active]);

  if (children?.length && drawerOpen) {
    return (
      <>
        <ListItemButton
          color="primary.main"
          disableGutters
          sx={[styles.listItemButton, styles.listItemButtonCollapsible]}
          onClick={() => setIsOpen(!open)}
          data-testid={dataTestid}
          data-navlevel={level}
        >
          {item.icon && (
            <ListItemIcon sx={styles.listItemIcon}>
              <NavItemIcon icon={item.icon} />
            </ListItemIcon>
          )}
          <ListItemText
            primary={item.text}
            primaryTypographyProps={{ style: styles.text }}
          />
          <Stack pl={2} pr={2}>
            <KeyboardArrowDownIcon
              sx={(theme) => ({
                rotate: open ? '180deg' : 0,
                transition: theme.transitions.create('rotate'),
              })}
            />
          </Stack>
        </ListItemButton>
        <Collapse
          in={open}
          timeout="auto"
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
    <ListItem key={item.url} disablePadding>
      <ListItemButton
        disableGutters
        sx={styles.listItemButton}
        selected={active}
        {...linkProps}
        data-testid={dataTestid}
        data-navlevel={level}
      >
        {item.icon && (
          <ListItemIcon sx={styles.listItemIcon}>
            <NavItemIcon icon={item.icon} />
          </ListItemIcon>
        )}
        {drawerOpen && (
          <ListItemText
            primary={item.text}
            primaryTypographyProps={{ style: styles.text }}
          />
        )}
      </ListItemButton>
    </ListItem>
  );
};

export default NavItem;
