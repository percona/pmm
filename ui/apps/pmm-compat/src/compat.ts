import { locationService } from '@grafana/runtime';
import { CrossFrameMessenger, DashboardVariablesMessage, LocationChangeMessage } from '@pmm/shared';
import { GRAFANA_SUB_PATH, PMM_UI_PATH } from 'lib/constants';
import { applyCustomStyles } from 'styles';
import { changeTheme } from 'theme';
import { adjustToolbar } from 'compat/toolbar';
import { isWithinIframe, getLinkWithVariables } from 'lib/utils';

export const initialize = () => {
  if (!isWithinIframe()) {
    // redirect user to the new UI
    window.location.replace(window.location.href.replace(GRAFANA_SUB_PATH, PMM_UI_PATH));
    return;
  }

  const messenger = new CrossFrameMessenger('GRAFANA').setTargetWindow(window.top!).register();

  messenger.sendMessage({
    type: 'MESSENGER_READY',
  });

  applyCustomStyles();

  adjustToolbar();

  // sync with PMM UI theme
  changeTheme('light');

  messenger.sendMessage({
    type: 'GRAFANA_READY',
  });

  messenger.addListener({
    type: 'LOCATION_CHANGE',
    onMessage: (message: LocationChangeMessage) => locationService.push(message.payload!),
  });

  let prevLocation: Location | undefined;
  locationService.getHistory().listen((location: Location) => {
    // re-add custom toolbar buttons after closing kiosk mode
    if (prevLocation?.search.includes('kiosk') && !location.search.includes('kiosk')) {
      adjustToolbar();
    }

    messenger.sendMessage({
      type: 'LOCATION_CHANGE',
      payload: location,
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
