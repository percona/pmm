import { locationService } from '@grafana/runtime';
import messager from 'messenger';
import { LinkVariablesMessage, NavigateToMessage } from 'types';
import { appendCustomStyles, getLinkWithVariables, isWithinIframe, log } from 'utils';

const logMessage = (...msg: any[]) => log('compat', ...msg);

export const initialize = () => {
  if (!isWithinIframe()) {
    logMessage('not within an iframe');
    return;
  }

  logMessage('applying custom styles');
  appendCustomStyles();

  messager.register();

  messager.addListener({
    type: 'NAVIGATE_TO',
    onMessage: (msg: NavigateToMessage) => {
      if (!msg.data.to) {
        return;
      }

      locationService.push(msg.data.to);
    },
  });

  messager.addListener({
    type: 'LINK_VARIABLES',
    onMessage: (msg: LinkVariablesMessage) => {
      if (!msg.data.url) {
        return;
      }

      const urlWithVariables = getLinkWithVariables(msg.data.url);

      messager.sendMessage({
        type: 'LINK_VARIABLES_RESULT',
        data: {
          id: msg.data.id,
          url: urlWithVariables,
        },
      });
    },
  });

  logMessage('settings up location observable');

  locationService.getHistory().listen((location: any) => {
    messager.sendMessage({
      type: 'LOCATION_CHANGE',
      data: {
        location,
      },
    });
  });
};
