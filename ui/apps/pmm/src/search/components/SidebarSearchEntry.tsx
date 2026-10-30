import ListItem from '@mui/material/ListItem';
import { useTheme } from '@mui/material/styles';
import { SearchTrigger } from '@percona/peak-ui';
import { getSearchShortcut } from '@pmm/shared';
import { useQueryClient } from '@tanstack/react-query';
import { drawerTransition } from 'components/sidebar/drawer/Drawer.styles';
import { getStyles } from 'components/sidebar/nav-item/SidebarNavItem.styles';
import { type FC, useMemo, useRef } from 'react';
import { SIDEBAR_TRIGGER_TEST_ID } from '../search.constants';
import { useSearchExperiment } from '../search.experiment';
import { Messages } from '../search.messages';
import { openSearch } from '../search.store';
import { prefetchDashboardItems } from '../sources/dashboards.source';

interface SidebarSearchEntryProps {
  drawerOpen: boolean;
}

// The list's first entry, shaped like the nav items below it and morphing with the drawer as they do.
const SidebarSearchTrigger: FC<SidebarSearchEntryProps> = ({ drawerOpen }) => {
  const theme = useTheme();
  const ref = useRef<HTMLButtonElement>(null);
  const queryClient = useQueryClient();
  const shortcut = useMemo(() => getSearchShortcut(), []);
  const styles = getStyles(theme, drawerOpen, 0);

  const prefetch = () => {
    void prefetchDashboardItems(queryClient);
  };

  return (
    <ListItem disablePadding>
      <SearchTrigger
        ref={ref}
        collapsed={!drawerOpen}
        label={Messages.trigger}
        shortcut={shortcut}
        morphTransition={drawerTransition(theme, drawerOpen)}
        onClick={() => openSearch('sidebar', { opener: ref.current })}
        onTypeAhead={(text: string) =>
          openSearch('sidebar', { query: text, opener: ref.current })
        }
        onPointerEnter={prefetch}
        onFocus={prefetch}
        sx={[styles.navItem, styles.itemEnd, styles.navItemRoot]}
        data-testid={SIDEBAR_TRIGGER_TEST_ID}
      />
    </ListItem>
  );
};

// The off path calls this one hook and renders nothing.
export const SidebarSearchEntry: FC<SidebarSearchEntryProps> = ({
  drawerOpen,
}) => {
  const enabled = useSearchExperiment();

  return enabled ? <SidebarSearchTrigger drawerOpen={drawerOpen} /> : null;
};
