import { locationService, getAppEvents, ThemeChangedEvent, config } from '@grafana/runtime';
import {
  ChangeThemeMessage,
  CrossFrameMessenger,
  DashboardVariablesMessage,
  HistoryAction,
  LocationChangeMessage,
  ColorMode,
  isRenderingServer,
} from '@pmm/shared';
import {
  GRAFANA_DOCKED_LOCAL_STORAGE_KEY,
  GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY,
  GRAFANA_LOGIN_PATH,
  GRAFANA_SUB_PATH,
  PMM_UI_GRAFANA_PATH,
  PMM_UI_HELP_PATH,
  PMM_UI_PATH,
} from 'lib/constants';
import { applyCustomStyles } from 'styles';
import { changeTheme } from 'theme';
import { adjustToolbar } from 'compat/toolbar';
import { isWithinIframe, getLinkWithVariables } from 'lib/utils';
import { documentTitleObserver, updateBodyClassByLocation } from 'lib/utils/document';
import { isFirstLogin, updateIsFirstLogin, isUserLoggedIn } from 'lib/utils/login';
import {
  ServiceAddedEvent,
  ServiceDeletedEvent,
  SettingsUpdatedEvent,
  FrontendSettingsUpdatedEvent,
  TimeZoneUpdatedEvent,
} from 'lib/events';
import { handleExternalLinks } from 'compat/links';

export const initialize = () => {
  // Image renderer (headless Chrome) loads the panel URL directly. Skip all compat logic so the dashboard renders normally.
  if (isRenderingServer()) {
    return;
  }

  // If Grafana is opened outside of iframe (or on login), redirect to PMM UI
  if (!isWithinIframe() && !window.location.pathname.startsWith(GRAFANA_LOGIN_PATH)) {
    const isHomePath =
      window.location.pathname === GRAFANA_SUB_PATH || window.location.pathname === `${GRAFANA_SUB_PATH}/`;

    // upon first login redirect user to the help page with welcome modal
    if (isFirstLogin() && isHomePath) {
      updateIsFirstLogin();

      window.location.replace(isUserLoggedIn() ? PMM_UI_HELP_PATH : PMM_UI_PATH);
    } else {
      // redirect user to the new UI
      window.location.replace(window.location.href.replace(GRAFANA_SUB_PATH, PMM_UI_GRAFANA_PATH));
    }
    return;
  }

  // Collapse Grafana docked nav via localStorage before the shell reads it on boot. If keys were
  // missing or not "false" (e.g. after "clear site data"), Grafana may have mounted nav already;
  // reload once so the next boot sees the correct keys. When both are already "false", skip reload.
  const prevOpen = localStorage.getItem(GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY);
  const prevDock = localStorage.getItem(GRAFANA_DOCKED_LOCAL_STORAGE_KEY);
  const needsNavReload = prevOpen !== 'false' || prevDock !== 'false';
  localStorage.setItem(GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY, 'false');
  localStorage.setItem(GRAFANA_DOCKED_LOCAL_STORAGE_KEY, 'false');
  if (needsNavReload) {
    window.location.reload();
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

  updateBodyClassByLocation(window.location);
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

    // Update body class for custom page styles
    updateBodyClassByLocation(location);
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

  messenger.addListener({
    type: 'SETTINGS_CHANGED',
    onMessage: () => getAppEvents().publish(new SettingsUpdatedEvent()),
  });

  getAppEvents().subscribe(SettingsUpdatedEvent, () => {
    messenger.sendMessage({
      type: 'SETTINGS_CHANGED',
    });
  });

  getAppEvents().subscribe(FrontendSettingsUpdatedEvent, () => {
    messenger.sendMessage({
      type: 'FRONTEND_SETTINGS_CHANGED',
    });
    window.location.reload();
  });

  getAppEvents().subscribe(ServiceAddedEvent, () => {
    messenger.sendMessage({
      type: 'SERVICE_ADDED',
    });
  });

  getAppEvents().subscribe(ServiceDeletedEvent, () => {
    messenger.sendMessage({
      type: 'SERVICE_DELETED',
    });
  });

  getAppEvents().subscribe(TimeZoneUpdatedEvent, () => {
    messenger.sendMessage({
      type: 'TIMEZONE_CHANGED',
    });
  });

  handleExternalLinks();
};
