import { useLinkWithVariables } from 'hooks/utils/useLinkWithVariables';
import { isActive } from 'lib/utils/navigation.utils';
import { FC, useCallback, useState } from 'react';
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

  const handleToggle = useCallback(() => {
    setIsOpen((open) => !open);
  }, []);

  const handleOpenCollapsible = () => {
    const firstChild = (item.children || [])[0];
    setIsOpen(true);

    if (firstChild?.url) {
      navigate(firstChild.url);
    }
  };

  if (children?.length && drawerOpen) {
    return (
      <>
        <Stack
          direction="row"
          alignItems="center"
          justifyContent="space-between"
        >
          <ListItemButton
            color="primary.main"
            disableGutters
            sx={[
              styles.listItemButton,
              level === 0 && styles.navItemRootCollapsible,
            ]}
            onClick={handleOpenCollapsible}
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
              primaryTypographyProps={level > 0 ? { style: styles.text } : {}}
            />
          </ListItemButton>
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
        </Stack>
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
    <NavItemTooltip key={item.url} drawerOpen={drawerOpen} item={item}>
      <ListItem disablePadding>
        <ListItemButton
          disableGutters
          sx={[styles.listItemButton, level === 0 && styles.navItemRoot]}
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
              primaryTypographyProps={level > 0 ? { style: styles.text } : {}}
            />
          )}
        </ListItemButton>
      </ListItem>
    </NavItemTooltip>
  );
};

export default NavItem;
