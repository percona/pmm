import { locationService } from '@grafana/runtime';
import { CrossFrameMessenger, LocationChangeMessage } from '@pmm/shared';
import { applyCustomStyles } from 'styles';
import { isWithinIframe } from 'utils';

export const initialize = () => {
  if (!isWithinIframe()) {
    console.log('pmm-compat', 'not within iframe');

    window.location.replace(window.location.href.replace('/graph', '/pmm-ui/with-nav/graph'));

    return;
  }

  const messenger = new CrossFrameMessenger();

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
    console.log('location', location);

    messenger.sendMessage({
      type: 'LOCATION_CHANGE',
      data: location,
    });
  });
};
