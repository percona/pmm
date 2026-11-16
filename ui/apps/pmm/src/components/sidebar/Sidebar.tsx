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
import { LABEL_WIDTH_VAR } from './nav-item/SidebarNavItem.styles';
import Box from '@mui/material/Box';
import List from '@mui/material/List';
import { listItemTextClasses } from '@mui/material/ListItemText';
import useMediaQuery from '@mui/material/useMediaQuery';
import { SidebarScrollButton, useScrollEdges } from './scroll-button';
import { useTheme } from '@mui/material/styles';
import { findActiveNavItem, isActive } from 'utils/navigation.utils';
import { addHomePage } from 'contexts/navigation/navigation.utils';
import { useUser } from 'contexts/user';
import { useLocation } from 'react-router-dom';
import type { NavItem as NavItemType } from 'types/navigation.types';
import { hiddenScrollbarMixin, scrollFadeMixin } from './drawer/Drawer.styles';

const edgeMarker = { height: '1px', listStyle: 'none' };

export const Sidebar: FC = () => {
  const { navTree, navOpen, setNavOpen } = useNavigation();
  const { user } = useUser();
  const [activeItem, setActiveItem] = useState<NavItemType>();
  const [animating, setAnimating] = useState(false);
  const [labelWidths, setLabelWidths] = useState<Record<string, string>>({});
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

  // Labels hold a single line while the drawer moves, however it was opened or
  // closed. Nothing moves under reduced motion, so they are free at once.
  const [prevNavOpen, setPrevNavOpen] = useState(navOpen);
  if (navOpen !== prevNavOpen) {
    setPrevNavOpen(navOpen);
    setAnimating(!reduceMotion);
  }

  const setSidebarOpen = useCallback(
    (open: boolean) => {
      // Recorded only from a settled, open drawer: mid-animation widths would
      // break labels into more lines.
      // Kept by entry rather than on the label, so an entry that remounts
      // while collapsed still opens at its width.
      if (navOpen && !open && !animating && list) {
        const widths: Record<string, string> = {};

        list
          .querySelectorAll<HTMLElement>(`.${listItemTextClasses.root}`)
          .forEach((label) => {
            const testId =
              label.closest<HTMLElement>('[data-navlevel]')?.dataset.testid;

            if (testId) {
              widths[testId] = `${label.getBoundingClientRect().width}px`;
            }
          });
        setLabelWidths(widths);
      }

      setNavOpen(open);
    },
    [navOpen, animating, list, setNavOpen]
  );

  const toggleSidebar = useCallback(
    () => setSidebarOpen(!navOpen),
    [setSidebarOpen, navOpen]
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

    // keep previous item active if there isn't a match
    if (activeItem) {
      setActiveItem(activeItem);
    } else if (
      // The home dashboard has no entry of its own: the logo leads there.
      isActive(addHomePage(user?.preferences), location.pathname)
    ) {
      setActiveItem(undefined);
    }
  }, [navTree, location.pathname, user?.preferences]);

  const handleNavItemClick = useCallback(() => {
    // autoclose sidebar when layout is narrow
    if (isNarrow) {
      setSidebarOpen(false);
    }
  }, [isNarrow, setSidebarOpen]);

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
              width: '100%',
              overflowY: 'auto',
              overflowX: 'hidden',
              pb: 3,
              pt: 0,
            },
            hiddenScrollbarMixin,
            scrollFadeMixin(canScrollUp, canScrollDown),
            // Labels keep their open width while the drawer moves or is collapsed,
            // so they slide out of view rather than rewrap.
            (animating || !navOpen) && {
              [`.${listItemTextClasses.root}`]: {
                flex: `0 0 var(${LABEL_WIDTH_VAR}, auto)`,
              },
              ...Object.fromEntries(
                Object.entries(labelWidths).map(([testId, width]) => [
                  `[data-testid="${testId}"] .${listItemTextClasses.root}`,
                  { [LABEL_WIDTH_VAR]: width },
                ])
              ),
            },
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
