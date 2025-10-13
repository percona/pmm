import { locationService } from '@grafana/runtime';
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
    },
  });

  messenger.sendMessage({
    type: 'MESSENGER_READY',
  });

  // set docked state to false
  localStorage.setItem(GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY, 'false');

  applyCustomStyles();

  adjustToolbar();

  // sync with PMM UI theme
  changeTheme('light');

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
