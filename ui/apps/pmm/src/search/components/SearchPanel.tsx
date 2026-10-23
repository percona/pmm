import AddBoxOutlined from '@mui/icons-material/AddBoxOutlined';
import ArticleOutlined from '@mui/icons-material/ArticleOutlined';
import DashboardOutlined from '@mui/icons-material/DashboardOutlined';
import DnsOutlined from '@mui/icons-material/DnsOutlined';
import FolderOutlined from '@mui/icons-material/FolderOutlined';
import HelpOutlineOutlined from '@mui/icons-material/HelpOutlineOutlined';
import ManageSearchOutlined from '@mui/icons-material/ManageSearchOutlined';
import OpenInNewOutlined from '@mui/icons-material/OpenInNewOutlined';
import RefreshOutlined from '@mui/icons-material/RefreshOutlined';
import SettingsOutlined from '@mui/icons-material/SettingsOutlined';
import StorageOutlined from '@mui/icons-material/StorageOutlined';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { SearchDialog, getMatchRanges } from '@percona/peak-ui';
import { getModifierKeyCap, isApplePlatform } from '@pmm/shared';
import NavItemIcon from 'components/sidebar/nav-item/nav-item-icon/NavItemIcon';
import { useGrafana } from 'contexts/grafana';
import { useNavigation } from 'contexts/navigation';
import { useUser } from 'contexts/user';
import { type FC, type ReactNode, useMemo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import type { NavItem } from 'types/navigation.types';
import {
  REMOTE_MIN_QUERY_LENGTH,
  SEARCH_DIALOG_TEST_ID,
  SEARCH_INPUT_TEST_ID,
} from '../search.constants';
import { Messages } from '../search.messages';
import {
  buildSearchGroups,
  buildZeroQueryGroups,
  prepareItems,
} from '../search.rank';
import {
  getRecentKey,
  pushRecent,
  readRecent,
  resolveRecent,
} from '../search.recent';
import { performSelection } from '../search.select';
import { closeSearch, setSearchQuery, useSearchState } from '../search.store';
import { tokenize } from '../search.text';
import type {
  RecentEntry,
  SearchIndex,
  SearchItem,
  SearchItemKind,
  SelectDetails,
} from '../search.types';

// A Peak UI SearchDialogItem that still carries everything selection needs.
interface PanelItem extends SearchItem {
  icon: ReactNode;
  meta?: ReactNode;
  labelRanges?: readonly (readonly [number, number])[];
}

interface PanelGroup {
  id: string;
  label: string;
  items: PanelItem[];
}

const KIND_ICON: Record<SearchItemKind, NonNullable<NavItem['icon']>> = {
  page: ArticleOutlined,
  dashboard: DashboardOutlined,
  folder: FolderOutlined,
  setting: SettingsOutlined,
  // Most actions create or add something; one that doesn't brings its own icon.
  action: AddBoxOutlined,
  help: HelpOutlineOutlined,
  service: StorageOutlined,
  node: DnsOutlined,
  link: ManageSearchOutlined,
  retry: RefreshOutlined,
};

// Snapshot taken when the dialog opens, so Recent holds still while it fades out.
interface RecentSnapshot {
  open: boolean;
  entries: RecentEntry[];
  pathname: string;
}

export const SearchPanel: FC<{ index: SearchIndex }> = ({ index }) => {
  const { open, query } = useSearchState();
  const theme = useTheme();
  const isXs = useMediaQuery(theme.breakpoints.down('sm'));
  const isNarrow = useMediaQuery(theme.breakpoints.down('md'));
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const { user } = useUser();
  const { isFrameLoaded } = useGrafana();
  const { navOpen, setNavOpen } = useNavigation();
  const recentKey = user ? getRecentKey(user) : undefined;
  const [recent, setRecent] = useState<RecentSnapshot>({
    open: false,
    entries: [],
    pathname,
  });

  if (recent.open !== open) {
    setRecent(
      open
        ? {
            open,
            entries: recentKey ? readRecent(recentKey) : [],
            pathname,
          }
        : { ...recent, open }
    );
  }

  const tokens = useMemo(() => tokenize(query), [query]);
  const prepared = useMemo(() => prepareItems(index.items), [index.items]);
  const recentItems = useMemo(
    () => resolveRecent(recent.entries, index.items, recent.pathname),
    [recent, index.items]
  );

  const groups = useMemo<PanelGroup[]>(() => {
    const highlightQuery = tokens.join(' ');
    const resultGroups = tokens.length
      ? buildSearchGroups(prepared, query, {
          recentIds: recent.entries.map((entry) => entry.id),
          dashboards: index.dashboards,
          inventory: index.inventory,
        })
      : buildZeroQueryGroups(index.items, recentItems);

    return resultGroups.map((group) => ({
      id: group.id,
      label: group.label,
      items: group.items.map((item) => ({
        ...item,
        icon: <NavItemIcon icon={item.navIcon ?? KIND_ICON[item.kind]} />,
        meta: item.external ? (
          <OpenInNewOutlined fontSize="small" />
        ) : undefined,
        labelRanges: highlightQuery
          ? getMatchRanges(item.label, highlightQuery)
          : undefined,
      })),
    }));
  }, [
    tokens,
    prepared,
    query,
    recent.entries,
    recentItems,
    index.items,
    index.dashboards,
    index.inventory,
  ]);

  const keyHints = useMemo(
    () => [
      { keys: ['↑', '↓'], label: Messages.keyHints.navigate },
      { keys: ['↵'], label: Messages.keyHints.open },
      {
        keys: [getModifierKeyCap(), '↵'],
        label: Messages.keyHints.openInNewTab,
      },
      { keys: ['Esc'], label: Messages.keyHints.close },
    ],
    []
  );

  const loading =
    tokens.join(' ').length >= REMOTE_MIN_QUERY_LENGTH &&
    (index.dashboards.status === 'loading' ||
      index.inventory.status === 'loading');

  const handleSelect = (item: PanelItem, details: SelectDetails) => {
    void performSelection(item, details, {
      navigate: (to) => navigate(to),
      close: closeSearch,
      closeNavIfNarrow: () => {
        if (isNarrow && navOpen) {
          setNavOpen(false);
        }
      },
      isFrameLoaded,
      pathname,
      record: (selected) => {
        if (recentKey) {
          pushRecent(recentKey, selected);
        }
      },
    });
  };

  return (
    <SearchDialog
      open={open}
      onClose={() => closeSearch()}
      query={query}
      onQueryChange={(next: string) => setSearchQuery(next)}
      groups={groups}
      onSelect={handleSelect}
      loading={loading}
      ctrlNavigation={isApplePlatform()}
      placeholder={isXs ? Messages.placeholderNarrow : Messages.placeholder}
      labels={{
        dialog: Messages.dialog,
        noResultsHint: Messages.noResultsHint,
      }}
      keyHints={keyHints}
      slotProps={{
        dialog: {
          disableRestoreFocus: true,
          'data-testid': SEARCH_DIALOG_TEST_ID,
        },
        input: { 'data-testid': SEARCH_INPUT_TEST_ID },
      }}
    />
  );
};
