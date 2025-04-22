import { locationService } from '@grafana/runtime';
import { CrossFrameMessenger, DashboardVariablesMessage, LocationChangeMessage } from '@pmm/shared';
import { applyCustomStyles } from 'styles';
import { isWithinIframe } from 'utils';
import { getLinkWithVariables } from 'variables';

export const initialize = () => {
  if (!isWithinIframe()) {
    console.log('pmm-compat', 'not within iframe');

    // redirect user to the new UI
    window.location.replace(window.location.href.replace('/graph', '/pmm-ui/next/graph'));

    return;
  }

  const messenger = new CrossFrameMessenger('GRAFANA');

  messenger.setWindow(window.top!);
  messenger.register();

  applyCustomStyles();

  messenger.sendMessage({
    type: 'MESSENGER_READY',
  });

  messenger.addListener({
    type: 'LOCATION_CHANGE',
    onMessage: (message: LocationChangeMessage) => {
      console.log('GRAFANA', 'LOCATION_CHANGE', message.data);

      locationService.push(message.data!);
    },
  });

  locationService.getHistory().listen((location: Location) => {
    messenger.sendMessage({
      type: 'LOCATION_CHANGE',
      data: location,
    });
  });

  messenger.addListener({
    type: 'DASHBOARD_VARIABLES',
    onMessage: (msg: DashboardVariablesMessage) => {
      if (!msg.data || !msg.data.url) {
        return;
      }

      const url = getLinkWithVariables(msg.data.url);

      messenger.sendMessage({
        id: msg.id,
        type: msg.type,
        data: {
          url: url,
        },
      });
    },
  });
};
