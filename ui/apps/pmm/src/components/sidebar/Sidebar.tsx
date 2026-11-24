import {
  type FC,
  type TransitionEvent,
  useCallback,
  useEffect,
  useState,
} from 'react';
import { useNavigation } from 'contexts/navigation';
import { NavigationHeading } from './nav-heading';
import { Drawer } from './drawer';
import { SidebarNavItem } from './nav-item';
import { DRAWER_WIDTH } from './drawer/Drawer.constants';
import Box from '@mui/material/Box';
import List from '@mui/material/List';
import useMediaQuery from '@mui/material/useMediaQuery';
import { SidebarScrollButton, useScrollEdges } from './scroll-button';
import { useTheme } from '@mui/material/styles';
import { findActiveNavItem } from 'utils/navigation.utils';
import { useLocation } from 'react-router-dom';
import type { NavItem as NavItemType } from 'types/navigation.types';
import { hiddenScrollbarMixin, scrollFadeMixin } from './drawer/Drawer.styles';

const edgeMarker = { height: '1px', listStyle: 'none' };

export const Sidebar: FC = () => {
  const { navTree, navOpen, setNavOpen } = useNavigation();
  const [activeItem, setActiveItem] = useState<NavItemType>();
  const [animating, setAnimating] = useState(false);
  const location = useLocation();
  const theme = useTheme();
  const isNarrow = useMediaQuery(theme.breakpoints.down('md'));
  const reduceMotion = useMediaQuery('(prefers-reduced-motion: reduce)');
  const {
    root: list,
    rootRef,
    startRef,
    endRef,
    canScrollUp,
    canScrollDown,
  } = useScrollEdges<HTMLUListElement, HTMLLIElement>();

  // Scrolls by most of a screenful, keeping a little of the previous view as
  // context.
  const scrollList = (direction: 1 | -1) => {
    list?.scrollBy({
      top: direction * list.clientHeight * 0.8,
      behavior: reduceMotion ? 'auto' : 'smooth',
    });
  };

  // Set for as long as the drawer moves, however it was opened or closed.
  // Nothing moves under reduced motion.
  const [prevNavOpen, setPrevNavOpen] = useState(navOpen);
  if (navOpen !== prevNavOpen) {
    setPrevNavOpen(navOpen);
    setAnimating(!reduceMotion);
  }

  const toggleSidebar = useCallback(
    () => setNavOpen(!navOpen),
    [setNavOpen, navOpen]
  );

  // Released when the paper's own width transition ends, not on a timer: the
  // transition starts only once React has re-rendered, which takes a moment.
  const handlePaperTransitionEnd = (event: TransitionEvent<HTMLDivElement>) => {
    if (
      event.target === event.currentTarget &&
      event.propertyName === 'width'
    ) {
      setAnimating(false);
    }
  };

  useEffect(() => {
    const activeItem = findActiveNavItem(navTree, location.pathname);

    // Keeps the previous item without a match, except on a dashboard: one with
    // no entry (any home dashboard included) is no entry's page.
    if (activeItem) {
      setActiveItem(activeItem);
    } else if (location.pathname.includes('/d/')) {
      setActiveItem(undefined);
    }
  }, [navTree, location.pathname]);

  const handleNavItemClick = useCallback(() => {
    // autoclose sidebar when layout is narrow
    if (isNarrow) {
      setNavOpen(false);
    }
  }, [isNarrow, setNavOpen]);

  const renderItem = (item: NavItemType) => (
    <SidebarNavItem
      key={item.id}
      item={item}
      activeItem={activeItem}
      drawerOpen={navOpen}
      onClick={handleNavItemClick}
    />
  );

  return (
    <Drawer
      open={navOpen}
      variant="permanent"
      anchor="left"
      data-testid="pmm-sidebar"
      slotProps={{ paper: { onTransitionEnd: handlePaperTransitionEnd } }}
    >
      <NavigationHeading
        sidebarOpen={navOpen}
        onToggleSidebar={toggleSidebar}
        onNavigate={handleNavItemClick}
      />
      <Box
        sx={{ position: 'relative', display: 'flex', flex: 1, minHeight: 0 }}
      >
        <SidebarScrollButton
          direction="up"
          visible={canScrollUp}
          onClick={() => scrollList(-1)}
        />
        <List
          ref={rootRef}
          disablePadding
          sx={[
            {
              // Laid out at the open width while the drawer moves, so labels
              // slide out of view instead of rewrapping; the paper clips it.
              width: animating ? DRAWER_WIDTH : '100%',
              flexShrink: 0,
              overflowY: 'auto',
              overflowX: 'hidden',
              pb: 3,
              pt: 0,
            },
            hiddenScrollbarMixin,
            scrollFadeMixin(canScrollUp, canScrollDown),
          ]}
        >
          <Box component="li" ref={startRef} aria-hidden sx={edgeMarker} />
          {navTree.map(renderItem)}
          <Box component="li" ref={endRef} aria-hidden sx={edgeMarker} />
        </List>
        <SidebarScrollButton
          direction="down"
          visible={canScrollDown}
          onClick={() => scrollList(1)}
        />
      </Box>
    </Drawer>
  );
};
