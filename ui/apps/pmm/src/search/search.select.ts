import type {
  DashboardVariablesResult,
  DashboardVariablesMessage,
} from '@pmm/shared';
import {
  GRAFANA_SUB_PATH,
  PMM_BASE_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import messenger from 'lib/messenger';
import { VARIABLES_TIMEOUT_MS } from './search.constants';
import { isRecordable } from './search.recent';
import { canonicalKey } from './search.text';
import type { SearchItem, SelectDetails } from './search.types';

export interface SelectionDeps {
  navigate: (to: string) => void;
  close: () => void;
  closeNavIfNarrow: () => void;
  isFrameLoaded: boolean;
  pathname: string;
  record: (item: SearchItem) => void;
}

// The sidebar's useLinkWithVariables as a plain function: the open dashboard's variables and time range carry over.
export const withDashboardVariables = async (to: string) => {
  try {
    const res = await messenger.sendMessageWithResult<
      DashboardVariablesResult,
      'DASHBOARD_VARIABLES',
      NonNullable<DashboardVariablesMessage['payload']>
    >(
      {
        type: 'DASHBOARD_VARIABLES',
        payload: { url: to.replace(PMM_NEW_NAV_GRAFANA_PATH, '') },
      },
      VARIABLES_TIMEOUT_MS
    );

    return res.url.startsWith(GRAFANA_SUB_PATH)
      ? PMM_NEW_NAV_PATH + res.url
      : PMM_NEW_NAV_GRAFANA_PATH + res.url;
  } catch {
    return to;
  }
};

export const performSelection = async (
  item: SearchItem,
  details: SelectDetails,
  deps: SelectionDeps
) => {
  if (item.kind === 'retry') {
    item.perform?.();
    return;
  }

  deps.close();

  if (item.href) {
    deps.closeNavIfNarrow();

    if (item.target === '_self') {
      item.perform?.();
      window.location.assign(item.href);
    } else {
      window.open(item.href, '_blank', 'noopener,noreferrer');
    }

    return;
  }

  if (!item.to) {
    // Two frames, so it runs after the focus restore that the close schedules for the next one.
    requestAnimationFrame(() => requestAnimationFrame(() => item.perform?.()));
    return;
  }

  deps.closeNavIfNarrow();

  if (isRecordable(item)) {
    deps.record(item);
  }

  if (details.newTab) {
    window.open(PMM_BASE_PATH + item.to, '_blank', 'noopener');
    return;
  }

  if (
    item.carryVariables &&
    deps.isFrameLoaded &&
    canonicalKey(item.to) !== canonicalKey(deps.pathname)
  ) {
    deps.navigate(await withDashboardVariables(item.to));
    return;
  }

  deps.navigate(item.to);
};
