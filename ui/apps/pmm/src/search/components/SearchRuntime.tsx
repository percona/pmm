import { isRenderingServer } from '@pmm/shared';
import { useAuth } from 'contexts/auth';
import { useGrafana } from 'contexts/grafana';
import { useUser } from 'contexts/user';
import { type FC, useEffect, useRef } from 'react';
import { connectSearchBridge } from '../search.bridge';
import {
  GRAFANA_IFRAME_SELECTOR,
  SIDEBAR_TRIGGER_TEST_ID,
} from '../search.constants';
import { isOtherModalOpen } from '../search.dom';
import {
  getSearchState,
  openSearch,
  resetSearchStore,
  takeSearchOpener,
  useSearchState,
} from '../search.store';
import { useRecentVisits } from '../useRecentVisits';
import { useSearchIndex } from '../useSearchIndex';
import { useSearchShortcut } from '../useSearchShortcut';
import { SearchPanel } from './SearchPanel';

export const SearchRuntimeGate: FC = () => {
  const { isLoggedIn } = useAuth();
  const { user } = useUser();

  if (!(isLoggedIn || user?.isAnonymous) || isRenderingServer()) {
    return null;
  }

  return <SearchRuntime />;
};

// The dialog restores no focus itself: focus that came from the Grafana frame has no record in the focus trap.
const useFocusReturn = (open: boolean) => {
  const wasOpen = useRef(open);

  useEffect(() => {
    if (wasOpen.current && !open) {
      const opener = takeSearchOpener();

      requestAnimationFrame(() => {
        if (!opener || getSearchState().open) {
          return;
        }

        if (opener.isConnected) {
          opener.focus({ preventScroll: true });
        }

        // The opener is gone or hidden, such as the Grafana frame after picking a PMM page.
        if (document.activeElement !== opener) {
          document
            .querySelector<HTMLElement>(
              `[data-testid="${SIDEBAR_TRIGGER_TEST_ID}"]`
            )
            ?.focus({ preventScroll: true });
        }
      });
    }

    wasOpen.current = open;
  }, [open]);
};

export const SearchRuntime: FC = () => {
  const { open } = useSearchState();
  const index = useSearchIndex(open);
  const { frameRef } = useGrafana();

  useSearchShortcut();
  useRecentVisits(index.staticItems);
  useFocusReturn(open);

  useEffect(
    () =>
      connectSearchBridge({
        getFrameWindow: () =>
          frameRef?.current?.contentWindow ??
          document.querySelector<HTMLIFrameElement>(GRAFANA_IFRAME_SELECTOR)
            ?.contentWindow ??
          null,
        onOpen: () => {
          if (!isOtherModalOpen()) {
            openSearch('grafana');
          }
        },
      }),
    [frameRef]
  );

  useEffect(() => () => resetSearchStore(), []);

  return <SearchPanel index={index} />;
};
