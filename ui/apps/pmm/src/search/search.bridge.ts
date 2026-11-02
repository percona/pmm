import {
  CrossFrameMessenger,
  type OpenSearchMessage,
  type SearchConfigPayload,
} from '@pmm/shared';

export interface SearchBridgeOptions {
  getFrameWindow: () => Window | null;
  onOpen: () => void;
}

// Its own messenger instance, so GrafanaProvider's messenger.unregister() never drops these listeners.
export const connectSearchBridge = ({
  getFrameWindow,
  onOpen,
}: SearchBridgeOptions) => {
  const bridge = new CrossFrameMessenger('PMM')
    .setTargetOrigin(window.location.origin)
    .register();

  const send = (enabled: boolean) => {
    const frameWindow = getFrameWindow();

    if (!frameWindow) {
      return;
    }

    try {
      bridge
        .setTargetWindow(frameWindow)
        .sendMessage<'SEARCH_CONFIG', SearchConfigPayload>({
          type: 'SEARCH_CONFIG',
          payload: { enabled },
        });
    } catch {
      // A frame that is going away cannot take the message; GRAFANA_READY resends it.
    }
  };

  bridge.addListener<'OPEN_SEARCH', undefined>({
    type: 'OPEN_SEARCH',
    onMessage: (message: OpenSearchMessage) => {
      if (message.source === 'GRAFANA') {
        onOpen();
      }
    },
  });

  bridge.addListener<'GRAFANA_READY', undefined>({
    type: 'GRAFANA_READY',
    onMessage: () => send(true),
  });

  send(true);

  return () => {
    send(false);
    bridge.unregister();
  };
};
