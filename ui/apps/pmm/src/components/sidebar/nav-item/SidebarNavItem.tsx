import { useLinkWithVariables } from 'hooks/utils/useLinkWithVariables';
import {
  FC,
  KeyboardEvent,
  MouseEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { NavItemProps } from './SidebarNavItem.types';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import {
  getLinkProps,
  hasChildMatch,
  shouldShowBadge,
} from './SidebarNavItem.utils';
import { getStyles } from './SidebarNavItem.styles';
import { DRAWER_WIDTH } from '../drawer/Drawer.constants';
import { drawerTransition } from '../drawer/Drawer.styles';
import { useTheme } from '@mui/material/styles';
import { NavFlyout, NavItem } from '@percona/peak-ui';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import Box from '@mui/material/Box';
import ListItem from '@mui/material/ListItem';
import List from '@mui/material/List';
import Collapse from '@mui/material/Collapse';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import NavItemIcon from './nav-item-icon/NavItemIcon';
import NavItemTooltip from './nav-item-tooltip/NavItemTooltip';
import NavItemBadge from './nav-item-badge/NavItemBadge';
import NavItemAction from './nav-item-action/NavItemAction';

const SidebarNavItem: FC<NavItemProps> = ({
  activeItem,
  item,
  drawerOpen,
  level = 0,
  onClick,
  headingAccessory,
  trailingControl,
  disableLink,
  inFlyout,
}) => {
  const hasActiveChild = useMemo(
    () => hasChildMatch(item, activeItem),
    [activeItem, item]
  );
  const active = activeItem === item || hasActiveChild;
  const [open, setIsOpen] = useState(active);
  // Only one item shows the selected background: the deepest one on the
  // active path that is in view. A group whose active child shows takes just
  // the selected colour; once the child is hidden, the group is selected.
  const childInView = open && drawerOpen;
  const standsInForChild = hasActiveChild && !childInView;
  // A group with a page of its own opens that page; one without opens its
  // first child.
  const url = useLinkWithVariables(item.url ?? item.children?.[0]?.url);
  const linkProps = getLinkProps(item, url);
  const theme = useTheme();
  const styles = getStyles(theme, drawerOpen, level);
  const dataTestid = `navitem-${item.id}`;
  const showBadge = shouldShowBadge(item, open);
  // In the collapsed rail a group cannot open in place, so its entries open
  // in a panel beside it instead.
  const opensFlyout = !drawerOpen && level === 0 && !!item.children?.length;
  const [flyoutAnchor, setFlyoutAnchor] = useState<HTMLElement | null>(null);
  const flyoutId = `${dataTestid}-flyout`;

  useEffect(() => {
    if (active && drawerOpen) {
      setIsOpen(true);
    } else if (level === 0) {
      setIsOpen(false);
    }
  }, [level, drawerOpen, active]);

  useEffect(() => {
    if (drawerOpen) {
      setFlyoutAnchor(null);
    }
  }, [drawerOpen]);

  const handleToggle = useCallback(() => {
    if (!drawerOpen) {
      return;
    }

    setIsOpen((open) => !open);
  }, [drawerOpen]);

  const handleOpenCollapsible = (event: MouseEvent<HTMLElement>) => {
    if (opensFlyout) {
      event.preventDefault();
      setFlyoutAnchor(event.currentTarget);
      return;
    }

    setIsOpen(true);

    if (onClick) {
      onClick();
    }
  };

  // Enter follows the link and so opens the panel through the click above;
  // Space does nothing on a link, so it opens the panel here, as on a button.
  const handleTriggerKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (opensFlyout && event.key === ' ') {
      event.preventDefault();
      setFlyoutAnchor(event.currentTarget);
    }
  };

  const handleFlyoutItemClick = () => {
    setFlyoutAnchor(null);

    if (onClick) {
      onClick();
    }
  };

  const handleItemClick = (event?: MouseEvent) => {
    if (disableLink) {
      event?.preventDefault();
      return;
    }

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
        <Stack
          direction="row"
          alignItems="center"
          sx={styles.groupRow}
          data-testid={dataTestid + '-list-item'}
        >
          <NavItemTooltip
            drawerOpen={drawerOpen || !!flyoutAnchor}
            item={item}
            level={level}
          >
            <NavItem
              text={item.text ?? ''}
              icon={item.icon ? <NavItemIcon icon={item.icon} /> : undefined}
              showDot={showBadge && !!item.icon}
              badge={
                item.badge && item.badgeAlwaysVisible && drawerOpen ? (
                  <NavItemBadge badge={item.badge} />
                ) : undefined
              }
              selected={activeItem === item || standsInForChild}
              hasSelectedChild={hasActiveChild && childInView}
              // Standing in for a hidden child, the group marks the current
              // section rather than the page: its link opens the first child.
              aria-current={
                activeItem === item
                  ? 'page'
                  : standsInForChild
                    ? 'true'
                    : undefined
              }
              sx={[
                styles.navItem,
                styles.itemEnd,
                level === 0 && styles.navItemRoot,
                level === 0 &&
                  !item.badgeAlwaysVisible &&
                  styles.labelWidthEstimate(!!item.icon, true),
                level > 0 && !!item.icon && styles.childIcon,
                styles.navItemWithButton,
              ]}
              {...(linkProps as Omit<typeof linkProps, 'component'> & {
                component?: React.ElementType;
              })}
              {...(opensFlyout && {
                role: 'button',
                'aria-expanded': !!flyoutAnchor,
                'aria-controls': flyoutAnchor ? flyoutId : undefined,
              })}
              onClick={handleOpenCollapsible}
              onKeyDown={handleTriggerKeyDown}
              data-testid={dataTestid}
              data-navlevel={level}
            />
          </NavItemTooltip>
          <IconButton
            data-testid={`${dataTestid}-toggle`}
            onClick={handleToggle}
            aria-label={`${item.text} submenu`}
            aria-expanded={open}
            aria-controls={`${dataTestid}-children`}
            sx={styles.groupToggle}
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
          timeout={drawerTransition(theme, drawerOpen).duration}
          id={`${dataTestid}-children`}
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
                inFlyout={inFlyout}
              />
            ))}
          </List>
        </Collapse>
        {level === 0 && (
          <NavFlyout
            id={flyoutId}
            label={item.text ?? ''}
            // As wide as the group's entries are in the expanded sidebar, less
            // their indent, so labels wrap the same way.
            width={`calc(${DRAWER_WIDTH}px - ${theme.spacing(4.75)})`}
            anchorEl={flyoutAnchor}
            onClose={() => setFlyoutAnchor(null)}
          >
            {item.children.map((child) => (
              <SidebarNavItem
                key={child.id}
                item={child}
                activeItem={activeItem}
                drawerOpen
                level={1}
                onClick={handleFlyoutItemClick}
                inFlyout
              />
            ))}
          </NavFlyout>
        )}
      </>
    );
  }

  if (item.type === 'menu-section') {
    const showHeading = drawerOpen || level > 0;

    // The collapsed rail has no room for a label, so the section reads as a
    // divider. One element serves both states so its height can animate.
    return (
      <ListItem
        data-testid={dataTestid + (showHeading ? '-section' : '-divider')}
        disableGutters
        disablePadding
        sx={[styles.sectionRow, !!inFlyout && styles.flyoutSection]}
      >
        <Box sx={styles.sectionLabel} aria-hidden={!showHeading}>
          <ListItemText primary={item.text} sx={styles.sectionHeading} />
        </Box>
        <Box sx={styles.sectionLine} aria-hidden />
        {headingAccessory && showHeading && (
          <Box sx={styles.headingAccessory}>{headingAccessory}</Box>
        )}
        {item.action && drawerOpen && (
          <NavItemAction action={item.action} testId={`${dataTestid}-action`} />
        )}
      </ListItem>
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
            styles.itemEnd,
            level === 0 && styles.navItemRoot,
            styles.textOnly,
          ]}
        />
      </ListItem>
    );
  }

  return (
    <ListItem
      disablePadding
      data-testid={dataTestid + '-list-item'}
      data-item-id={item.id}
    >
      <NavItemTooltip drawerOpen={drawerOpen} item={item} level={level}>
        <NavItem
          text={item.text ?? ''}
          secondaryText={item.secondaryText}
          icon={item.icon ? <NavItemIcon icon={item.icon} /> : undefined}
          badge={item.badge ? <NavItemBadge badge={item.badge} /> : undefined}
          selected={active}
          sx={[
            styles.navItem,
            styles.itemEnd,
            level === 0 && styles.navItemRoot,
            level === 0 &&
              !item.badge &&
              styles.labelWidthEstimate(
                !!item.icon,
                !!item.action || !!trailingControl
              ),
            level > 0 && !!item.icon && styles.childIcon,
            (!!item.action || !!trailingControl) && styles.navItemWithButton,
            !!disableLink && styles.disabledLink,
          ]}
          {...(linkProps as Omit<typeof linkProps, 'component'> & {
            component?: React.ElementType;
          })}
          onClick={handleItemClick}
          tabIndex={disableLink ? -1 : undefined}
          aria-disabled={disableLink || undefined}
          data-testid={dataTestid}
          data-navlevel={level}
        />
      </NavItemTooltip>
      {item.action && drawerOpen && (
        <NavItemAction
          action={item.action}
          testId={`${dataTestid}-action`}
          sx={inFlyout ? styles.flyoutAction : styles.itemAction}
        />
      )}
      {trailingControl && drawerOpen && (
        <Box sx={styles.trailingButton}>{trailingControl}</Box>
      )}
    </ListItem>
  );
};

export default SidebarNavItem;
