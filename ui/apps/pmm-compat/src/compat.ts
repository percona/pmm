import { locationService, getAppEvents, ThemeChangedEvent, config } from '@grafana/runtime';
import {
  ChangeThemeMessage,
  CrossFrameMessenger,
  DashboardVariablesMessage,
  HistoryAction,
  LocationChangeMessage,
  ColorMode,
} from '@pmm/shared';
import {
  GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY,
  GRAFANA_LOGIN_PATH,
  GRAFANA_SUB_PATH,
  PMM_UI_PATH,
} from 'lib/constants';
import { applyCustomStyles } from 'styles';
import { changeTheme } from 'theme';
import { adjustToolbar } from 'compat/toolbar';
import { isWithinIframe, getLinkWithVariables } from 'lib/utils';
import { documentTitleObserver } from 'lib/utils/document';

export const initialize = () => {
  // If Grafana is opened outside of iframe (or on login), redirect to PMM UI
  if (!isWithinIframe() && !window.location.pathname.startsWith(GRAFANA_LOGIN_PATH)) {
    window.location.replace(window.location.href.replace(GRAFANA_SUB_PATH, PMM_UI_PATH));
    return;
  }

  // Register messenger to communicate with PMM UI (top frame)
  const messenger = new CrossFrameMessenger('GRAFANA').setTargetWindow(window.top!).register();

  // React to PMM → Grafana theme changes
  messenger.addListener({
    type: 'CHANGE_THEME',
    onMessage: (msg: ChangeThemeMessage) => {
      if (!msg.payload) {
        return;
      }
      // Apply theme on Grafana side
      changeTheme(msg.payload.theme);
    },
  });

  messenger.sendMessage({ type: 'MESSENGER_READY' });

  // Ensure docked menu is closed in the iframe
  localStorage.setItem(GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY, 'false');

  applyCustomStyles();
  adjustToolbar();

  // -------- Theme relay: Grafana (right) → PMM UI (left) --------
  // Initial emit from current Grafana runtime config (no parsing, no fallbacks beyond default)
  const initialMode = (config?.theme2?.colors?.mode ?? 'light') as ColorMode;
  messenger.sendMessage({
    type: 'GRAFANA_THEME_CHANGED',
    payload: { theme: initialMode },
  });

  // Forward future ThemeChangedEvent to PMM UI.
  // Known Grafana type: evt.payload.colors.mode is 'light' | 'dark'
  getAppEvents().subscribe(ThemeChangedEvent, (evt: ThemeChangedEvent) => {
    const nextMode = (evt.payload?.colors?.mode ?? 'light') as ColorMode;
    messenger.sendMessage({
      type: 'GRAFANA_THEME_CHANGED',
      payload: { theme: nextMode },
    });
  });
  // --------------------------------------------------------------

  messenger.sendMessage({ type: 'GRAFANA_READY' });

  // PMM → Grafana: location changes
  messenger.addListener({
    type: 'LOCATION_CHANGE',
    onMessage: ({ payload: location }: LocationChangeMessage) => {
      if (!location) {
        return;
      }
      locationService.replace(location);
    },
  });

  // Report current document title once
  messenger.sendMessage({
    type: 'DOCUMENT_TITLE_CHANGE',
    payload: { title: document.title },
  });

  // Observe future title updates
  documentTitleObserver.listen((title) => {
    messenger.sendMessage({
      type: 'DOCUMENT_TITLE_CHANGE',
      payload: { title },
    });
  });

  // Relay Grafana history changes back to PMM
  let prevLocation: Location | undefined;
  locationService.getHistory().listen((location: Location, action: HistoryAction) => {
    // Re-add custom toolbar buttons after exiting kiosk mode
    if (prevLocation?.search.includes('kiosk') && !location.search.includes('kiosk')) {
      adjustToolbar();
    }

    messenger.sendMessage({
      type: 'LOCATION_CHANGE',
      payload: {
        action,
        ...location,
      },
    });

    prevLocation = location;
  });

  // PMM → Grafana: expand dashboard URL with variables and echo back
  messenger.addListener({
    type: 'DASHBOARD_VARIABLES',
    onMessage: (msg: DashboardVariablesMessage) => {
      if (!msg.payload?.url) {
        return;
      }

      const url = getLinkWithVariables(msg.payload.url);

      messenger.sendMessage({
        id: msg.id,
        type: msg.type,
        payload: { url },
      });
    },
  });
};
