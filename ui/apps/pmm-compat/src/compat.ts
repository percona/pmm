import { locationService } from '@grafana/runtime';
import { CrossFrameMessenger, DashboardVariablesMessage, LocationChangeMessage } from '@pmm/shared';
import { applyCustomStyles } from 'styles';
import { changeTheme } from 'theme';
import { isWithinIframe } from 'utils';
import { getLinkWithVariables } from 'variables';

export const initialize = () => {
  if (!isWithinIframe()) {
    // redirect user to the new UI
    window.location.replace(window.location.href.replace('/graph', '/pmm-ui/next/graph'));
    return;
  }

  const messenger = new CrossFrameMessenger('GRAFANA').setTargetWindow(window.top!).register();

  messenger.sendMessage({
    type: 'MESSENGER_READY',
  });

  applyCustomStyles();

  // sync with PMM UI theme
  changeTheme('light');

  messenger.sendMessage({
    type: 'GRAFANA_READY',
  });

  messenger.addListener({
    type: 'LOCATION_CHANGE',
    onMessage: (message: LocationChangeMessage) => locationService.push(message.payload!),
  });

  locationService.getHistory().listen((location: Location) =>
    messenger.sendMessage({
      type: 'LOCATION_CHANGE',
      payload: location,
    })
  );

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
