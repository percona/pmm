import messenger from 'lib/messenger';
import { connectSearchBridge } from './search.bridge';

const fromGrafana = (type: string, source = 'GRAFANA') =>
  window.dispatchEvent(new MessageEvent('message', { data: { type, source } }));

const setup = (withFrame = true) => {
  const postMessage = vi.fn();
  const frame = withFrame ? ({ postMessage } as unknown as Window) : null;
  const onOpen = vi.fn();
  const disconnect = connectSearchBridge({
    getFrameWindow: () => frame,
    onOpen,
  });

  return { onOpen, disconnect, postMessage };
};

const searchConfig = (enabled: boolean) =>
  expect.objectContaining({
    type: 'SEARCH_CONFIG',
    source: 'PMM',
    payload: { enabled },
  });

describe('connectSearchBridge', () => {
  it('turns search on in the frame as soon as it connects', () => {
    const { postMessage, disconnect } = setup();

    expect(postMessage).toHaveBeenCalledTimes(1);
    expect(postMessage).toHaveBeenCalledWith(
      searchConfig(true),
      window.location.origin
    );

    disconnect();
  });

  it('answers every GRAFANA_READY again', () => {
    const { postMessage, disconnect } = setup();

    fromGrafana('GRAFANA_READY');
    fromGrafana('GRAFANA_READY');

    expect(postMessage).toHaveBeenCalledTimes(3);
    expect(postMessage).toHaveBeenLastCalledWith(
      searchConfig(true),
      window.location.origin
    );

    disconnect();
  });

  it('opens search only for OPEN_SEARCH from Grafana', () => {
    const { onOpen, disconnect } = setup();

    fromGrafana('OPEN_SEARCH', 'SOMEONE_ELSE');
    expect(onOpen).not.toHaveBeenCalled();

    fromGrafana('OPEN_SEARCH');
    expect(onOpen).toHaveBeenCalledTimes(1);

    disconnect();
  });

  it('keeps listening after the shared messenger is unregistered', () => {
    const { onOpen, disconnect } = setup();

    messenger.register();
    messenger.unregister();
    fromGrafana('OPEN_SEARCH');

    expect(onOpen).toHaveBeenCalledTimes(1);

    disconnect();
  });

  it('turns search off in the frame and stops listening on disconnect', () => {
    const { onOpen, postMessage, disconnect } = setup();

    disconnect();

    expect(postMessage).toHaveBeenLastCalledWith(
      searchConfig(false),
      window.location.origin
    );

    fromGrafana('OPEN_SEARCH');
    fromGrafana('GRAFANA_READY');

    expect(onOpen).not.toHaveBeenCalled();
    expect(postMessage).toHaveBeenCalledTimes(2);
  });

  it('does nothing without a frame', () => {
    const { onOpen, disconnect } = setup(false);

    expect(() => fromGrafana('GRAFANA_READY')).not.toThrow();

    fromGrafana('OPEN_SEARCH');
    expect(onOpen).toHaveBeenCalledTimes(1);

    expect(() => disconnect()).not.toThrow();
  });
});
