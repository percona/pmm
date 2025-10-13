import { locationService, getAppEvents, ThemeChangedEvent, config } from '@grafana/runtime';
import {
  ChangeThemeMessage,
  CrossFrameMessenger,
  DashboardVariablesMessage,
  HistoryAction,
  LocationChangeMessage,
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
  if (!isWithinIframe() && !window.location.pathname.startsWith(GRAFANA_LOGIN_PATH)) {
    // redirect user to the new UI
    window.location.replace(window.location.href.replace(GRAFANA_SUB_PATH, PMM_UI_PATH));
    return;
  }

  const messenger = new CrossFrameMessenger('GRAFANA').setTargetWindow(window.top!).register();

  messenger.addListener({
    type: 'CHANGE_THEME',
    onMessage: (msg: ChangeThemeMessage) => {
      if (!msg.payload) {
        return;
      }

      changeTheme(msg.payload.theme);

      // try {
      //   const onProfile = /\/profile$/.test(window.location.pathname || '');
      //   if (onProfile) {
      //     const FLAG = '__pmm_theme_iframe_reloaded__';
      //     const now = Date.now();
      //     const last = Number(sessionStorage.getItem(FLAG) || 0);
      //
      //     if (!last || now - last > 1500) {
      //       sessionStorage.setItem(FLAG, String(now));
      //       // eslint-disable-next-line no-console
      //       console.log('[pmm-compat] Reloading iframe to refresh Preferences dropdown');
      //       window.location.reload();
      //     } else {
      //       // eslint-disable-next-line no-console
      //       console.log('[pmm-compat] Skip reload (recently reloaded)');
      //     }
      //   }
      // } catch (err) {
      //   // eslint-disable-next-line no-console
      //   console.warn('[pmm-compat] Failed to reload iframe after theme change:', err);
      // }
    },
  });

  messenger.sendMessage({
    type: 'MESSENGER_READY',
  });

  // set docked state to false
  localStorage.setItem(GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY, 'false');

  applyCustomStyles();

  adjustToolbar();

  // Wire Grafana theme events to Percona scheme attribute and notify parent
  setupThemeWiring();

  messenger.sendMessage({
    type: 'GRAFANA_READY',
  });

  messenger.addListener({
    type: 'LOCATION_CHANGE',
    onMessage: ({ payload: location }: LocationChangeMessage) => {
      if (!location) {
        return;
      }
      locationService.replace(location);
    },
  });

  messenger.sendMessage({
    type: 'DOCUMENT_TITLE_CHANGE',
    payload: { title: document.title },
  });
  documentTitleObserver.listen((title) => {
    messenger.sendMessage({
      type: 'DOCUMENT_TITLE_CHANGE',
      payload: { title },
    });
  });

  let prevLocation: Location | undefined;
  locationService.getHistory().listen((location: Location, action: HistoryAction) => {
    // re-add custom toolbar buttons after closing kiosk mode
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

  messenger.addListener({
    type: 'DASHBOARD_VARIABLES',
    onMessage: (msg: DashboardVariablesMessage) => {
      if (!msg.payload || !msg.payload.url) {
        return;
      }

      const url = getLinkWithVariables(msg.payload.url);

      messenger.sendMessage({
        id: msg.id,
        type: msg.type,
        payload: {
          url: url,
        },
      });
    },
  });
};

/**
 * Wires Grafana theme changes (ThemeChangedEvent) to Percona CSS scheme.
 * Ensures <html> has the attribute our CSS reads and informs the parent frame.
 */
function setupThemeWiring() {
  // Helper to apply our scheme attribute and notify parent
  const apply = (mode: 'light' | 'dark') => {
    const html = document.documentElement;
    const scheme = mode === 'dark' ? 'percona-dark' : 'percona-light';

    // Set the attribute expected by our CSS tokens
    html.setAttribute('data-md-color-scheme', scheme);

    // Optional helpers for broader compatibility
    html.setAttribute('data-theme', mode);
    (html.style as any).colorScheme = mode;

    // Notify outer PMM UI (new nav) to sync immediately
    try {
      const target = (window.top && window.top !== window) ? window.top : (window.parent || window);
      target?.postMessage({ type: 'grafana.theme.changed', payload: { mode } }, window.location.origin);
    } catch {
      // ignore cross-origin or other unexpected errors
    }
  };

  // Initial apply from current Grafana theme config
  const initialMode = (config?.theme2?.colors?.mode === 'dark' ? 'dark' : 'light') as 'light' | 'dark';
  apply(initialMode);

  // React to Grafana theme changes (Preferences save or changeTheme())
  getAppEvents().subscribe(ThemeChangedEvent, (evt: any) => {
    const mode: 'light' | 'dark' = evt?.payload?.colors?.mode === 'dark' ? 'dark' : 'light';
    apply(mode);
  });
}
